package gpsp

import (
	"wwfc/common"
	"wwfc/gpcm"
	"wwfc/logging"
)

func replyError(moduleName string, connIndex uint64, err gpcm.GPError) {
	logging.Error(moduleName, "Reply error:", err.ErrorString)
	msg := err.GetMessage()
	common.SendPacket(ServerName, connIndex, []byte(msg))
	if err.Fatal {
		common.CloseConnection(ServerName, connIndex)
	}
}
