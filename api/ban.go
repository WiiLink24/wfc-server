package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"
	"wwfc/database"
	"wwfc/gpcm"
)

func HandleBan(w http.ResponseWriter, r *http.Request) {
	errorString, ip := handleBanImpl(w, r)
	if errorString != "" {
		jsonData, _ := json.Marshal(map[string]string{"error": errorString})
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
		w.Write(jsonData)
	} else {
		jsonData, _ := json.Marshal(map[string]string{"success": "true", "ip": ip})
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
		w.Write(jsonData)
	}
}

func handleBanImpl(w http.ResponseWriter, r *http.Request) (string, string) {
	// TODO: Actual authentication rather than a fixed secret
	// TODO: Use POST instead of GET

	u, err := url.Parse(r.URL.String())
	if err != nil {
		return "Bad request", ""
	}

	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return "Bad request", ""
	}

	if apiSecret == "" || query.Get("secret") != apiSecret {
		return "Invalid API secret", ""
	}

	pidStr := query.Get("pid")
	if pidStr == "" {
		return "Missing pid in request", ""
	}

	pid, err := strconv.ParseUint(pidStr, 10, 32)
	if err != nil {
		return "Invalid pid", ""
	}

	tosStr := query.Get("tos")
	if tosStr == "" {
		return "Missing tos in request", ""
	}

	tos, err := strconv.ParseBool(tosStr)
	if err != nil {
		return "Invalid tos", ""
	}

	minutes := uint64(0)
	if query.Get("minutes") != "" {
		minutesStr := query.Get("minutes")
		minutes, err = strconv.ParseUint(minutesStr, 10, 32)
		if err != nil {
			return "Invalid minutes", ""
		}
	}

	hours := uint64(0)
	if query.Get("hours") != "" {
		hoursStr := query.Get("hours")
		hours, err = strconv.ParseUint(hoursStr, 10, 32)
		if err != nil {
			return "Invalid hours", ""
		}
	}

	days := uint64(0)
	if query.Get("days") != "" {
		daysStr := query.Get("days")
		days, err = strconv.ParseUint(daysStr, 10, 32)
		if err != nil {
			return "Invalid days", ""
		}
	}

	reason := query.Get("reason")
	if "reason" == "" {
		return "Missing ban reason", ""
	}

	// reason_hidden is optional
	reasonHidden := query.Get("reason_hidden")

	moderator := query.Get("moderator")
	if "moderator" == "" {
		moderator = "admin"
	}

	minutes = days*24*60 + hours*60 + minutes
	if minutes == 0 {
		return "Missing ban length", ""
	}

	length := time.Duration(minutes) * time.Minute

	if !database.BanUser(pool, ctx, uint32(pid), tos, length, reason, reasonHidden, moderator) {
		return "Failed to ban user", ""
	}

	if tos {
		gpcm.KickPlayer(uint32(pid), "banned")
	} else {
		gpcm.KickPlayer(uint32(pid), "restricted")
	}

	ip := database.GetUserIP(pool, ctx, uint32(pid))

	return "", ip
}
