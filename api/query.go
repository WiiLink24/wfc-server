package api

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"wwfc/database"
)

type QueryRequest struct {
	Secret   string `json:"secret"`
	IP       string `json:"ip"`
	DeviceID uint32 `json:"deviceID"`
	Csnum    string `json:"csnum"`
	// 0: Either, 1: No Ban, 2: Ban
	HasBan byte `json:"hasban"`
}

type QueryResponse struct {
	Users   []database.User
	Success bool
	Error   string
}

var QueryRoute = MakeRouteSpec[QueryRequest, QueryResponse](
	true,
	"/api/query",
	HandleQuery,
	http.MethodPost,
)

var (
	ErrInvalidIPFormat = errors.New("Invalid IP Format. IPs must be in the format '%d.%d.%d.%d'.")
	ErrInvalidDeviceID = errors.New("DeviceID cannot be 0.")
	ErrInvalidCsnum    = errors.New("Csnums must be less than 16 characters long and match the format '^[a-zA-Z0-9]+$'.")
	ErrInvalidHasBan   = errors.New("HasBan must be either 0 (Either), 1 (No Ban), 2 (Ban)")
	ErrEmptyParams     = errors.New("At least one of IP, Csnum, and DeviceID must be nonzero or nonempty")

	ipRegex    = regexp.MustCompile(`\d\.\d\.\d\.`)
	csnumRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
)

const QUERYBASE = `SELECT profile_id, user_id, gsbrcd, ng_device_id, email, unique_nick, firstname, lastname, has_ban, ban_reason, open_host, last_ingamesn, last_ip_address, csnum, ban_moderator, ban_reason_hidden, ban_issued, ban_expires FROM users WHERE`

func HandleQuery(req any, _ bool, _ *http.Request) (any, int, error) {
	_req := req.(QueryRequest)

	if _req.IP != "" && !ipRegex.MatchString(_req.IP) {
		return nil, http.StatusBadRequest, ErrInvalidIPFormat
	}

	if _req.Csnum != "" && (len(_req.Csnum) > 16 || !csnumRegex.MatchString(_req.Csnum)) {
		return nil, http.StatusBadRequest, ErrInvalidCsnum
	}

	if _req.HasBan != 0 && _req.HasBan != 1 && _req.HasBan != 2 {
		return nil, http.StatusBadRequest, ErrInvalidHasBan
	}

	if _req.IP == "" && _req.Csnum == "" && _req.DeviceID == 0 {
		return nil, http.StatusBadRequest, ErrEmptyParams
	}

	query := QUERYBASE

	if _req.IP != "" {
		query += fmt.Sprintf(" last_ip_address = '%s' AND", _req.IP)
	}

	if _req.DeviceID != 0 {
		query += fmt.Sprintf(" %d = ANY(ng_device_id) AND", _req.DeviceID)
	}

	if _req.Csnum != "" {
		query += fmt.Sprintf(" '%s' = ANY(csnum) AND", _req.Csnum)
	}

	if _req.HasBan == 1 {
		query += fmt.Sprintf(" has_ban = false AND")
	} else if _req.HasBan == 2 {
		query += fmt.Sprintf(" has_ban = true AND")
	}

	query = query[0 : len(query)-4]
	query += ";"

	rows, err := pool.Query(ctx, query)
	defer rows.Close()
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	res := QueryResponse{}
	res.Users = []database.User{}

	count := 0
	for rows.Next() {
		count++

		user := database.User{}

		// May be null
		var firstName *string
		var lastName *string
		var banReason *string
		var lastInGameSn *string
		var lastIPAddress *string
		var banModerator *string
		var banHiddenReason *string

		err := rows.Scan(&user.ProfileId, &user.UserId, &user.GsbrCode, &user.NgDeviceId, &user.Email, &user.UniqueNick, &firstName, &lastName, &user.Restricted, &banReason, &user.OpenHost, &lastInGameSn, &lastIPAddress, &user.Csnum, &banModerator, &banHiddenReason, &user.BanIssued, &user.BanExpires)

		if err != nil {
			return nil, http.StatusInternalServerError, err
		}

		if firstName != nil {
			user.FirstName = *firstName
		}

		if lastName != nil {
			user.LastName = *lastName
		}

		if banReason != nil {
			user.BanReason = *banReason
		}

		if lastInGameSn != nil {
			user.LastInGameSn = *lastInGameSn
		}

		if lastIPAddress != nil {
			user.LastIPAddress = *lastIPAddress
		}

		if banModerator != nil {
			user.BanModerator = *banModerator
		}

		if banHiddenReason != nil {
			user.BanReasonHidden = *banHiddenReason
		}

		res.Users = append(res.Users, user)

		// TODO: Return a count of the total number of matches, do something
		// about returing 80000 matches if the query is vague enough
	}

	return res, http.StatusOK, nil
}
