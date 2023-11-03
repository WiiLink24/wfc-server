package nas

import (
	"strconv"
	"wwfc/database"
	"wwfc/logging"
)

// TODO: Generate and store in database!!!
const Challenge = "0qUekMb4"

func login(r *Response, fields map[string]string) map[string]string {
	moduleName := "NAS:" + r.request.RemoteAddr

	param := map[string]string{
		"retry":   "0",
		"locator": "gamespy.com",
	}

	strUserId, ok := fields["userid"]
	if !ok {
		logging.Error(moduleName, "No userid in form")
		param["returncd"] = "103"
		return param
	}

	userId, err := strconv.ParseInt(strUserId, 10, 64)
	if err != nil {
		logging.Error(moduleName, "Invalid userid string in form")
		param["returncd"] = "103"
		return param
	}

	gsbrcd, ok := fields["gsbrcd"]
	if !ok {
		logging.Error(moduleName, "No gsbrcd in form")
		param["returncd"] = "103"
		return param
	}

	authToken := database.GenerateAuthToken(pool, ctx, userId, string(gsbrcd))

	param["returncd"] = "001"
	param["challenge"] = Challenge
	param["token"] = authToken
	return param
}
