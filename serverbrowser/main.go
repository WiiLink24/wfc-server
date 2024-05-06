package serverbrowser

import (
	"encoding/binary"
	"wwfc/common"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
	"github.com/sasha-s/go-deadlock"
)

var ServerName = "serverbrowser"

const (
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

var (
	connBuffers = map[uint64]*[]byte{}
	mutex       = deadlock.RWMutex{}
)

func StartServer() {
}

func NewConnection(index uint64, address string) {
}

func CloseConnection(index uint64) {
	mutex.Lock()
	delete(connBuffers, index)
	mutex.Unlock()
}

func HandlePacket(index uint64, data []byte, address string) {
	moduleName := "SB:" + address

	mutex.RLock()
	buffer := connBuffers[index]
	mutex.RUnlock()

	if buffer == nil {
		buffer = &[]byte{}
		defer func() {
			if buffer == nil {
				return
			}

			mutex.Lock()
			connBuffers[index] = buffer
			mutex.Unlock()
		}()
	}

	if len(*buffer)+len(data) > 0x1000 {
		logging.Error(moduleName, "Buffer overflow")
		common.CloseConnection(ServerName, index)
		buffer = nil
		return
	}

	*buffer = append(*buffer, data...)

	// Packets can be sent in fragments, so we need to check if we have a full packet
	// The first two bytes signify the packet size
	if len(*buffer) < 2 {
		return
	}

	packetSize := binary.BigEndian.Uint16((*buffer)[:2])
	if packetSize < 3 || packetSize > 0x1000 {
		logging.Error(moduleName, "Invalid packet size - terminating")
		common.CloseConnection(ServerName, index)
		buffer = nil
		return
	}

	if len(*buffer) < int(packetSize) {
		return
	}

	switch (*buffer)[2] {
	case ServerListRequest:
		// logging.Info(moduleName, "Command:", aurora.Yellow("SERVER_LIST_REQUEST"))
		handleServerListRequest(moduleName, index, address, (*buffer)[:packetSize])

	case ServerInfoRequest:
		logging.Info(moduleName, "Command:", aurora.Yellow("SERVER_INFO_REQUEST"))

	case SendMessageRequest:
		// logging.Info(moduleName, "Command:", aurora.Yellow("SEND_MESSAGE_REQUEST"))
		handleSendMessageRequest(moduleName, index, address, (*buffer)[:packetSize])

	case KeepaliveReply:
		logging.Info(moduleName, "Command:", aurora.Yellow("KEEPALIVE_REPLY"))

	case MapLoopRequest:
		logging.Info(moduleName, "Command:", aurora.Yellow("MAPLOOP_REQUEST"))

	case PlayerSearchRequest:
		logging.Info(moduleName, "Command:", aurora.Yellow("PLAYER_SEARCH_REQUEST"))

	default:
		logging.Error(moduleName, "Unknown command:", aurora.Cyan((*buffer)[2]))
	}

	if len(*buffer) > int(packetSize) {
		*buffer = (*buffer)[packetSize:]
	} else {
		*buffer = []byte{}
		buffer = nil
	}
}
