package gpcm

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"log"
	"strconv"
	"strings"
	"wwfc/common"
	"wwfc/database"
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
		log.Fatalf("Attempt to login twice")
	}

	authToken := command.OtherValues["authtoken"]
	challenge := database.GetChallenge(pool, ctx, authToken)
	if challenge == "" {
		// TODO: Error out
		log.Fatalf("Invalid auth token")
	}

	response := generateResponse(g.Challenge, challenge, authToken, command.OtherValues["challenge"])
	if response != command.OtherValues["response"] {
		// TODO: Return an error
		log.Fatalf("response mismatch")
	}

	proof := generateProof(g.Challenge, challenge, command.OtherValues["authtoken"], command.OtherValues["challenge"])

	// Perform the login with the database.
	// TODO: Check valid result
	user, ok := database.LoginUserToGPCM(pool, ctx, authToken)
	if !ok {
		// TODO: Return an error
		log.Fatalf("GPCM login error")
	}
	g.User = user

	// Check to see if a session is already open with this profile ID
	// TODO: Test this
	mutex.Lock()
	_, exists := sessions[g.User.ProfileId]
	if exists {
		mutex.Unlock()
		// TODO: Return an error
		log.Fatalf("Session with profile ID %d is already open", g.User.ProfileId)
	}
	sessions[g.User.ProfileId] = g
	mutex.Unlock()

	loginTicket := strings.Replace(base64.StdEncoding.EncodeToString([]byte(common.RandomString(16))), "=", "_", -1)
	// Now initiate the session
	_ = database.CreateSession(pool, ctx, g.User.ProfileId, loginTicket)

	g.LoggedIn = true
	g.ModuleName += ":" + strconv.FormatInt(int64(g.User.ProfileId), 10)
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
