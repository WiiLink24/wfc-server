package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"
	"wwfc/common"
)

type BanInfoResponseSpec struct {
	ProfileID  uint32    `json:"pid"`
	FriendCode string    `json:"fc,omitempty"`
	InGameName string    `json:"name,omitempty"`
	Reason     string    `json:"reason,omitempty"`
	TOS        bool      `json:"tos"`
	Issued     time.Time `json:"issued"`
	Expires    time.Time `json:"expires"`
}

func HandleBanInfo(w http.ResponseWriter, r *http.Request) {
	query, err := parseGet(r, w, RoleNone)
	if err != nil {
		return
	}

	search := query.Get("q")
	if search == "" {
		replyError(w, http.StatusBadRequest, APIErrorInvalidBanQuery)
		return
	}

	search = strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(search, " ", ""), "-", ""))

	profileId := uint32(0)
	ngDeviceId := uint32(0)
	if strings.HasPrefix(search, "NG") {
		ngId, err := strconv.ParseUint(search[2:], 16, 32)
		if err != nil {
			replyError(w, http.StatusBadRequest, APIErrorInvalidBanQuery)
			return
		}
		ngDeviceId = uint32(ngId)
	} else {
		pId, err := strconv.ParseUint(search, 10, 64)
		if err != nil {
			replyError(w, http.StatusBadRequest, APIErrorInvalidBanQuery)
			return
		}
		// Truncate to 32 bits as that's how friend codes work
		profileId = uint32(pId)
	}

	tos, issued, expires, reason, bannedProfileId, gsbrCode, inGameName, err := db.SearchUserBan(profileId, ngDeviceId, "", "")
	if err != nil {
		replyError(w, http.StatusOK, APIErrorBanNotFound)
		return
	}

	if bannedProfileId == 0 {
		replyError(w, http.StatusOK, APIErrorBanNotFound)
		return
	}

	fc := common.CalcFriendCodeString(bannedProfileId, gsbrCode)
	replyOK(w, BanInfoResponseSpec{
		ProfileID:  bannedProfileId,
		FriendCode: fc,
		InGameName: inGameName,
		Reason:     reason,
		TOS:        tos,
		Issued:     issued,
		Expires:    expires,
	})
}
