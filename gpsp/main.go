package gpsp

import (
	"fmt"
	"net"
	"os"
	"wwfc/common"
	"wwfc/logging"
)

func StartServer() {
	// Get config
	config := common.GetConfig()

	address := config.Address + ":27900"
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
	// Make a buffer to hold incoming data.
	buf := make([]byte, 1024)
	// Read the incoming connection into the buffer.
	reqLen, err := conn.Read(buf)
	if err != nil {
		panic(err)
	}

	// Send a response back to person contacting us.
	conn.Write([]byte(`\ka\\final\`))
	// Close the connection when you're done with it.
	conn.Close()
}
