package gamestats

import (
	"wwfc/common"
	"wwfc/gpcm"
	"wwfc/logging"
)

func (g *GameStatsSession) replyError(gpErr gpcm.GPError) {
	logging.Error(g.ModuleName, "Reply error:", gpErr.ErrorString)
	err := common.SendPacket(ServerName, g.ConnIndex, []byte(gpErr.GetMessage()))
	if gpErr.Fatal || err != nil {
		if err != nil {
			logging.Error(g.ModuleName, "Failed to send error message:", err)
		}
		if err := common.CloseConnection(ServerName, g.ConnIndex); err != nil {
			logging.Error(g.ModuleName, "Failed to close connection:", err)
		}
	}
}
