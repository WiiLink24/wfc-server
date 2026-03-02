package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"wwfc/qr2"
)

type RaceResultInfo struct {
	Results map[int][]qr2.RaceResult `json:"results"`
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

	results := qr2.GetRaceResultsForGroup(query["id"][0])
	if results == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var jsonData []byte
	if len(results) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	} else {
		jsonData, err = json.Marshal(RaceResultInfo{
			Results: results,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
	w.Write(jsonData)
}
