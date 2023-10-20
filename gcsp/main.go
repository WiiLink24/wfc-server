package gcsp

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
	"wwfc/logging"
)

var (
	ctx    = context.Background()
	pool   *pgxpool.Pool
	userId int64
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

	address := config.Address + ":29901"
	l, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}

	// Close the listener when the application closes.
	defer l.Close()
	logging.Notice("GCSP", "Listening on", address)

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
	defer conn.Close()

	err := conn.(*net.TCPConn).SetKeepAlive(true)
	if err != nil {
		logging.Notice("GCSP", "Unable to set keepalive:", err.Error())
	}

	err = conn.(*net.TCPConn).SetKeepAlivePeriod(time.Hour * 1000)
	if err != nil {
		logging.Notice("GCSP", "Unable to set keepalive:", err.Error())
	}

	logging.Notice("GCSP", "Connection established from", conn.RemoteAddr().String())

	// Here we go into the listening loop
	for {
		buffer := make([]byte, 1024)
		_, err := bufio.NewReader(conn).Read(buffer)
		if err != nil {
			if errors.Is(err, io.EOF) {
				// Client closed connection, terminate.
				return
			}
		}

		commands, err := common.ParseGameSpyMessage(string(buffer))
		if err != nil {
			panic(err)
		}

		for _, command := range commands {
			logging.Notice("GCSP", "Command:", aurora.Yellow(command.Command).String())
			switch command.Command {
			case "ka":
				conn.Write([]byte(`\ka\\final\`))
				break
			case "otherslist":
				// This message needs to be sorted so we can't use a regular map
				payload := `\otherslist\\oldone\\o\0\uniquenick\7me4ijr5sRMCJ3d9uhvh\final\`
				conn.Write([]byte(payload))
				break
			}
		}
	}
}
