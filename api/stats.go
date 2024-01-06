package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"wwfc/common"
	"wwfc/qr2"
)

type Stats struct {
	OnlinePlayerCount int `json:"online"`
	ActivePlayerCount int `json:"active"`
	GroupCount        int `json:"groups"`
}

func HandleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	u, err := url.Parse(r.URL.String())
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	games := query["game"]

	stats := map[string]Stats{}

	servers := qr2.GetSessionServers()
	groups := qr2.GetGroups([]string{}, []string{})

	globalStats := Stats{
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
			gameStats = Stats{
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

	jsonData, err := json.Marshal(stats)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
	w.Write(jsonData)
}
