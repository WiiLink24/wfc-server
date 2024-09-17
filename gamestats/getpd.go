package gamestats

import (
	"strconv"
	"time"
	"wwfc/common"
)

func (g *GameStatsSession) getpd(command common.GameSpyCommand) {
	// Temporary empty data, it's an embedded gamespy \key\value message excluding \final\
	data := `\\`

	g.Write(common.GameSpyCommand{
		Command:      "getpdr",
		CommandValue: "1",
		OtherValues: map[string]string{
			"lid":    strconv.Itoa(g.LoginID),
			"pid":    command.OtherValues["pid"],
			"mod":    strconv.Itoa(int(time.Now().Unix())),
			"length": strconv.Itoa(len(data)),
			"data":   `\` + data + `\`,
		},
	})
}
