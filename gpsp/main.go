package gpsp

import (
	"fmt"
	"net"
	"os"
)

func StartServer() {
	l, err := net.Listen("tcp", "127.0.0.1:27900")
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	fmt.Println("Listening on " + "127.0.0.1:29901")
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		fmt.Println("aaaa")
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
		fmt.Println("Error reading:", err.Error())
	}
	fmt.Println(reqLen)
	fmt.Println(string(buf))
	// Send a response back to person contacting us.
	conn.Write([]byte(`\ka\\final\`))
	// Close the connection when you're done with it.
	conn.Close()
}
