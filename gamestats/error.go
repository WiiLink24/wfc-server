package gamestats

import (
	"wwfc/common"
	"wwfc/gpcm"
	"wwfc/logging"
)

func (g *GameStatsSession) replyError(err gpcm.GPError) {
	logging.Error(g.ModuleName, "Reply error:", err.ErrorString)
	common.SendPacket(ServerName, g.ConnIndex, []byte(err.GetMessage()))
	if err.Fatal {
		common.CloseConnection(ServerName, g.ConnIndex)
	}
}
