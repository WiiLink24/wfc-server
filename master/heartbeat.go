package master

import (
	"encoding/binary"
	"github.com/logrusorgru/aurora/v3"
	"net"
	"strconv"
	"strings"
	"wwfc/logging"
)

func heartbeat(conn net.PacketConn, addr net.Addr, buffer []byte) {
	sessionId := binary.BigEndian.Uint32(buffer[1:5])
	moduleName := "AVAILABLE:" + strconv.FormatInt(int64(sessionId), 10)

	logging.Notice(moduleName, "Received heartbeat from", aurora.Cyan(addr).String())
	values := strings.Split(string(buffer[5:]), "\u0000")

	payload := map[string]string{}
	for i := 0; i < len(values); i += 2 {
		if values[i] == "" {
			break
		}

		payload[values[i]] = values[i+1]
		logging.Notice(moduleName, aurora.Cyan(values[i]).String()+":", aurora.Cyan(values[i+1]).String())
	}

	publicip, ok := payload["publicip"]
	if !ok || publicip == "0" {
		sendChallenge(conn, addr, sessionId)
		return
	}

	// TODO: Check if the client is registered

	statechanged, ok := payload["statechanged"]
	if ok {
		if statechanged == "1" {
			// statechanged is 1 and publicip is not 0
			// TODO: This would be a good place to run the server->client message exploit
			// for DNS patcher games that require code patches. The status code should be
			// set to 5 at this point, which is required.
			logging.Notice(moduleName, "Client server update")
			// Fall through
		}

		if statechanged == "2" {
			logging.Notice(moduleName, "Client server shutdown")
			return
		}
	}
}
