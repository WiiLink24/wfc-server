package gpsp

import (
	"bufio"
	"context"
	"net"
	"wwfc/common"
	"wwfc/gpcm"
	"wwfc/logging"

	"github.com/jackc/pgx/v4/pgxpool"
)

var (
	ctx    = context.Background()
	pool   *pgxpool.Pool
	userId int64
)

func StartServer() {
	// Get config
	config := common.GetConfig()

	address := *config.GameSpyAddress + ":29901"
	l, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}

	// Close the listener when the application closes.
	defer l.Close()
	logging.Notice("GPSP", "Listening on", address)

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

	moduleName := "GPSP"

	err := conn.(*net.TCPConn).SetKeepAlive(true)
	if err != nil {
		logging.Notice(moduleName, "Unable to set keepalive:", err.Error())
	}

	// Here we go into the listening loop
	for {
		// TODO: Handle split packets
		buffer := make([]byte, 1024)
		_, err := bufio.NewReader(conn).Read(buffer)
		if err != nil {
			return
		}

		commands, err := common.ParseGameSpyMessage(string(buffer))
		if err != nil {
			logging.Error(moduleName, "Error parsing message:", err.Error())
			logging.Error(moduleName, "Raw data:", string(buffer))
			replyError(moduleName, conn, gpcm.ErrParse)
			return
		}

		for _, command := range commands {
			switch command.Command {
			default:
				logging.Error(moduleName, "Unknown command:", command.Command)

			case "ka":
				conn.Write([]byte(`\ka\\final\`))
				break

			case "otherslist":
				conn.Write([]byte(handleOthersList(command)))
				break
			}
		}
	}
}
