package matchmaking

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
	"wwfc/common"
	"wwfc/logging"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/logrusorgru/aurora/v3"
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

func checkError(err error) {
	if err != nil {
		log.Fatalf("GCSP server has encountered a fatal error! Reason: %v\n", err)
	}
}

func StartServer() {
	// Get config
	config := common.GetConfig()

	// Start SQL
	dbString := fmt.Sprintf("postgres://%s:%s@%s/%s", config.Username, config.Password, config.DatabaseAddress, config.DatabaseName)
	dbConf, err := pgxpool.ParseConfig(dbString)
	checkError(err)
	pool, err = pgxpool.ConnectConfig(ctx, dbConf)
	checkError(err)

	l, err := net.Listen("tcp", "127.0.0.1:28910")
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	fmt.Println("Listening on " + "127.0.0.1:28910")
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

	// log.Printf("%s: Connection established from %s. Sending challenge.", aurora.Green("[NOTICE]"), aurora.Yellow(conn.RemoteAddr()))
	// conn.Write([]byte(fmt.Sprintf(`\lc\1\challenge\%s\id\1\final\`, challenge)))

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
			serverList(conn, buffer)
			break

		default:
			logging.Notice(ModuleName, "Command:", aurora.Yellow(buffer[2]).String())
			break
		}
	}
}
