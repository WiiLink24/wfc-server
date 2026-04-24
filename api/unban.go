package api

import (
	"net/http"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

type UnbanRequestSpec struct {
	AuthInfo
	ProfileID uint32 `json:"pid"`
}

func HandleUnban(w http.ResponseWriter, r *http.Request) {
	var req UnbanRequestSpec
	if err := parsePost(r, w, &req, RoleModerator); err != nil {
		return
	}

	if req.ProfileID == 0 {
		replyError(w, http.StatusBadRequest, APIErrorInvalidProfileID)
		return
	}

	if !db.UnbanUser(req.ProfileID) {
		replyError(w, http.StatusInternalServerError, APIErrorUnbanFailed)
		return
	}

	replyOK(w, nil)

	logging.Event("profile_unbanned", map[string]any{
		"profile_id": req.ProfileID,
	})

	logging.Notice("API:admin", "Unban:", aurora.Cyan(req.ProfileID))
}
