package gcsp

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
	userId int64
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

	l, err := net.Listen("tcp", "127.0.0.1:29901")
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	fmt.Println("Listening on " + "127.0.0.1:29901")
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
			log.Fatal(err)
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
