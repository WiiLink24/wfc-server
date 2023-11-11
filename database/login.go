package database

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/logrusorgru/aurora/v3"
	"os"
	"strconv"
	"wwfc/common"
	"wwfc/logging"
)

const (
	DoesAuthTokenExist = `SELECT EXISTS(SELECT 1 FROM logins WHERE auth_token = $1)`
	DoesNASUserExist   = `SELECT EXISTS(SELECT 1 FROM logins WHERE user_id = $1 AND gsbrcd = $2)`
	UpdateUserLogin    = `UPDATE logins SET auth_token = $1, challenge = $2 WHERE user_id = $3 AND gsbrcd = $4`
	InsertUserLogin    = `INSERT INTO logins (auth_token, user_id, gsbrcd, challenge) VALUES ($1, $2, $3, $4)`
	GetNASUserLogin    = `SELECT user_id, gsbrcd FROM logins WHERE auth_token = $1 LIMIT 1`
	GetNASChallenge    = `SELECT challenge FROM logins WHERE auth_token = $1`
)

var salt []byte

// GenerateAuthToken generates and stores the auth token for this user as well as a challenge.
func GenerateAuthToken(pool *pgxpool.Pool, ctx context.Context, userId int64, gsbrcd string) (string, string) {
	var userExists bool
	err := pool.QueryRow(ctx, DoesNASUserExist, userId, gsbrcd).Scan(&userExists)
	if err != nil {
		panic(err)
	}

	authToken := "NDS" + common.RandomString(80)
	for {
		// We must make sure that the auth token doesn't exist before attempting to insert it into the database.
		var tokenExists bool
		err := pool.QueryRow(ctx, DoesAuthTokenExist, authToken).Scan(&tokenExists)
		if err != nil {
			panic(err)
		}

		if !tokenExists {
			break
		}

		authToken = "NDS" + common.RandomString(80)
	}

	challenge := common.RandomString(8)
	if userExists {
		// UPDATE rather than INSERT
		_, err = pool.Exec(ctx, UpdateUserLogin, authToken, challenge, userId, gsbrcd)
		if err != nil {
			panic(err)
		}
	} else {
		_, err = pool.Exec(ctx, InsertUserLogin, authToken, userId, gsbrcd, challenge)
		if err != nil {
			panic(err)
		}
	}

	return authToken, challenge
}

func GetNASLogin(pool *pgxpool.Pool, ctx context.Context, authToken string) (int64, string) {
	var userId int64
	var gsbrcd string
	err := pool.QueryRow(ctx, GetNASUserLogin, authToken).Scan(&userId, &gsbrcd)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, ""
		} else {
			panic(err)
		}
	}

	return userId, gsbrcd
}

func LoginUserToGPCM(pool *pgxpool.Pool, ctx context.Context, authToken string) (User, bool) {
	// Make sure salt is loaded
	if salt == nil {
		var err error
		salt, err = os.ReadFile("salt.bin")
		if err != nil {
			panic(err)
		}
	}

	// Query login table with the auth token.
	userId, gsbrcd := GetNASLogin(pool, ctx, authToken)
	if userId == 0 {
		logging.Error("DATABASE", "Invalid auth token:", aurora.Cyan(authToken))
		return User{}, false
	}

	var exists bool
	err := pool.QueryRow(ctx, DoesUserExist, userId, gsbrcd).Scan(&exists)
	if err != nil {
		panic(err)
	}

	uniqueNickname := common.Base32Encode(userId) + gsbrcd
	password := sha512.Sum512(append(salt, []byte(gsbrcd)...))

	user := User{
		UserId:     userId,
		GsbrCode:   gsbrcd,
		Password:   hex.EncodeToString(password[:]),
		Email:      uniqueNickname + "@nds",
		UniqueNick: uniqueNickname,
	}

	if !exists {
		// Create the GPCM account
		user.CreateUser(pool, ctx)

		logging.Notice("DATABASE", "Created new GPCM user:", aurora.Cyan(strconv.FormatInt(user.UserId, 10)), aurora.Cyan(user.GsbrCode), "-", aurora.Cyan(strconv.FormatInt(int64(user.ProfileId), 10)))
	} else {
		err := pool.QueryRow(ctx, GetUserProfileID, userId, gsbrcd).Scan(&user.ProfileId)
		if err != nil {
			panic(err)
		}

		logging.Notice("DATABASE", "Log in GPCM user:", aurora.Cyan(strconv.FormatInt(user.UserId, 10)), aurora.Cyan(user.GsbrCode), "-", aurora.Cyan(strconv.FormatInt(int64(user.ProfileId), 10)))
	}

	return user, true
}

func GetChallenge(pool *pgxpool.Pool, ctx context.Context, authToken string) string {
	var challenge string
	err := pool.QueryRow(ctx, GetNASChallenge, authToken).Scan(&challenge)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Invalid auth token
			return ""
		} else {
			panic(err)
		}
	}

	return challenge
}
