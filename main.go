package main

import (
	"errors"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"wwfc/api"
	"wwfc/common"
	"wwfc/gamestats"
	"wwfc/gpcm"
	"wwfc/gpsp"
	"wwfc/logging"
	"wwfc/nas"
	"wwfc/natneg"
	"wwfc/qr2"
	"wwfc/sake"
	"wwfc/serverbrowser"

	"github.com/logrusorgru/aurora/v3"
)

var config = common.GetConfig()

func main() {
	logging.SetLevel(*config.LogLevel)

	args := os.Args[1:]

	// Separate frontend and backend into two separate processes.
	// This is to allow restarting the backend without closing all connections.

	// Start the backend instead of the frontend if the first argument is "backend"
	if len(args) > 0 && args[0] == "backend" {
		backendMain()
	} else {
		frontendMain()
	}
}

type RPCPacket struct {
	Server  string
	Index   uint64
	Address string
	Data    []byte
}

// backendMain starts all the servers and creates an RPC server to communicate with the frontend
func backendMain() {
	if err := logging.SetOutput(config.LogOutput); err != nil {
		logging.Error("BACKEND", err)
	}

	rpc.Register(&RPCPacket{})
	address := "localhost:29999"

	l, err := net.Listen("tcp", address)
	if err != nil {
		logging.Error("BACKEND", "Failed to listen on", aurora.BrightCyan(address))
		os.Exit(1)
	}

	logging.Notice("BACKEND", "Listening on", aurora.BrightCyan(address))

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				logging.Error("BACKEND", "Failed to accept connection on", aurora.BrightCyan(address))
				continue
			}

			go rpc.ServeConn(conn)
		}
	}()

	// TODO: Wait until the servers are started before allowing in connections

	wg := &sync.WaitGroup{}
	actions := []func(){nas.StartServer, gpcm.StartServer, qr2.StartServer, gpsp.StartServer, serverbrowser.StartServer, sake.StartServer, natneg.StartServer, api.StartServer, gamestats.StartServer}
	wg.Add(len(actions))
	for _, action := range actions {
		go func(ac func()) {
			defer wg.Done()
			ac()
		}(action)
	}

	wg.Wait()
}

// RPCPacket.NewConnection is called by the frontend to notify the backend of a new connection
func (r *RPCPacket) NewConnection(args RPCPacket, _ *struct{}) error {
	switch args.Server {
	case "gpcm":
		gpcm.NewConnection(args.Index, args.Address)
	}

	return nil
}

// RPCPacket.HandlePacket is called by the frontend to forward a packet to the backend
func (r *RPCPacket) HandlePacket(args RPCPacket, _ *struct{}) error {
	switch args.Server {
	case "gpcm":
		gpcm.HandlePacket(args.Index, args.Data)
	}

	return nil
}

// rpcPacket.closeConnection is called by the frontend to notify the backend of a closed connection
func (r *RPCPacket) CloseConnection(args RPCPacket, _ *struct{}) error {
	switch args.Server {
	case "gpcm":
		gpcm.CloseConnection(args.Index)
	}

	return nil
}

type serverInfo struct {
	rpcName  string
	protocol string
	port     int
}

type RPCFrontendPacket struct {
	Server string
	Index  uint64
	Data   []byte
}

var (
	rpcClient *rpc.Client

	rpcMutex     sync.Mutex
	rpcBusyCount sync.WaitGroup

	connections = map[string]map[uint64]net.Conn{}
)

// frontendMain starts the backend process and communicates with it using RPC
func frontendMain() {
	// Don't allow the frontend to output to a file (there's no reason to)
	logOutput := config.LogOutput
	if logOutput == "StdOutAndFile" {
		logOutput = "StdOut"
	}

	if err := logging.SetOutput(logOutput); err != nil {
		logging.Error("FRONTEND", err)
	}

	rpcMutex.Lock()

	startFrontendServer()
	go startBackendProcess()

	servers := []serverInfo{
		// {rpcName: "serverbrowser", protocol: "tcp", port: 28910},
		{rpcName: "gpcm", protocol: "tcp", port: 29900},
		// {rpcName: "gpsp", protocol: "tcp", port: 29901},
		// {rpcName: "gamestats", protocol: "tcp", port: 29920},
	}

	for _, server := range servers {
		connections[server.rpcName] = map[uint64]net.Conn{}
		go frontendListen(server)
	}

	// Prevent application from exiting
	select {}
}

// startFrontendServer starts the frontend RPC server.
func startFrontendServer() {
	rpc.Register(&RPCFrontendPacket{})
	address := "localhost:29998"

	l, err := net.Listen("tcp", address)
	if err != nil {
		logging.Error("FRONTEND", "Failed to listen on", aurora.BrightCyan(address))
		os.Exit(1)
	}

	logging.Notice("FRONTEND", "Listening on", aurora.BrightCyan(address))

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				logging.Error("FRONTEND", "Failed to accept connection on", aurora.BrightCyan(address))
				continue
			}

			go rpc.ServeConn(conn)
		}
	}()
}

// startBackendProcess starts the backend process and waits for the RPC server to start.
// Expects the RPC mutex to be locked.
func startBackendProcess() {
	cmd := exec.Command(os.Args[0], "backend")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		logging.Error("FRONTEND", "Failed to start backend process:", err)
		os.Exit(1)
	}

	for {
		client, err := rpc.Dial("tcp", "localhost:29999")
		if err == nil {
			rpcClient = client
			rpcMutex.Unlock()
			break
		}
	}
}

// frontendListen listens on the specified port and forwards each packet to the backend
func frontendListen(server serverInfo) {
	address := *config.GameSpyAddress + ":" + strconv.Itoa(server.port)
	l, err := net.Listen(server.protocol, address)
	if err != nil {
		logging.Error("FRONTEND", "Failed to listen on", aurora.BrightCyan(address))
		return
	}

	logging.Notice("FRONTEND", "Listening on", aurora.BrightCyan(address), "for", aurora.BrightCyan(server.rpcName))

	// Increment by 1 for each connection, never decrement. Unlikely to overflow but it doesn't matter if it does.
	count := uint64(0)

	for {
		conn, err := l.Accept()
		if err != nil {
			logging.Error("FRONTEND", "Failed to accept connection on", aurora.BrightCyan(address))
			continue
		}

		if server.protocol == "tcp" {
			err := conn.(*net.TCPConn).SetKeepAlive(true)
			if err != nil {
				logging.Warn("FRONTEND", "Unable to set keepalive", err.Error())
			}
		}

		count++

		go handleConnection(server, conn, count)
	}
}

// handleConnection forwards packets between the frontend and backend
func handleConnection(server serverInfo, conn net.Conn, index uint64) {
	defer conn.Close()

	rpcMutex.Lock()
	rpcBusyCount.Add(1)
	connections[server.rpcName][index] = conn
	rpcMutex.Unlock()

	err := rpcClient.Call("RPCPacket.NewConnection", RPCPacket{Server: server.rpcName, Index: index, Address: conn.RemoteAddr().String(), Data: []byte{}}, nil)

	rpcBusyCount.Done()

	if err != nil {
		logging.Error("FRONTEND", "Failed to forward new connection to backend:", err)

		rpcMutex.Lock()
		delete(connections[server.rpcName], index)
		rpcMutex.Unlock()
		return
	}

	for {
		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)
		if err != nil {
			break
		}

		rpcMutex.Lock()
		rpcBusyCount.Add(1)
		rpcMutex.Unlock()

		// Forward the packet to the backend
		err = rpcClient.Call("RPCPacket.HandlePacket", RPCPacket{Server: server.rpcName, Index: index, Address: conn.RemoteAddr().String(), Data: buffer[:n]}, nil)

		rpcBusyCount.Done()

		if err != nil {
			logging.Error("FRONTEND", "Failed to forward packet to backend:", err)
			break
		}
	}

	rpcMutex.Lock()
	rpcBusyCount.Add(1)
	delete(connections[server.rpcName], index)
	rpcMutex.Unlock()

	err = rpcClient.Call("RPCPacket.CloseConnection", RPCPacket{Server: server.rpcName, Index: index, Address: conn.RemoteAddr().String(), Data: []byte{}}, nil)

	rpcBusyCount.Done()

	if err != nil {
		logging.Error("FRONTEND", "Failed to forward close connection to backend:", err)
	}
}

var ErrBadIndex = errors.New("incorrect connection index")

// RPCFrontendPacket.SendPacket is called by the backend to send a packet to a connection
func (r *RPCFrontendPacket) SendPacket(args RPCFrontendPacket, _ *struct{}) error {
	rpcMutex.Lock()
	defer rpcMutex.Unlock()

	conn, ok := connections[args.Server][args.Index]
	if !ok {
		return ErrBadIndex
	}

	_, err := conn.Write(args.Data)
	return err
}

// RPCFrontendPacket.CloseConnection is called by the backend to close a connection
func (r *RPCFrontendPacket) CloseConnection(args RPCFrontendPacket, _ *struct{}) error {
	rpcMutex.Lock()
	defer rpcMutex.Unlock()

	conn, ok := connections[args.Server][args.Index]
	if !ok {
		return ErrBadIndex
	}

	delete(connections[args.Server], args.Index)
	return conn.Close()
}
