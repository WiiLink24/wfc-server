package matchmaking

import (
	"bufio"
	"context"
	"encoding/binary"
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
	ModuleName = "MATCHMAKING"

	// Requests sent from the client
	ServerListRequest   = 0x00
	ServerInfoRequest   = 0x01
	SendMessageRequest  = 0x02
	KeepaliveReply      = 0x03
	MapLoopRequest      = 0x04
	PlayerSearchRequest = 0x05

	// Requests sent from the server to the client
	PushKeysMessage     = 0x01
	PushServerMessage   = 0x02
	KeepaliveMessage    = 0x03
	DeleteServerMessage = 0x04
	MapLoopMessage      = 0x05
	PlayerSearchMessage = 0x06
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
	bufferSize := 0
	packetSize := uint16(0)
	buffer := []byte{}
	for {
		// Remove stale data and remake the buffer
		buffer = append(buffer[packetSize:], make([]byte, 1024-packetSize)...)
		bufferSize -= int(packetSize)
		packetSize = 0

		for {
			if bufferSize > 2 {
				packetSize = binary.BigEndian.Uint16(buffer[:2])
				if packetSize < 3 {
					logging.Notice(ModuleName, "Invalid packet size - terminating")
					return
				}

				if bufferSize >= int(packetSize) {
					// Got a full packet, break to continue
					break
				}
			}

			readSize, err := bufio.NewReader(conn).Read(buffer[bufferSize:])
			if err != nil {
				if errors.Is(err, io.EOF) {
					logging.Notice(ModuleName, "Connection closed")
					return
				}

				logging.Notice(ModuleName, "Connection error")
				return
			}

			bufferSize += readSize
		}

		logging.Notice(ModuleName, "packet size:", aurora.Cyan(packetSize).String())
		logging.Notice(ModuleName, "buffer size:", aurora.Cyan(bufferSize).String())

		switch buffer[2] {
		case ServerListRequest:
			logging.Notice(ModuleName, "Command:", aurora.Yellow("SERVER_LIST_REQUEST").String())
			handleServerListRequest(conn, buffer[:packetSize])
			break

		case ServerInfoRequest:
			logging.Notice(ModuleName, "Command:", aurora.Yellow("SERVER_INFO_REQUEST").String())
			break

		case SendMessageRequest:
			logging.Notice(ModuleName, "Command:", aurora.Yellow("SEND_MESSAGE_REQUEST").String())
			handleSendMessageRequest(conn, buffer[:packetSize])
			break

		case KeepaliveReply:
			logging.Notice(ModuleName, "Command:", aurora.Yellow("KEEPALIVE_REPLY").String())
			break

		case MapLoopRequest:
			logging.Notice(ModuleName, "Command:", aurora.Yellow("MAPLOOP_REQUEST").String())
			break

		case PlayerSearchRequest:
			logging.Notice(ModuleName, "Command:", aurora.Yellow("PLAYER_SEARCH_REQUEST").String())
			break

		default:
			logging.Notice(ModuleName, "Unknown command:", aurora.Yellow(buffer[2]).String())
			break
		}
	}
}
