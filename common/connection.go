package common

import (
	"net/rpc"
	"wwfc/logging"
)

var rpcFrontend *rpc.Client

type RPCFrontendPacket struct {
	Server string
	Index  uint64
	Data   []byte
}

// ConnectFrontend connects to the frontend RPC server
func ConnectFrontend() {
	var err error
	rpcFrontend, err = rpc.Dial("tcp", "localhost:29998")
	if err != nil {
		logging.Error("BACKEND", "Failed to connect to frontend:", err)
	}
}

// SendPacket is used by backend servers to send a packet to a connection
func SendPacket(server string, index uint64, data []byte) error {
	if rpcFrontend == nil {
		ConnectFrontend()
	}

	err := rpcFrontend.Call("RPCFrontendPacket.SendPacket", RPCFrontendPacket{Server: server, Index: index, Data: data}, nil)
	if err != nil {
		logging.Error("COMMON", "Failed to send packet to frontend:", err)
	}
	return err
}

// CloseConnection is used by backend servers to close a connection
func CloseConnection(server string, index uint64) error {
	if rpcFrontend == nil {
		ConnectFrontend()
	}

	err := rpcFrontend.Call("RPCFrontendPacket.CloseConnection", RPCFrontendPacket{Server: server, Index: index}, nil)
	if err != nil {
		logging.Error("COMMON", "Failed to close connection:", err)
	}
	return err
}
