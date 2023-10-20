package matchmaking

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
	"time"
	"wwfc/common"
	"wwfc/logging"
)

var (
	ctx    = context.Background()
	pool   *pgxpool.Pool
	userId int
)

const (
	ServerList = iota
	ModuleName = "MATCHMAKING"
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

	address := config.Address + ":28910"
	l, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}

	// Close the listener when the application closes.
	defer l.Close()
	logging.Notice("MATCHMAKING", "Listening on", address)

	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
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
		fmt.Printf("Unable to set keepalive - %s", err)
	}

	err = conn.(*net.TCPConn).SetKeepAlivePeriod(time.Hour * 1000)
	if err != nil {
		fmt.Printf("Unable to set keepalive - %s", err)
	}

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

		switch buffer[2] {
		case ServerList:
			logging.Notice(ModuleName, "Command:", aurora.Yellow("SERVER_LIST").String())

			serverList(conn, buffer)
			break

		default:
			logging.Notice(ModuleName, "Unknown command:", aurora.Yellow(buffer[2]).String())
			break
		}
	}
}
