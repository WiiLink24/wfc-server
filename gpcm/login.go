package gpcm

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
	"wwfc/common"
	"wwfc/database"
	"wwfc/logging"
	"wwfc/qr2"

	"github.com/logrusorgru/aurora/v3"
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

var msPublicKey = []byte{
	0x00, 0xFD, 0x56, 0x04, 0x18, 0x2C, 0xF1, 0x75, 0x09, 0x21, 0x00, 0xC3, 0x08, 0xAE, 0x48, 0x39,
	0x91, 0x1B, 0x6F, 0x9F, 0xA1, 0xD5, 0x3A, 0x95, 0xAF, 0x08, 0x33, 0x49, 0x47, 0x2B, 0x00, 0x01,
	0x71, 0x31, 0x69, 0xB5, 0x91, 0xFF, 0xD3, 0x0C, 0xBF, 0x73, 0xDA, 0x76, 0x64, 0xBA, 0x8D, 0x0D,
	0xF9, 0x5B, 0x4D, 0x11, 0x04, 0x44, 0x64, 0x35, 0xC0, 0xED, 0xA4, 0x2F,
}

func verifySignature(moduleName string, authToken string, signature string) uint32 {
	sigBytes, err := common.Base64DwcEncoding.DecodeString(signature)
	if err != nil || len(sigBytes) != 0x144 {
		return 0
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

	if !verifyECDSA(msPublicKey, msSignature, ngCertBlobHash[:]) {
		logging.Error(moduleName, "NG cert verify failed")
		return 0
	}
	logging.Info(moduleName, "NG cert verified")

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
		logging.Error(moduleName, "AP cert verify failed")
		return 0
	}
	logging.Info(moduleName, "AP cert verified")

	authTokenHash := sha1.Sum([]byte(authToken))
	if !verifyECDSA(apPublicKey, apSignature, authTokenHash[:]) {
		logging.Error(moduleName, "Auth token signature failed")
		return 0
	}
	logging.Notice(moduleName, "Auth token signature verified; NG ID:", aurora.Cyan(fmt.Sprintf("%08x", ngId)))

	return binary.BigEndian.Uint32(ngId)
}

func (g *GameSpySession) login(command common.GameSpyCommand) {
	if g.LoggedIn {
		logging.Error(g.ModuleName, "Attempt to login twice")
		g.replyError(ErrLogin)
		return
	}

	authToken := command.OtherValues["authtoken"]
	if authToken == "" {
		g.replyError(ErrLogin)
		return
	}

	err, gamecd, issueTime, userId, gsbrcd, cfc, _, _, ingamesn, challenge, isLocalhost := common.UnmarshalNASAuthToken(authToken)
	if err != nil {
		g.replyError(ErrLogin)
		return
	}

	currentTime := time.Now()
	if issueTime.Before(currentTime.Add(-10*time.Minute)) || issueTime.After(currentTime) {
		g.replyError(ErrLoginLoginTicketExpired)
		return
	}

	payloadVer, payloadVerExists := command.OtherValues["payload_ver"]
	signature, signatureExists := command.OtherValues["wwfc_sig"]
	deviceId := uint32(0)

	if isLocalhost && !payloadVerExists && !signatureExists {
		// Players using the DNS exploit, need patching using a QR2 exploit
		// TODO: Check that the game is compatible with the DNS
		g.NeedsExploit = true
	} else {
		if !payloadVerExists || payloadVer != "2" {
			g.replyError(GPError{
				ErrorCode:   ErrLogin.ErrorCode,
				ErrorString: "The payload version is invalid.",
				Fatal:       true,
			})
			return
		}

		if !signatureExists {
			g.replyError(GPError{
				ErrorCode:   ErrLogin.ErrorCode,
				ErrorString: "Missing authentication signature.",
				Fatal:       true,
			})
			return
		}

		if deviceId = verifySignature(g.ModuleName, authToken, signature); deviceId == 0 {
			g.replyError(GPError{
				ErrorCode:   ErrLogin.ErrorCode,
				ErrorString: "The authentication signature is invalid.",
				Fatal:       true,
			})
			return
		}
	}

	response := generateResponse(g.Challenge, challenge, authToken, command.OtherValues["challenge"])
	if response != command.OtherValues["response"] {
		g.replyError(ErrLogin)
		return
	}

	proof := generateProof(g.Challenge, challenge, command.OtherValues["authtoken"], command.OtherValues["challenge"])

	cmdProfileId := uint32(0)
	if cmdProfileIdStr, exists := command.OtherValues["profileid"]; exists {
		cmdProfileId2, err := strconv.ParseUint(cmdProfileIdStr, 10, 32)
		if err != nil {
			g.replyError(GPError{
				ErrorCode:   ErrLogin.ErrorCode,
				ErrorString: "The provided profile ID is invalid.",
				Fatal:       true,
			})
			return
		}

		cmdProfileId = uint32(cmdProfileId2)
	}

	// Perform the login with the database.
	user, ok := database.LoginUserToGPCM(pool, ctx, userId, gsbrcd, cmdProfileId, deviceId)
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
	otherSession, exists := sessions[g.User.ProfileId]
	if exists {
		otherSession.replyError(ErrForcedDisconnect)
		otherSession.Conn.Close()

		for i := 0; ; i++ {
			mutex.Unlock()
			time.Sleep(300 * time.Millisecond)
			mutex.Lock()

			if _, exists = sessions[g.User.ProfileId]; !exists {
				break
			}

			// Give up after 6 seconds
			if i >= 20 {
				mutex.Unlock()
				logging.Error(g.ModuleName, "Failed to disconnect other session")
				g.replyError(ErrForcedDisconnect)
				return
			}
		}
	}
	sessions[g.User.ProfileId] = g
	mutex.Unlock()

	g.AuthToken = authToken
	g.LoginTicket = common.MarshalGPCMLoginTicket(g.User.ProfileId)
	g.SessionKey = rand.Int31n(290000000) + 10000000
	g.GameCode = gamecd
	g.InGameName = ingamesn

	g.DeviceAuthenticated = !g.NeedsExploit
	g.LoggedIn = true
	g.ModuleName = "GPCM:" + strconv.FormatInt(int64(g.User.ProfileId), 10)
	g.ModuleName += "/" + common.CalcFriendCodeString(g.User.ProfileId, "RMCJ")

	// Notify QR2 of the login
	qr2.Login(g.User.ProfileId, gamecd, ingamesn, cfc, g.Conn.RemoteAddr().String(), g.NeedsExploit, g.DeviceAuthenticated)

	payload := common.CreateGameSpyMessage(common.GameSpyCommand{
		Command:      "lc",
		CommandValue: "2",
		OtherValues: map[string]string{
			"sesskey":    strconv.FormatInt(int64(g.SessionKey), 10),
			"proof":      proof,
			"userid":     strconv.FormatUint(g.User.UserId, 10),
			"profileid":  strconv.FormatUint(uint64(g.User.ProfileId), 10),
			"uniquenick": g.User.UniqueNick,
			"lt":         g.LoginTicket,
			"id":         command.OtherValues["id"],
		},
	})

	g.Conn.Write([]byte(payload))

	g.sendFriendRequests()
}

func (g *GameSpySession) exLogin(command common.GameSpyCommand) {
	if !g.LoggedIn {
		logging.Warn(g.ModuleName, "Ignoring exlogin before login")
		return
	}

	payloadVer, payloadVerExists := command.OtherValues["payload_ver"]
	signature, signatureExists := command.OtherValues["wwfc_sig"]
	deviceId := uint32(0)

	if !payloadVerExists || payloadVer != "2" {
		g.replyError(GPError{
			ErrorCode:   ErrLogin.ErrorCode,
			ErrorString: "The payload version is invalid.",
			Fatal:       true,
		})
		return
	}

	if !signatureExists {
		g.replyError(GPError{
			ErrorCode:   ErrLogin.ErrorCode,
			ErrorString: "Missing authentication signature.",
			Fatal:       true,
		})
		return
	}

	if deviceId = verifySignature(g.ModuleName, g.AuthToken, signature); deviceId == 0 {
		g.replyError(GPError{
			ErrorCode:   ErrLogin.ErrorCode,
			ErrorString: "The authentication signature is invalid.",
			Fatal:       true,
		})
		return
	}

	g.DeviceAuthenticated = true
	qr2.SetDeviceAuthenticated(g.User.ProfileId)
}

func IsLoggedIn(profileID uint32) bool {
	mutex.Lock()
	defer mutex.Unlock()

	session, exists := sessions[profileID]
	return exists && session.LoggedIn
}
