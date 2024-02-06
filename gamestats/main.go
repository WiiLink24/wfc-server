package gamestats

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
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

	moduleName := "GSTATS"

	err := conn.(*net.TCPConn).SetKeepAlive(true)
	if err != nil {
		logging.Notice(moduleName, "Unable to set keepalive:", err.Error())
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

	// Here we go into the listening loop
	for {
		// TODO: Handle split packets
		buffer := make([]byte, 1024)
		n, err := bufio.NewReader(conn).Read(buffer)
		if err != nil {
			return
		}

		// Decrypt the data
		for i := 0; i < n; i++ {
			if i+7 <= n && bytes.Equal(buffer[i:i+7], []byte(`\final\`)) {
				i += 6
				continue
			}

			buffer[i] ^= "GameSpy3D"[i%9]
		}

		commands, err := common.ParseGameSpyMessage(string(buffer[:n]))
		if err != nil {
			logging.Error(moduleName, "Error parsing message:", err.Error())
			logging.Error(moduleName, "Raw data:", string(buffer[:n]))
			session.replyError(gpcm.ErrParse)
			return
		}

		commands = session.handleCommand("ka", commands, func(command common.GameSpyCommand) {
			session.Conn.Write([]byte(`\ka\\final\`))
		})

		commands = session.handleCommand("auth", commands, session.auth)
		commands = session.handleCommand("authp", commands, session.authp)

		if len(commands) != 0 && session.Authenticated == false {
			logging.Error(session.ModuleName, "Attempt to run command before authentication:", aurora.Cyan(commands[0]))
			session.replyError(gpcm.ErrNotLoggedIn)
			return
		}

		commands = session.handleCommand("setpd", commands, session.setpd)

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
