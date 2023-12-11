package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"wwfc/qr2"
)

func HandleGroups(w http.ResponseWriter, r *http.Request) {
	u, err := url.Parse(r.URL.String())
	if err != nil {
		panic(err)
	}

	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		panic(err)
	}

	groups := qr2.GetGroups(append(query["gamename"], "")[0])

	for _, group := range groups {
		for i, player := range group.Players {
			filtered := map[string]string{}

			filtered["dwc_pid"] = player["dwc_pid"]
			filtered["ingamename"] = player["ingamename"]

			if player["gamename"] == "mariokartwii" {
				filtered["ev"] = player["ev"]
				filtered["eb"] = player["eb"]
			}

			group.Players[i] = filtered
		}
	}

	jsonData, err := json.Marshal(groups)
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "text/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
	w.Write(jsonData)
}
