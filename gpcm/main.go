package gpcm

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/logrusorgru/aurora/v3"
	"io"
	"net"
	"os"
	"sync"
	"wwfc/common"
	"wwfc/database"
	"wwfc/logging"
)

type GameSpySession struct {
	Conn       net.Conn
	User       database.User
	ModuleName string
	LoggedIn   bool
	Challenge  string
	Status     string
	LocString  string
	FriendList []uint32
}

var (
	ctx  = context.Background()
	pool *pgxpool.Pool
	Salt []byte
	// I would use a sync.Map instead of the map mutex combo, but this performs better.
	sessions = map[uint32]*GameSpySession{}
	mutex    = sync.RWMutex{}
)

func StartServer() {
	// Get config
	config := common.GetConfig()

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

	Salt, err = os.ReadFile("salt.bin")
	if err != nil {
		panic(err)
	}

	address := config.Address + ":29900"
	l, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}

	// Close the listener when the application closes.
	defer l.Close()
	logging.Notice("GPCM", "Listening on", address)

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

func (g *GameSpySession) closeSession() {
	if g.LoggedIn {
		g.sendLogoutStatus()
	}

	mutex.Lock()
	defer mutex.Unlock()

	g.Conn.Close()
	if g.LoggedIn {
		delete(sessions, g.User.ProfileId)
	}
}

// Handles incoming requests.
func handleRequest(conn net.Conn) {
	session := GameSpySession{
		Conn:       conn,
		User:       database.User{},
		ModuleName: "GPCM",
		LoggedIn:   false,
		Challenge:  "",
		Status:     "",
		LocString:  "",
		FriendList: []uint32{},
	}

	defer session.closeSession()

	// Set session ID and challenge
	session.Challenge = common.RandomString(10)

	err := conn.(*net.TCPConn).SetKeepAlive(true)
	if err != nil {
		logging.Notice(session.ModuleName, "Unable to set keepalive (1):", err.Error())
	}

	payload := common.CreateGameSpyMessage(common.GameSpyCommand{
		Command:      "lc",
		CommandValue: "1",
		OtherValues: map[string]string{
			"challenge": session.Challenge,
			"id":        "1",
		},
	})
	conn.Write([]byte(payload))

	logging.Notice(session.ModuleName, "Connection established from", conn.RemoteAddr())

	// Here we go into the listening loop
	for {
		buffer := make([]byte, 1024)
		_, err := bufio.NewReader(conn).Read(buffer)
		if err != nil {
			if errors.Is(err, io.EOF) {
				// Client closed connection, terminate.
				logging.Notice(session.ModuleName, "Client closed connection")
				return
			}

			logging.Error(session.ModuleName, "Connection lost")
			return
		}

		commands, err := common.ParseGameSpyMessage(string(buffer))
		if err != nil {
			logging.Error(session.ModuleName, "Error parsing message:", err.Error())
			logging.Error(session.ModuleName, "Raw data:", string(buffer))
			return
		}

		// Commands must be handled in a certain order, not in the order supplied by the client

		commands = session.handleCommand("ka", commands, func(command common.GameSpyCommand) {
			session.Conn.Write([]byte(`\ka\\final\`))
		})
		commands = session.handleCommand("login", commands, session.login)
		commands = session.ignoreCommand("logout", commands)

		if len(commands) != 0 && session.LoggedIn == false {
			logging.Error(session.ModuleName, "Attempt to run command before login!")
			return
		}

		commands = session.handleCommand("updatepro", commands, session.updateProfile)
		commands = session.handleCommand("status", commands, session.setStatus)
		commands = session.handleCommand("addbuddy", commands, session.addFriend)
		commands = session.handleCommand("delbuddy", commands, session.removeFriend)
		commands = session.handleCommand("authadd", commands, session.authAddFriend)
		commands = session.handleCommand("bm", commands, session.bestieMessage)
		commands = session.handleCommand("getprofile", commands, session.getProfile)

		for _, command := range commands {
			logging.Error(session.ModuleName, "Unknown command:", aurora.Cyan(command.Command))
		}
	}
}

func (g *GameSpySession) handleCommand(name string, commands []common.GameSpyCommand, handler func(command common.GameSpyCommand)) []common.GameSpyCommand {
	unhandled := []common.GameSpyCommand{}

	for _, command := range commands {
		if command.Command != name {
			unhandled = append(unhandled, command)
			continue
		}

		logging.Notice(g.ModuleName, "Command:", aurora.Yellow(command.Command))
		handler(command)
	}

	return unhandled
}

func (g *GameSpySession) ignoreCommand(name string, commands []common.GameSpyCommand) []common.GameSpyCommand {
	unhandled := []common.GameSpyCommand{}

	for _, command := range commands {
		if command.Command != name {
			unhandled = append(unhandled, command)
		}
	}

	return unhandled
}
