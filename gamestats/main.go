package gamestats

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"wwfc/common"
	"wwfc/database"
	"wwfc/gpcm"
	"wwfc/logging"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/logrusorgru/aurora/v3"
)

type GameStatsSession struct {
	Conn       net.Conn
	ModuleName string
	Challenge  string

	SessionKey int32
	GameInfo   *common.GameInfo

	Authenticated bool
	LoginID       int
	User          database.User

	WriteBuffer []byte
}

var (
	ctx  = context.Background()
	pool *pgxpool.Pool

	serverName string
	webSalt    string
	webHashPad string
)

func StartServer() {
	// Get config
	config := common.GetConfig()

	serverName = config.ServerName
	webSalt = common.RandomString(32)
	webHashPad = common.RandomString(8)

	common.ReadGameList()

	// Start SQL
	dbString := fmt.Sprintf("postgres://%s:%s@%s/%s", config.Username, config.Password, config.DatabaseAddress, config.DatabaseName)
	dbConf, err := pgxpool.ParseConfig(dbString)
	if err != nil {
		panic(err)
	}

	pool, err = pgxpool.ConnectConfig(ctx, dbConf)
	if err != nil {
		panic(err)
	}

	address := *config.GameSpyAddress + ":29920"

	l, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}

	go func() {
		// Close the listener when the application closes.
		defer l.Close()
		logging.Notice("GSTATS", "Listening on", address)

		for {
			// Listen for an incoming connection.
			conn, err := l.Accept()
			if err != nil {
				panic(err)
			}

			// Handle connections in a new goroutine.
			go handleRequest(conn)
		}
	}()
}

// Handles incoming requests.
func handleRequest(conn net.Conn) {
	session := GameStatsSession{
		Conn:       conn,
		ModuleName: "GSTATS:" + conn.RemoteAddr().String(),
		Challenge:  common.RandomString(10),

		WriteBuffer: []byte{},
	}

	defer conn.Close()

	err := conn.(*net.TCPConn).SetKeepAlive(true)
	if err != nil {
		logging.Notice(session.ModuleName, "Unable to set keepalive:", err.Error())
	}

	// Send challenge
	session.Write(common.GameSpyCommand{
		Command:      "lc",
		CommandValue: "1",
		OtherValues: map[string]string{
			"challenge": session.Challenge,
			"id":        "1",
		},
	})
	conn.Write(session.WriteBuffer)
	session.WriteBuffer = []byte{}

	logging.Notice(session.ModuleName, "Connection established from", conn.RemoteAddr())

	// Here we go into the listening loop
	for {
		buffer := make([]byte, 0x4000)
		bufferSize := 0
		message := ""

		// Packets can be received in fragments, so this loop makes sure the packet has been fully received before continuing
		for {
			if bufferSize >= len(buffer) {
				logging.Error(session.ModuleName, "Buffer overflow")
				return
			}

			readSize, err := bufio.NewReader(conn).Read(buffer[bufferSize:])
			if err != nil {
				if errors.Is(err, io.EOF) {
					logging.Info(session.ModuleName, "Connection closed")
					return
				}

				logging.Error(session.ModuleName, "Connection error:", err.Error())
				return
			}

			bufferSize += readSize

			if !bytes.Contains(buffer[max(0, bufferSize-readSize-6):bufferSize], []byte(`\final\`)) {
				continue
			}

			// Decrypt the data
			decrypted := ""
			for i := 0; i < bufferSize; i++ {
				if i+7 <= bufferSize && bytes.Equal(buffer[i:i+7], []byte(`\final\`)) {
					// Append the decrypted content to the message
					message += decrypted + `\final\`
					decrypted = ""

					// Remove the processed data
					buffer = buffer[i+7:]
					bufferSize -= i + 7
					i = 0

					if bufferSize < 7 || !bytes.Contains(buffer[:bufferSize], []byte(`\final\`)) {
						break
					}
					continue
				}

				decrypted += string(rune(buffer[i] ^ "GameSpy3D"[i%9]))
			}

			// Continue to processing the message if we have a full message and another message is not expected
			if len(message) > 0 && bufferSize <= 0 {
				break
			}
		}

		commands, err := common.ParseGameSpyMessage(message)
		if err != nil {
			logging.Error(session.ModuleName, "Error parsing message:", err.Error())
			logging.Error(session.ModuleName, "Raw data:", message)
			session.replyError(gpcm.ErrParse)
			return
		}

		commands = session.handleCommand("ka", commands, func(command common.GameSpyCommand) {
			session.Write(common.GameSpyCommand{
				Command: "ka",
			})
		})

		commands = session.handleCommand("auth", commands, session.auth)
		commands = session.handleCommand("authp", commands, session.authp)

		if len(commands) != 0 && !session.Authenticated {
			logging.Error(session.ModuleName, "Attempt to run command before authentication:", aurora.Cyan(commands[0]))
			session.replyError(gpcm.ErrNotLoggedIn)
			return
		}

		commands = session.handleCommand("getpd", commands, session.getpd)
		commands = session.handleCommand("setpd", commands, session.setpd)
		common.UNUSED(session.ignoreCommand)

		for _, command := range commands {
			logging.Error(session.ModuleName, "Unknown command:", aurora.Cyan(command))
		}

		if len(session.WriteBuffer) > 0 {
			conn.Write(session.WriteBuffer)
			session.WriteBuffer = []byte{}
		}
	}
}

func (g *GameStatsSession) handleCommand(name string, commands []common.GameSpyCommand, handler func(command common.GameSpyCommand)) []common.GameSpyCommand {
	var unhandled []common.GameSpyCommand

	for _, command := range commands {
		if command.Command != name {
			unhandled = append(unhandled, command)
			continue
		}

		logging.Info(g.ModuleName, "Command:", aurora.Yellow(command.Command))
		handler(command)
	}

	return unhandled
}

func (g *GameStatsSession) ignoreCommand(name string, commands []common.GameSpyCommand) []common.GameSpyCommand {
	var unhandled []common.GameSpyCommand

	for _, command := range commands {
		if command.Command != name {
			unhandled = append(unhandled, command)
		}
	}

	return unhandled
}

func (g *GameStatsSession) Write(command common.GameSpyCommand) {
	// Encrypt the data and append it to be sent
	payload := []byte(common.CreateGameSpyMessage(command))
	// Exclude trailing \final\
	for i := 0; i < len(payload)-7; i++ {
		payload[i] ^= "GameSpy3D"[i%9]
	}
	g.WriteBuffer = append(g.WriteBuffer, payload...)
}
