package gpcm

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/logrusorgru/aurora/v3"
	"io"
	"log"
	"net"
	"os"
	"time"
	"wwfc/common"
)

var (
	ctx  = context.Background()
	pool *pgxpool.Pool
)

func checkError(err error) {
	if err != nil {
		log.Fatalf("GPCM server has encountered a fatal error! Reason: %v\n", err)
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

	l, err := net.Listen("tcp", "127.0.0.1:29900")
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	fmt.Println("Listening on " + "127.0.0.1:29900")
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

	// Set session ID and challenge
	challenge := common.RandomString(10)

	err := conn.(*net.TCPConn).SetKeepAlive(true)
	if err != nil {
		fmt.Printf("Unable to set keepalive - %s", err)
	}

	err = conn.(*net.TCPConn).SetKeepAlivePeriod(time.Hour * 1000)
	if err != nil {
		fmt.Printf("Unable to set keepalive - %s", err)
	}

	log.Printf("%s: Connection established from %s. Sending challenge.", aurora.Green("[NOTICE]"), aurora.Yellow(conn.RemoteAddr()))
	conn.Write([]byte(fmt.Sprintf(`\lc\1\challenge\%s\id\1\final\`, challenge)))

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

		command, err := common.ParseGameSpyMessage(string(buffer))
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("%s: Message received. Command: %s", aurora.Green("[NOTICE]"), aurora.Yellow(command.Command))
		switch command.Command {
		case "ka":
			conn.Write([]byte(`\ka\final\`))
			break
		case "login":
			payload := login(pool, ctx, command, challenge)
			conn.Write([]byte(payload))
			break
		case "getprofile":
			payload := getProfile(pool, ctx, command)
			fmt.Println(payload)
			conn.Write([]byte(payload))
			break
		}
	}
}
