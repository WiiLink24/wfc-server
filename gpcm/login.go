package gpcm

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"github.com/jackc/pgx/v4/pgxpool"
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
	str += "                                                "
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

func login(pool *pgxpool.Pool, ctx context.Context, command common.GameSpyCommand, challenge string) string {
	// TODO: Validate login token with one in database
	authToken := command.OtherValues["authtoken"]
	response := generateResponse(challenge, "0qUekMb4", authToken, command.OtherValues["challenge"])
	if response != command.OtherValues["response"] {
		log.Fatalf("i hate my life")
	}

	proof := generateProof(challenge, "0qUekMb4", command.OtherValues["authtoken"], command.OtherValues["challenge"])

	// Perform the login with the database.
	// TODO: Check valid result
	user, _ := database.LoginUserToGPCM(pool, ctx, authToken)
	loginTicket := strings.Replace(base64.StdEncoding.EncodeToString([]byte(common.RandomString(16))), "=", "_", -1)
	// TODO: Remove in favour of proper thread safe holding
	userId = user.UserId
	// Now initiate the session
	_ = database.CreateSession(pool, ctx, user.ProfileId, loginTicket)

	return common.CreateGameSpyMessage(common.GameSpyCommand{
		Command:      "lc",
		CommandValue: "2",
		OtherValues: map[string]string{
			"sesskey":    "199714190",
			"proof":      proof,
			"userid":     strconv.FormatInt(user.UserId, 10),
			"profileid":  strconv.FormatInt(user.ProfileId, 10),
			"uniquenick": user.UniqueNick,
			"lt":         loginTicket,
			"id":         command.OtherValues["id"],
		},
	})
}
