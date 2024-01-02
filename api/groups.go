package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"wwfc/common"
	"wwfc/qr2"
)

func HandleGroups(w http.ResponseWriter, r *http.Request) {
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

	gameName := query.Get("game")
	groups := qr2.GetGroups(gameName)

	for _, group := range groups {
		for i, player := range group.Players {
			filtered := map[string]string{}

			filtered["count"] = player["+localplayers"]
			filtered["pid"] = player["dwc_pid"]
			filtered["name"] = player["+ingamesn"]

			if player["gamename"] == "mariokartwii" {
				filtered["ev"] = player["ev"]
				filtered["eb"] = player["eb"]
				pid, err := strconv.ParseUint(player["dwc_pid"], 10, 32)
				if err == nil {
					filtered["fc"] = common.CalcFriendCodeString(uint32(pid), "RMCJ")
				}
			}

			group.Players[i] = filtered
		}
	}

	jsonData, err := json.Marshal(groups)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
	w.Write(jsonData)
}
