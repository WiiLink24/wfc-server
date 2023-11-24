package gpcm

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"strconv"
	"strings"
	"wwfc/common"
	"wwfc/database"
	"wwfc/logging"
)

func generateResponse(gpcmChallenge, nasChallenge, authToken, clientChallenge string) string {
	hasher := md5.New()
	hasher.Write([]byte(nasChallenge))
	str := hex.EncodeToString(hasher.Sum(nil))
	str += strings.Repeat(" ", 48)
	str += authToken
	str += clientChallenge
	str += gpcmChallenge
	str += hex.EncodeToString(hasher.Sum(nil))

	_hasher := md5.New()
	_hasher.Write([]byte(str))
	return hex.EncodeToString(_hasher.Sum(nil))
}

func generateProof(gpcmChallenge, nasChallenge, authToken, clientChallenge string) string {
	return generateResponse(clientChallenge, nasChallenge, authToken, gpcmChallenge)
}

func (g *GameSpySession) login(command common.GameSpyCommand) {
	if g.LoggedIn {
		logging.Error(g.ModuleName, "Attempt to login twice")
		g.replyError(ErrLogin)
		return
	}

	if command.OtherValues["payload_ver"] != "1" {
		g.replyError(GPError{
			ErrorCode:   ErrLogin.ErrorCode,
			ErrorString: "The payload version is invalid.",
			Fatal:       true,
		})
		return
	}

	authToken := command.OtherValues["authtoken"]
	challenge := database.GetChallenge(pool, ctx, authToken)
	if challenge == "" {
		g.replyError(ErrLogin)
		return
	}

	response := generateResponse(g.Challenge, challenge, authToken, command.OtherValues["challenge"])
	if response != command.OtherValues["response"] {
		g.replyError(ErrLogin)
		return
	}

	proof := generateProof(g.Challenge, challenge, command.OtherValues["authtoken"], command.OtherValues["challenge"])

	// Perform the login with the database.
	user, ok := database.LoginUserToGPCM(pool, ctx, authToken)
	if !ok {
		// There was an error logging in to the GP backend.
		g.replyError(ErrLogin)
		return
	}
	g.User = user

	g.ModuleName = "GPCM:" + strconv.FormatInt(int64(g.User.ProfileId), 10) + "*"
	g.ModuleName += "/" + common.CalcFriendCodeString(g.User.ProfileId, "RMCJ") + "*"

	// Check to see if a session is already open with this profile ID
	mutex.Lock()
	_, exists := sessions[g.User.ProfileId]
	if exists {
		mutex.Unlock()
		// Original GPCM would've force kicked the other logged in client,
		// but we just kick this client
		g.replyError(ErrForcedDisconnect)
		return
	}
	sessions[g.User.ProfileId] = g
	mutex.Unlock()

	loginTicket := strings.Replace(base64.StdEncoding.EncodeToString([]byte(common.RandomString(16))), "=", "_", -1)
	// Now initiate the session
	_ = database.CreateSession(pool, ctx, g.User.ProfileId, loginTicket)

	g.LoggedIn = true
	g.ModuleName = "GPCM:" + strconv.FormatInt(int64(g.User.ProfileId), 10)
	g.ModuleName += "/" + common.CalcFriendCodeString(g.User.ProfileId, "RMCJ")

	payload := common.CreateGameSpyMessage(common.GameSpyCommand{
		Command:      "lc",
		CommandValue: "2",
		OtherValues: map[string]string{
			"sesskey":    "199714190",
			"proof":      proof,
			"userid":     strconv.FormatInt(g.User.UserId, 10),
			"profileid":  strconv.FormatInt(int64(g.User.ProfileId), 10),
			"uniquenick": g.User.UniqueNick,
			"lt":         loginTicket,
			"id":         command.OtherValues["id"],
		},
	})

	g.Conn.Write([]byte(payload))
}

func IsLoggedIn(profileID uint32) bool {
	mutex.Lock()
	defer mutex.Unlock()

	session, exists := sessions[profileID]
	return exists && session.LoggedIn
}

func GetSessionIP(profileID uint32) string {
	mutex.Lock()
	defer mutex.Unlock()

	session, exists := sessions[profileID]
	if exists && session.LoggedIn {
		return session.Conn.RemoteAddr().String()
	}

	return ""
}
