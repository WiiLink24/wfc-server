package gpsp

import (
	"net"
	"wwfc/gpcm"
	"wwfc/logging"
)

func replyError(moduleName string, conn net.Conn, err gpcm.GPError) {
	logging.Error(moduleName, "Reply error:", err.ErrorString)
	conn.Write([]byte(err.GetMessage()))
}
