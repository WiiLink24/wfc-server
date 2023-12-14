package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
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

	gameName := query.Get("gamename")
	groups := qr2.GetGroups(gameName)

	for _, group := range groups {
		for i, player := range group.Players {
			filtered := map[string]string{}

			filtered["pid"] = player["dwc_pid"]
			filtered["name"] = player["+ingamesn"]

			if player["gamename"] == "mariokartwii" {
				filtered["ev"] = player["ev"]
				filtered["eb"] = player["eb"]
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
