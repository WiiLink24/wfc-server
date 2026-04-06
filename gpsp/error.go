package gpsp

import (
	"wwfc/common"
	"wwfc/gpcm"
	"wwfc/logging"
)

func replyError(moduleName string, connIndex uint64, gpErr gpcm.GPError) {
	logging.Error(moduleName, "Reply error:", gpErr.ErrorString)
	err := common.SendPacket(ServerName, connIndex, []byte(gpErr.GetMessage()))
	if gpErr.Fatal || err != nil {
		if err != nil {
			logging.Error(moduleName, "Failed to send error message:", err)
		}
		if err := common.CloseConnection(ServerName, connIndex); err != nil {
			logging.Error(moduleName, "Failed to close connection:", err)
		}
	}
}
