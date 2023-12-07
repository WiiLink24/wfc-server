package gpcm

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/rand"
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

func verifySignature(authToken string, signature string) bool {
	sigBytes, err := common.Base64DwcEncoding.DecodeString(signature)
	if err != nil || len(sigBytes) != 0x144 {
		return false
	}

	ngId := sigBytes[0x000:0x004]
	ngTimestamp := sigBytes[0x004:0x008]
	caId := sigBytes[0x008:0x00C]
	msId := sigBytes[0x00C:0x010]
	apId := sigBytes[0x010:0x018]
	msSignature := sigBytes[0x018:0x054]
	ngPublicKey := sigBytes[0x054:0x090]
	ngSignature := sigBytes[0x090:0x0CC]
	apPublicKey := sigBytes[0x0CC:0x108]
	apSignature := sigBytes[0x108:0x144]

	ngIssuer := fmt.Sprintf("Root-CA%02x%02x%02x%02x-MS%02x%02x%02x%02x", caId[0], caId[1], caId[2], caId[3], msId[0], msId[1], msId[2], msId[3])
	ngName := fmt.Sprintf("NG%02x%02x%02x%02x", ngId[0], ngId[1], ngId[2], ngId[3])

	ngCertBlob := []byte(ngIssuer)
	ngCertBlob = append(ngCertBlob, make([]byte, 0x40-len(ngIssuer))...)
	ngCertBlob = append(ngCertBlob, 0x00, 0x00, 0x00, 0x02)
	ngCertBlob = append(ngCertBlob, []byte(ngName)...)
	ngCertBlob = append(ngCertBlob, make([]byte, 0x40-len(ngName))...)
	ngCertBlob = append(ngCertBlob, ngTimestamp...)
	ngCertBlob = append(ngCertBlob, ngPublicKey...)
	ngCertBlob = append(ngCertBlob, make([]byte, 0x3C)...)
	ngCertBlobHash := sha1.Sum(ngCertBlob)

	msPublicKey := []byte{0x00, 0xFD, 0x56, 0x04, 0x18, 0x2C, 0xF1, 0x75, 0x09, 0x21, 0x00, 0xC3, 0x08, 0xAE, 0x48, 0x39, 0x91, 0x1B, 0x6F, 0x9F, 0xA1, 0xD5, 0x3A, 0x95, 0xAF, 0x08, 0x33, 0x49, 0x47, 0x2B, 0x00, 0x01, 0x71, 0x31, 0x69, 0xB5, 0x91, 0xFF, 0xD3, 0x0C, 0xBF, 0x73, 0xDA, 0x76, 0x64, 0xBA, 0x8D, 0x0D, 0xF9, 0x5B, 0x4D, 0x11, 0x04, 0x44, 0x64, 0x35, 0xC0, 0xED, 0xA4, 0x2F}

	if !verifyECDSA(msPublicKey, msSignature, ngCertBlobHash[:]) {
		logging.Error("GPCM", "NG cert verify failed")
		return false
	}
	logging.Info("GPCM", "NG cert verified")

	apIssuer := ngIssuer + "-" + ngName
	apName := fmt.Sprintf("AP%02x%02x%02x%02x%02x%02x%02x%02x", apId[0], apId[1], apId[2], apId[3], apId[4], apId[5], apId[6], apId[7])

	apCertBlob := []byte(apIssuer)
	apCertBlob = append(apCertBlob, make([]byte, 0x40-len(apIssuer))...)
	apCertBlob = append(apCertBlob, 0x00, 0x00, 0x00, 0x02)
	apCertBlob = append(apCertBlob, []byte(apName)...)
	apCertBlob = append(apCertBlob, make([]byte, 0x40-len(apName))...)
	apCertBlob = append(apCertBlob, 0x00, 0x00, 0x00, 0x00)
	apCertBlob = append(apCertBlob, apPublicKey...)
	apCertBlob = append(apCertBlob, make([]byte, 0x3C)...)
	apCertBlobHash := sha1.Sum(apCertBlob)

	if !verifyECDSA(ngPublicKey, ngSignature, apCertBlobHash[:]) {
		logging.Error("GPCM", "AP cert verify failed")
		return false
	}
	logging.Info("GPCM", "AP cert verified")

	authTokenHash := sha1.Sum([]byte(authToken))
	if !verifyECDSA(apPublicKey, apSignature, authTokenHash[:]) {
		logging.Error("GPCM", "Auth token signature failed")
		return false
	}
	logging.Notice("GPCM", "Auth token signature verified")

	return true
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
	if authToken == "" {
		g.replyError(ErrLogin)
		return
	}

	signature, exists := command.OtherValues["wwfc_sig"]
	if exists {
		// TODO: This is still in testing so it's only checked if it exists
		if !verifySignature(authToken, signature) {
			g.replyError(GPError{
				ErrorCode:   ErrLogin.ErrorCode,
				ErrorString: "The authentication signature is invalid.",
				Fatal:       true,
			})
			return
		}
	}

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
	_, exists = sessions[g.User.ProfileId]
	if exists {
		mutex.Unlock()
		// Original GPCM would've force kicked the other logged in client,
		// but we just kick this client
		g.replyError(ErrForcedDisconnect)
		return
	}
	sessions[g.User.ProfileId] = g
	mutex.Unlock()

	g.LoginTicket = strings.Replace(base64.StdEncoding.EncodeToString([]byte(common.RandomString(16))), "=", "_", -1)
	// Now initiate the session
	_ = database.CreateSession(pool, ctx, g.User.ProfileId, g.LoginTicket)
	g.SessionKey = rand.Int31n(290000000) + 10000000

	g.LoggedIn = true
	g.ModuleName = "GPCM:" + strconv.FormatInt(int64(g.User.ProfileId), 10)
	g.ModuleName += "/" + common.CalcFriendCodeString(g.User.ProfileId, "RMCJ")

	payload := common.CreateGameSpyMessage(common.GameSpyCommand{
		Command:      "lc",
		CommandValue: "2",
		OtherValues: map[string]string{
			"sesskey":    strconv.FormatInt(int64(g.SessionKey), 10),
			"proof":      proof,
			"userid":     strconv.FormatInt(g.User.UserId, 10),
			"profileid":  strconv.FormatInt(int64(g.User.ProfileId), 10),
			"uniquenick": g.User.UniqueNick,
			"lt":         g.LoginTicket,
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
