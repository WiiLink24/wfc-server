package gpcm

import (
	"strconv"
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

		case "mkw_malicious_packet":
			if g.GameName != "mariokartwii" {
				logging.Warn(g.ModuleName, "Ignoring mkw_malicious_packet from wrong game")
				continue
			}

			profileId, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				logging.Error(g.ModuleName, "Error decoding mkw_malicious_packet:", err.Error())
				continue
			}

			logging.Warn(g.ModuleName, "Malicious packet from", aurora.BrightCyan(strconv.FormatUint(profileId, 10)))

		case "mkw_room_stall":
			if g.GameName != "mariokartwii" {
				logging.Warn(g.ModuleName, "Ignoring mkw_room_stall from wrong game")
				continue
			}

			profileId, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				logging.Error(g.ModuleName, "Error decoding mkw_room_stall:", err.Error())
				continue
			}

			logging.Warn(g.ModuleName, "Room stall caused by", aurora.BrightCyan(strconv.FormatUint(profileId, 10)))
		}
	}
}
