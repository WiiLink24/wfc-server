package main

import (
	"errors"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
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

var (
	config = common.GetConfig()
)

func main() {
	logging.SetLevel(*config.LogLevel)

	args := os.Args[1:]

	// Separate frontend and backend into two separate processes.
	// This is to allow restarting the backend without closing all connections.

	// Start the backend instead of the frontend if the first argument is "backend"
	if len(args) > 0 && args[0] == "backend" {
		backendMain(len(args) > 1 && args[1] == "reload")
	} else {
		frontendMain(len(args) > 0 && args[0] == "frontend")
	}
}

type RPCPacket struct {
	Server  string
	Index   uint64
	Address string
	Data    []byte
}

// backendMain starts all the servers and creates an RPC server to communicate with the frontend
func backendMain(reload bool) {
	sigExit := make(chan os.Signal, 1)
	signal.Notify(sigExit, syscall.SIGINT, syscall.SIGTERM)

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

	common.ConnectFrontend()

	wg := &sync.WaitGroup{}
	actions := []func(bool){nas.StartServer, gpcm.StartServer, qr2.StartServer, gpsp.StartServer, serverbrowser.StartServer, sake.StartServer, natneg.StartServer, api.StartServer, gamestats.StartServer}
	wg.Add(len(actions))
	for _, action := range actions {
		go func(ac func(bool)) {
			defer wg.Done()
			ac(reload)
		}(action)
	}

	// Wait for all servers to start
	wg.Wait()

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

	logging.Notice("BACKEND", "Listening on", aurora.BrightCyan(address))

	common.Ready()

	// Wait for a signal to shutdown
	<-sigExit

	err = common.Shutdown()
	if err != nil {
		panic(err)
	}

	(&RPCPacket{}).Shutdown(struct{}{}, &struct{}{})
}

// RPCPacket.NewConnection is called by the frontend to notify the backend of a new connection
func (r *RPCPacket) NewConnection(args RPCPacket, _ *struct{}) error {
	switch args.Server {
	case "serverbrowser":
		serverbrowser.NewConnection(args.Index, args.Address)
	case "gpcm":
		gpcm.NewConnection(args.Index, args.Address)
	case "gpsp":
		gpsp.NewConnection(args.Index, args.Address)
	case "gamestats":
		gamestats.NewConnection(args.Index, args.Address)
	}

	return nil
}

// RPCPacket.HandlePacket is called by the frontend to forward a packet to the backend
func (r *RPCPacket) HandlePacket(args RPCPacket, _ *struct{}) error {
	switch args.Server {
	case "serverbrowser":
		serverbrowser.HandlePacket(args.Index, args.Data, args.Address)
	case "gpcm":
		gpcm.HandlePacket(args.Index, args.Data)
	case "gpsp":
		gpsp.HandlePacket(args.Index, args.Data)
	case "gamestats":
		gamestats.HandlePacket(args.Index, args.Data)
	}

	return nil
}

// RPCPacket.closeConnection is called by the frontend to notify the backend of a closed connection
func (r *RPCPacket) CloseConnection(args RPCPacket, _ *struct{}) error {
	switch args.Server {
	case "serverbrowser":
		serverbrowser.CloseConnection(args.Index)
	case "gpcm":
		gpcm.CloseConnection(args.Index)
	case "gpsp":
		gpsp.CloseConnection(args.Index)
	case "gamestats":
		gamestats.CloseConnection(args.Index)
	}

	return nil
}

// RPCPacket.Shutdown is called by the frontend to shutdown the backend
func (r *RPCPacket) Shutdown(_ struct{}, _ *struct{}) error {
	wg := &sync.WaitGroup{}
	actions := []func(){nas.Shutdown, gpcm.Shutdown, qr2.Shutdown, gpsp.Shutdown, serverbrowser.Shutdown, sake.Shutdown, natneg.Shutdown, api.Shutdown, gamestats.Shutdown}
	wg.Add(len(actions))
	for _, action := range actions {
		go func(ac func()) {
			defer wg.Done()
			ac()
		}(action)
	}

	wg.Wait()

	os.Exit(0)
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
	backendReady = make(chan struct{})

	connections = map[string]map[uint64]net.Conn{}
)

// frontendMain starts the backend process and communicates with it using RPC
func frontendMain(noBackend bool) {
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

	if !noBackend {
		go startBackendProcess(false, true)
	} else {
		go waitForBackend()
	}

	servers := []serverInfo{
		{rpcName: "serverbrowser", protocol: "tcp", port: 28910},
		{rpcName: "gpcm", protocol: "tcp", port: 29900},
		{rpcName: "gpsp", protocol: "tcp", port: 29901},
		{rpcName: "gamestats", protocol: "tcp", port: 29920},
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

// startBackendProcess starts the backend process and (optionally) waits for the RPC server to start.
// If wait is true, expects the RPC mutex to be locked.
func startBackendProcess(reload bool, wait bool) {
	exe, err := os.Executable()
	if err != nil {
		logging.Error("FRONTEND", "Failed to get executable path:", err)
		os.Exit(1)
	}

	logging.Info("FRONTEND", "Running from", aurora.BrightCyan(exe))

	var cmd *exec.Cmd
	if reload {
		cmd = exec.Command(exe, "backend", "reload")
	} else {
		cmd = exec.Command(exe, "backend")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		logging.Error("FRONTEND", "Failed to start backend process:", err)
		os.Exit(1)
	}

	if wait {
		waitForBackend()
	}
}

// waitForBackend waits for the backend to start.
// Expects the RPC mutex to be locked.
func waitForBackend() {
	<-backendReady
	backendReady = make(chan struct{})

	for {
		client, err := rpc.Dial("tcp", "localhost:29999")
		if err == nil {
			rpcClient = client
			rpcMutex.Unlock()

			logging.Notice("FRONTEND", "Connected to backend")
			return
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

		if n == 0 {
			continue
		}

		rpcMutex.Lock()
		rpcBusyCount.Add(1)
		rpcMutex.Unlock()

		// Forward the packet to the backend
		err = rpcClient.Call("RPCPacket.HandlePacket", RPCPacket{Server: server.rpcName, Index: index, Address: conn.RemoteAddr().String(), Data: buffer[:n]}, nil)

		rpcBusyCount.Done()

		if err != nil {
			logging.Error("FRONTEND", "Failed to forward packet to backend:", err)
			if err == rpc.ErrShutdown {
				os.Exit(1)
			}
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
		if err == rpc.ErrShutdown {
			os.Exit(1)
		}
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

// RPCFrontendPacket.ReloadBackend is called by an external program to reload the backend
func (r *RPCFrontendPacket) ReloadBackend(_ struct{}, _ *struct{}) error {
	r.ShutdownBackend(struct{}{}, &struct{}{})

	err := rpcClient.Call("RPCPacket.Shutdown", struct{}{}, nil)
	if err != nil && !strings.Contains(err.Error(), "An existing connection was forcibly closed by the remote host.") {
		logging.Error("FRONTEND", "Failed to reload backend:", err)
	}

	err = rpcClient.Close()
	if err != nil {
		logging.Error("FRONTEND", "Failed to close RPC client:", err)
	}

	// Unlocks the mutex locked by ShutdownBackend
	startBackendProcess(true, true)

	return nil
}

// RPCFrontendPacket.ShutdownBackend is called by an external program to shutdown the backend
func (r *RPCFrontendPacket) ShutdownBackend(_ struct{}, _ *struct{}) error {
	logging.Notice("FRONTEND", "Shutting down backend")

	// Lock indefinitely
	rpcMutex.Lock()

	rpcBusyCount.Wait()

	go waitForBackend()

	return nil
}

// RPCFrontendPacket.Ready is called by the backend to indicate it is ready to accept connections
func (r *RPCFrontendPacket) Ready(_ struct{}, _ *struct{}) error {
	close(backendReady)
	return nil
}
