package api

import (
	"net/http"
	"wwfc/qr2"
)

func HandleGroups(w http.ResponseWriter, r *http.Request) {
	query, err := parseGet(r, w, RoleNone)
	if err != nil {
		return
	}

	groups := qr2.GetGroups(query["game"], query["id"], true)

	if len(groups) == 0 {
		// I would return No Content, but here is compatibility
		replyOK(w, "[]")
		return
	}

	replyOK(w, groups)
}
