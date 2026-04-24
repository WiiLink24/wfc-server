package api

import (
	"net/http"
	"wwfc/gpcm"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

type KickRequestSpec struct {
	AuthInfo
	Reason    string `json:"reason"`
	ProfileID uint32 `json:"pid"`
}

func HandleKick(w http.ResponseWriter, r *http.Request) {
	var req KickRequestSpec
	if err := parsePost(r, w, &req, RoleModerator); err != nil {
		return
	}

	if req.ProfileID == 0 {
		replyError(w, http.StatusBadRequest, APIErrorInvalidProfileID)
		return
	}

	if req.Reason == "" {
		replyError(w, http.StatusBadRequest, APIErrorInvalidReason)
		return
	}

	replyOK(w, nil)

	gpcm.KickPlayerCustomMessage(req.ProfileID, req.Reason, gpcm.WWFCMsgKickedCustom)

	logging.Event("profile_kicked", map[string]any{
		"profile_id": req.ProfileID,
		"reason":     req.Reason,
	})

	logging.Notice("API:admin", "Kick:", aurora.Cyan(req.ProfileID), "Reason:", aurora.BrightCyan(req.Reason))
}
