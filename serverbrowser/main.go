package serverbrowser

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
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
	ModuleName = "SB"

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

	address := *config.GameSpyAddress + ":28910"
	l, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}

	go func() {
		// Close the listener when the application closes.
		defer l.Close()
		logging.Notice(ModuleName, "Listening on", address)

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
	}()
}

// Handles incoming requests.
func handleRequest(conn net.Conn) {
	defer conn.Close()

	err := conn.(*net.TCPConn).SetKeepAlive(true)
	if err != nil {
		logging.Error(ModuleName, "Unable to set keepalive", err.Error())
	}

	logging.Info(ModuleName, "Connection established from", aurora.BrightCyan(conn.RemoteAddr()))

	// Here we go into the listening loop
	bufferSize := 0
	packetSize := uint16(0)
	var buffer []byte
	for {
		// Remove stale data and remake the buffer
		buffer = append(buffer[packetSize:], make([]byte, 1024-packetSize)...)
		bufferSize -= int(packetSize)
		packetSize = 0

		// Packets tend to be sent in fragments, so this loop makes sure the packets has been fully received before continuing
		for {
			if bufferSize > 2 {
				packetSize = binary.BigEndian.Uint16(buffer[:2])
				if packetSize < 3 || packetSize >= 1024 {
					logging.Error(ModuleName, "Invalid packet size - terminating")
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
					logging.Info(ModuleName, "Connection closed")
					return
				}

				logging.Error(ModuleName, "Connection error:", err.Error())
				return
			}

			bufferSize += readSize
		}

		switch buffer[2] {
		case ServerListRequest:
			logging.Info(ModuleName, "Command:", aurora.Yellow("SERVER_LIST_REQUEST"))
			handleServerListRequest(conn, buffer[:packetSize])
			break

		case ServerInfoRequest:
			logging.Info(ModuleName, "Command:", aurora.Yellow("SERVER_INFO_REQUEST"))
			break

		case SendMessageRequest:
			logging.Info(ModuleName, "Command:", aurora.Yellow("SEND_MESSAGE_REQUEST"))
			handleSendMessageRequest(conn, buffer[:packetSize])
			break

		case KeepaliveReply:
			logging.Info(ModuleName, "Command:", aurora.Yellow("KEEPALIVE_REPLY"))
			break

		case MapLoopRequest:
			logging.Info(ModuleName, "Command:", aurora.Yellow("MAPLOOP_REQUEST"))
			break

		case PlayerSearchRequest:
			logging.Info(ModuleName, "Command:", aurora.Yellow("PLAYER_SEARCH_REQUEST"))
			break

		default:
			logging.Error(ModuleName, "Unknown command:", aurora.Cyan(buffer[2]))
			break
		}
	}
}
