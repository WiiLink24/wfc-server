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
	"time"
	"wwfc/common"
	"wwfc/database"
	"wwfc/logging"
)

type GameSpySession struct {
	User       database.User
	ModuleName string
	LoggedIn   bool
}

var (
	ctx  = context.Background()
	pool *pgxpool.Pool
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

// Handles incoming requests.
func handleRequest(conn net.Conn) {
	session := GameSpySession{
		User:       database.User{},
		ModuleName: "GPCM",
		LoggedIn:   false,
	}

	defer conn.Close()

	// Set session ID and challenge
	challenge := common.RandomString(10)

	err := conn.(*net.TCPConn).SetKeepAlive(true)
	if err != nil {
		logging.Notice(session.ModuleName, "Unable to set keepalive:", err.Error())
	}

	err = conn.(*net.TCPConn).SetKeepAlivePeriod(time.Hour * 1000)
	if err != nil {
		logging.Notice(session.ModuleName, "Unable to set keepalive:", err.Error())
	}

	conn.Write([]byte(fmt.Sprintf(`\lc\1\challenge\%s\id\1\final\`, challenge)))

	logging.Notice(session.ModuleName, "Connection established from", conn.RemoteAddr().String())

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

			logging.Notice(session.ModuleName, "Connection lost")
			return
		}

		commands, err := common.ParseGameSpyMessage(string(buffer))
		if err != nil {
			logging.Notice(session.ModuleName, "Error parsing message:", err.Error())
			logging.Notice(session.ModuleName, "Raw data:", string(buffer))
			return
		}

		for _, command := range commands {
			logging.Notice(session.ModuleName, "Command:", aurora.Yellow(command.Command).String())

			if session.LoggedIn == false {
				if command.Command != "login" {
					logging.Notice(session.ModuleName, "Attempt to run command before login!!!")
					return
				}

				payload, _ := Login(&session, pool, ctx, command, challenge)
				conn.Write([]byte(payload))
			}
		}

		// Make sure commands that update the profile run before getprofile
		for _, command := range commands {
			switch command.Command {
			case "login":
				// User should already be authenticated
				break

			case "logout":
				// Bye
				return

			case "updatepro":
				UpdateProfile(&session, pool, ctx, command)
				break

			case "status":
				logging.Notice(session.ModuleName, "statstring:", aurora.Cyan(command.OtherValues["statstring"]).String())
				if command.OtherValues["locstring"] == "" {
					logging.Notice(session.ModuleName, "locstring: (empty)")
				} else {
					logging.Notice(session.ModuleName, "locstring:", aurora.Cyan(command.OtherValues["locstring"]).String())
				}
				break

			case "addbuddy":
				AddFriend(&session, pool, ctx, command)
				break

			case "delbuddy":
				RemoveFriend(&session, pool, ctx, command)
				break
			}
		}

		for _, command := range commands {
			switch command.Command {
			case "ka":
				conn.Write([]byte(`\ka\\final\`))
				break

			case "getprofile":
				payload := GetProfile(&session, pool, ctx, command)
				conn.Write([]byte(payload))
				break
			}
		}
	}
}
