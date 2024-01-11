package gpcm

import (
	"wwfc/common"
	"wwfc/logging"
	"wwfc/qr2"

	"github.com/logrusorgru/aurora/v3"
)

func (g *GameSpySession) handleWWFCReport(command common.GameSpyCommand) {
	for key, value := range command.OtherValues {
		logging.Info(g.ModuleName, "WWFC Report:", aurora.Yellow(key))

		switch key {
		case "mkw_user":
			if g.GameName != "mariokartwii" {
				logging.Warn(g.ModuleName, "Ignoring mkw_user packet from wrong game")
				continue
			}

			packet, err := common.Base64DwcEncoding.DecodeString(value)
			if err != nil {
				logging.Error(g.ModuleName, "Error decoding mkw_user packet:", err.Error())
				continue
			}

			if len(packet) != 0xC0 {
				logging.Error(g.ModuleName, "Invalid mkw_user packet length:", len(packet))
				continue
			}

			qr2.ProcessUSER(g.User.ProfileId, g.QR2IP, packet)
			break
		}
	}
}
