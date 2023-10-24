package master

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"wwfc/common"
)

func sendChallenge(conn net.PacketConn, addr net.Addr, session Session) {
	challenge := session.Challenge
	if challenge == "" {
		// Generate challenge
		addrString := strings.Split(addr.String(), ":")
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

		challenge = common.RandomString(6) + "00" + hexIP + hexPort
		mutex.Lock()
		sessions[session.SessionID].Challenge = challenge
		mutex.Unlock()
	}

	response := createResponseHeader(ChallengeRequest, session.SessionID)
	response = append(response, []byte(challenge)...)
	response = append(response, 0)

	conn.WriteTo(response, addr)
}
