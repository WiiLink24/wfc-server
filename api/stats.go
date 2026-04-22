package api

import (
	"net/http"
	"wwfc/common"
	"wwfc/qr2"
)

type StatsResponseSpec struct {
	OnlinePlayerCount int `json:"online"`
	ActivePlayerCount int `json:"active"`
	GroupCount        int `json:"groups"`
}

func HandleStats(w http.ResponseWriter, r *http.Request) {
	query, err := parseGet(r, w, RoleNone)
	if err != nil {
		return
	}

	games := query["game"]

	stats := map[string]StatsResponseSpec{}

	servers := qr2.GetSessionServers()
	groups := qr2.GetGroups([]string{}, []string{}, false)

	globalStats := StatsResponseSpec{
		OnlinePlayerCount: len(servers),
		ActivePlayerCount: 0,
		GroupCount:        len(groups),
	}

	for _, server := range servers {
		gameName := server["gamename"]

		if server["+joinindex"] != "" {
			globalStats.ActivePlayerCount += 1
		}

		if len(games) > 0 && !common.StringInSlice(gameName, games) {
			continue
		}

		gameStats, exists := stats[gameName]
		if !exists {
			gameStats = StatsResponseSpec{
				OnlinePlayerCount: 0,
				ActivePlayerCount: 0,
				GroupCount:        0,
			}

			for _, group := range groups {
				if group.GameName == gameName {
					gameStats.GroupCount += 1
				}
			}
		}

		gameStats.OnlinePlayerCount += 1
		if server["+joinindex"] != "" {
			gameStats.ActivePlayerCount += 1
		}

		stats[gameName] = gameStats
	}

	stats["global"] = globalStats
	replyOK(w, stats)
}
