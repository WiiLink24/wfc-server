package main

import (
	"errors"
	"io"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"wwfc/api"
	"wwfc/common"
	"wwfc/gamestats"
	"wwfc/gpcm"
	"wwfc/gpsp"
	"wwfc/logging"
	"wwfc/nas"
	"wwfc/natneg"
	"wwfc/nhttp"
	"wwfc/qr2"
	"wwfc/race"
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
	noSignal := false
	noReload := false

	if len(args) > 1 {
		for _, arg := range args[1:] {
			switch arg {
			case "--nosignal":
				noSignal = true
			case "--noreload":
				noReload = true
			}
		}
	}

	// Start the backend instead of the frontend if the first argument is "backend"
	if len(args) > 0 && args[0] == "backend" {
		backendMain(noSignal, noReload)
	} else {
		frontendMain(noSignal, len(args) > 0 && args[0] == "frontend")
	}
}

type RPCPacket struct {
	Server  string
	Index   uint64
	Address string
	Data    []byte
}

// backendMain starts all the servers and creates an RPC server to communicate with the frontend
func backendMain(noSignal, noReload bool) {
	err := os.Mkdir("state", 0755)
	if err != nil && !os.IsExist(err) {
		logging.Error("BACKEN", err)
		os.Exit(1)
	}

	sigExit := make(chan os.Signal, 1)
	signal.Notify(sigExit, syscall.SIGINT, syscall.SIGTERM)

	if err := logging.SetOutput(config.LogOutput); err != nil {
		logging.Error("BACKEND", err)
	}

	rpc.Register(&RPCPacket{})
	address := config.BackendAddress

	l, err := net.Listen("tcp", address)
	if err != nil {
		logging.Error("BACKEND", "Failed to listen on", aurora.BrightCyan(address))
		os.Exit(1)
	}

	common.ConnectFrontend()

	uuid := ""
	if !noReload {
		uuid = loadUuidFile()
	}

	reload, err := common.VerifyState(uuid)
	if err != nil {
		panic(err)
	}

	wg := &sync.WaitGroup{}
	actions := []func(bool){nas.StartServer, gpcm.StartServer, qr2.StartServer, gpsp.StartServer, serverbrowser.StartServer, race.StartServer, sake.StartServer, natneg.StartServer, api.StartServer, gamestats.StartServer}
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

	if noSignal {
		select {}
	}

	stateUuid, err := common.Shutdown()
	if err != nil {
		panic(err)
	}

	(&RPCPacket{}).Shutdown(stateUuid, &struct{}{})
}

func loadUuidFile() string {
	stateFile, err := os.Open("state/uuid.txt")
	if err != nil {
		return ""
	}

	defer stateFile.Close()

	uuid, err := io.ReadAll(stateFile)
	if err != nil {
		logging.Error("BACKEND", "Failed to read state file:", err)
		return ""
	}

	return string(uuid)
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
func (r *RPCPacket) Shutdown(stateUuid string, _ *struct{}) error {
	if stateUuid == "" {
		os.Exit(0)
		return nil
	}

	wg := &sync.WaitGroup{}
	actions := []func(){nas.Shutdown, gpcm.Shutdown, qr2.Shutdown, gpsp.Shutdown, serverbrowser.Shutdown, race.Shutdown, sake.Shutdown, natneg.Shutdown, api.Shutdown, gamestats.Shutdown}
	wg.Add(len(actions))
	for _, action := range actions {
		go func(ac func()) {
			defer wg.Done()
			ac()
		}(action)
	}

	wg.Wait()

	stateFile, err := os.OpenFile("state/uuid.txt", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}

	_, err = stateFile.WriteString(stateUuid)
	if err != nil {
		panic(err)
	}

	err = stateFile.Close()
	if err != nil {
		panic(err)
	}

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

	// This mutex could be locked for a very long time, don't use deadlock detection
	rpcMutex   sync.Mutex
	rpcWaiting nhttp.AtomicBool

	rpcBusyCount sync.WaitGroup
	backendReady = make(chan struct{})
	frontendUuid string

	connections = map[string]map[uint64]*net.Conn{}

	integrated = false
)

// frontendMain starts the backend process and communicates with it using RPC
func frontendMain(noSignal, noBackend bool) {
	rpcWaiting.SetFalse()

	integrated = !noBackend

	sigExit := make(chan os.Signal, 1)
	signal.Notify(sigExit, syscall.SIGINT, syscall.SIGTERM)

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
		connections[server.rpcName] = map[uint64]*net.Conn{}
		go frontendListen(server)
	}

	// Wait for a signal to shutdown
	<-sigExit

	if noSignal {
		select {}
	}

	// If we're waiting for the backend to connect, then don't try to lock the
	// mutex because it's never going to unlock
	if rpcWaiting.IsSet() {
		logging.Notice("FRONTEND", "Backend rpcClient is not connected")
		return
	}

	rpcMutex.Lock()
	if rpcClient == nil {
		logging.Notice("FRONTEND", "Backend rpcClient is not connected")
		rpcMutex.Unlock()
		return
	}
	rpcMutex.Unlock()

	logging.Notice("FRONTEND", "Sending RPCPacket.Shutdown")
	rpcClient.Call("RPCPacket.Shutdown", "", nil)
	rpcClient.Close()
}

// startFrontendServer starts the frontend RPC server.
func startFrontendServer() {
	rpc.Register(&RPCFrontendPacket{})
	address := config.FrontendAddress

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
		cmd = exec.Command(exe, "backend", "--nosignal")
	} else {
		cmd = exec.Command(exe, "backend", "--noreload", "--nosignal")
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
	rpcWaiting.SetTrue()
	<-backendReady
	backendReady = make(chan struct{})

	for {
		client, err := rpc.Dial("tcp", config.FrontendBackendAddress)
		if err == nil {
			rpcClient = client
			rpcMutex.Unlock()

			rpcWaiting.SetFalse()
			logging.Notice("FRONTEND", "Connected to backend")

			return
		}

		<-time.After(50 * time.Millisecond)
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
	pConn := &conn
	connections[server.rpcName][index] = pConn
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
	if connections[server.rpcName][index] != pConn {
		rpcMutex.Unlock()
		return
	}

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

var (
	ErrBadIndex = errors.New("incorrect connection index")
	ErrorBusy   = errors.New("backend is busy")
)

// RPCFrontendPacket.SendPacket is called by the backend to send a packet to a connection
func (r *RPCFrontendPacket) SendPacket(args RPCFrontendPacket, _ *struct{}) error {
	rpcMutex.Lock()
	defer rpcMutex.Unlock()

	conn := connections[args.Server][args.Index]
	if conn == nil {
		return ErrBadIndex
	}

	_, err := (*conn).Write(args.Data)

	return err
}

// RPCFrontendPacket.CloseConnection is called by the backend to close a connection
func (r *RPCFrontendPacket) CloseConnection(args RPCFrontendPacket, _ *struct{}) error {
	rpcMutex.Lock()
	defer rpcMutex.Unlock()

	conn := connections[args.Server][args.Index]
	if conn == nil {
		return ErrBadIndex
	}

	return (*conn).Close()
}

// RPCFrontendPacket.ReloadBackend is called by an external program to reload the backend
func (r *RPCFrontendPacket) ReloadBackend(_ struct{}, _ *struct{}) error {
	var stateUid string
	r.ShutdownBackend(struct{}{}, &stateUid)

	err := rpcClient.Call("RPCPacket.Shutdown", stateUid, nil)
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

// RPCFrontendPacket.ShutdownBackend is called by the backend to prepare for shutdown
func (r *RPCFrontendPacket) ShutdownBackend(_ struct{}, uuid *string) error {
	logging.Notice("FRONTEND", "Shutting down backend")

	// Lock indefinitely
	rpcMutex.Lock()

	rpcBusyCount.Wait()

	if !integrated {
		go waitForBackend()

		frontendUuid = common.RandomString(32)
		*uuid = frontendUuid
	} else {
		*uuid = ""
	}

	return nil
}

// RPCFrontendPacket.VerifyState is called by the backend to verify the state UUID
func (r *RPCFrontendPacket) VerifyState(uuid string, reload *bool) error {
	if rpcMutex.TryLock() {
		rpcMutex.Unlock()
		logging.Error("FRONTEND", "Failed to verify UUID, backend is active")
		*reload = false
		return ErrorBusy
	}

	if uuid != frontendUuid {
		logging.Notice("FRONTEND", "VerifyState: Resetting all connections")

		// Close all connections
		for _, server := range connections {
			for index, conn := range server {
				(*conn).Close()
				delete(server, index)
			}
		}

		*reload = false
		return nil
	}

	*reload = uuid != ""

	return nil
}

// RPCFrontendPacket.Ready is called by the backend to indicate it is ready to accept connections
func (r *RPCFrontendPacket) Ready(_ struct{}, _ *struct{}) error {
	close(backendReady)

	return nil
}
