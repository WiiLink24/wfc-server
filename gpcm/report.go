package gpcm

import (
	"encoding/json"
	"strconv"
	"time"
	"wwfc/common"
	"wwfc/logging"
	"wwfc/qr2"

	"github.com/logrusorgru/aurora/v3"
)

type RaceResultPlayer struct {
	Pid          int `json:"pid"`
	FinishTimeMs int `json:"finish_time_ms"`
	CharacterId  int `json:"character_id"`
	KartId       int `json:"kart_id"`
	PlayerCount  int `json:"player_count"`
}

type RaceResult struct {
	ClientReportVersion string            `json:"client_report_version"`
	Player              *RaceResultPlayer `json:"player"`
}

func (g *GameSpySession) handleWWFCReport(command common.GameSpyCommand) {
	for key, value := range command.OtherValues {
		logging.Info(g.ModuleName, "WiiLink Report:", aurora.Yellow(key))

		keyColored := aurora.BrightCyan(key).String()

		switch key {
		default:
			logging.Error(g.ModuleName, "Unknown record", aurora.Cyan(key).String()+":", aurora.Cyan(value))

		case "wl:bad_packet":
			profileId, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				logging.Error(g.ModuleName, "Error decoding", keyColored+":", err.Error())
				continue
			}

			logging.Warn(g.ModuleName, "Report bad packet from", aurora.BrightCyan(strconv.FormatUint(profileId, 10)))

		case "wl:stall":
			profileId, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				logging.Error(g.ModuleName, "Error decoding", keyColored+":", err.Error())
				continue
			}

			logging.Warn(g.ModuleName, "Room stall caused by", aurora.BrightCyan(strconv.FormatUint(profileId, 10)))

		case "wl:mkw_user":
			if g.GameName != "mariokartwii" {
				logging.Warn(g.ModuleName, "Ignoring", keyColored+":", "from wrong game")
				continue
			}

			packet, err := common.Base64DwcEncoding.DecodeString(value)
			if err != nil {
				logging.Error(g.ModuleName, "Error decoding", keyColored+":", err.Error())
				continue
			}

			if len(packet) != 0xC0 {
				logging.Error(g.ModuleName, "Invalid", keyColored, "record length:", len(packet))
				continue
			}

			qr2.ProcessUSER(g.User.ProfileId, g.QR2IP, packet)

		case "wl:mkw_select_course", "wl:mkw_select_cc":
			if g.GameName != "mariokartwii" {
				logging.Warn(g.ModuleName, "Ignoring", keyColored, "from wrong game")
				continue
			}

			qr2.ProcessMKWSelectRecord(g.User.ProfileId, key, value)

		case "wl:mkw_race_result":
			if g.GameName != "mariokartwii" {
				logging.Warn(g.ModuleName, "Ignoring", keyColored, "from wrong game")
				continue
			}

			logging.Info(g.ModuleName, "Received race result from profile", aurora.BrightCyan(strconv.FormatUint(uint64(g.User.ProfileId), 10)))

			var raceResult RaceResult
			err := json.Unmarshal([]byte(value), &raceResult)
			if err != nil {
				logging.Error(g.ModuleName, "Error parsing race result JSON:", err.Error())
				logging.Info(g.ModuleName, "Raw payload:", aurora.BrightMagenta(value))
				continue
			}

			logging.Info(g.ModuleName, "Race result version:", aurora.Yellow(raceResult.ClientReportVersion))

			player := raceResult.Player

			logging.Info(g.ModuleName,
				"Player",
				"- PID:", aurora.Cyan(strconv.Itoa(player.Pid)),
				"Time:", aurora.Cyan(strconv.Itoa(player.FinishTimeMs)), "ms",
				"Char:", aurora.Cyan(strconv.Itoa(player.CharacterId)),
				"Kart:", aurora.Cyan(strconv.Itoa(player.KartId)),
				"Count:", aurora.Cyan(strconv.Itoa(player.PlayerCount)))

			// Hand off to qr2 for processing
			qr2.ProcessMKWRaceResult(g.User.ProfileId, player.Pid, player.FinishTimeMs, player.CharacterId, player.KartId, player.PlayerCount)

		case "wl:mkw_race_start_time":
			serverTime := time.Now().UnixMilli()
			logging.Info(g.ModuleName,
				"Race start time:", aurora.Yellow(value),
				"Server time:", aurora.Yellow(strconv.FormatInt(serverTime, 10)))

			// Parse client timestamp
			clientTime, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				logging.Error(g.ModuleName, "Failed to parse client timestamp:", err.Error())
				return
			}

			// Store timing data in qr2 module
			qr2.StoreRaceStartTime(g.User.ProfileId, clientTime, serverTime)

		case "wl:mkw_race_finish_time":
			serverTime := time.Now().UnixMilli()
			logging.Info(g.ModuleName,
				"Race finish time:", aurora.Yellow(value),
				"Server time:", aurora.Yellow(strconv.FormatInt(serverTime, 10)))

			// Parse client timestamp
			clientTime, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				logging.Error(g.ModuleName, "Failed to parse client finish timestamp:", err.Error())
				return
			}

			// Store timing data in qr2 module
			qr2.StoreRaceFinishTime(g.User.ProfileId, clientTime, serverTime)
		}
	}
}
