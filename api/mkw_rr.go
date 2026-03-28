package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"wwfc/qr2"
)

type RaceResultInfo struct {
	Players map[uint32]qr2.PlayerInfo `json:"players"`
	Results map[int][]qr2.RaceResult  `json:"results"`
}

func HandleMKWRR(w http.ResponseWriter, r *http.Request) {
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

	groupNames := query["id"]
	if len(groupNames) != 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	groupName := query["id"][0]
	players := qr2.GetRacePlayersForGroup(groupName)
	results := qr2.GetRaceResultsForGroup(groupName)
	if results == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if players == nil {
		players = map[uint32]qr2.PlayerInfo{}
	}

	var jsonData []byte
	jsonData, err = json.Marshal(RaceResultInfo{
		Players: players,
		Results: results,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
	w.Write(jsonData)
}
