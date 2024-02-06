package gamestats

import (
	"wwfc/gpcm"
	"wwfc/logging"
)

func (g *GameStatsSession) replyError(err gpcm.GPError) {
	logging.Error(g.ModuleName, "Reply error:", err.ErrorString)
	g.Conn.Write([]byte(err.GetMessage()))
}
