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
		panic(err)
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

// Ready will notify the frontend that the backend is ready to accept connections
func Ready() error {
	if rpcFrontend == nil {
		ConnectFrontend()
	}

	err := rpcFrontend.Call("RPCFrontendPacket.Ready", struct{}{}, nil)
	if err != nil {
		logging.Error("COMMON", "Failed to notify frontend that backend is ready:", err)
	}
	return err
}

// Shutdown will notify the frontend that the backend is shutting down
func Shutdown() (string, error) {
	if rpcFrontend == nil {
		ConnectFrontend()
	}

	var stateUuid string
	err := rpcFrontend.Call("RPCFrontendPacket.ShutdownBackend", struct{}{}, &stateUuid)
	if err != nil {
		logging.Error("COMMON", "Failed to notify frontend that backend is shutting down:", err)
	}

	return stateUuid, err
}

// VerifyState will verify the state UUID with the frontend
func VerifyState(stateUuid string) (bool, error) {
	if rpcFrontend == nil {
		ConnectFrontend()
	}

	valid := false
	err := rpcFrontend.Call("RPCFrontendPacket.VerifyState", stateUuid, &valid)
	if err != nil {
		logging.Error("COMMON", "Failed to verify state UUID with frontend:", err)
	}
	return valid, err
}
