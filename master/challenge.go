package master

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"wwfc/common"
)

func sendChallenge(conn net.PacketConn, addr net.Addr, sessionId uint32) {
	addrString := strings.Split(addr.String(), ":")

	// Generate challenge and send to server
	var hexIP string
	for _, i := range strings.Split(addrString[0], ".") {
		val, err := strconv.ParseUint(i, 10, 64)
		if err != nil {
			panic(err)
		}

		hexIP += fmt.Sprintf("%02X", val)
	}

	port, err := strconv.ParseUint(addrString[1], 10, 64)
	if err != nil {
		panic(err)
	}

	hexPort := fmt.Sprintf("%04X", port)

	challenge := common.RandomString(6) + "00" + hexIP + hexPort
	mutex.Lock()
	session := sessions[sessionId]
	session.Challenge = challenge
	mutex.Unlock()

	response := createResponseHeader(Command_CHALLENGE, sessionId)
	response = append(response, []byte(challenge)...)
	response = append(response, 0)

	conn.WriteTo(response, addr)
}
