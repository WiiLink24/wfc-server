package database

import (
	"context"
	"database/sql"
	"wwfc/common"
	"wwfc/logging"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/logrusorgru/aurora/v3"
)

const (
	DoesAuthTokenExist = `SELECT EXISTS(SELECT 1 FROM logins WHERE auth_token = $1)`
	DoesNASUserExist   = `SELECT EXISTS(SELECT 1 FROM logins WHERE user_id = $1 AND gsbrcd = $2)`
	UpdateUserLogin    = `UPDATE logins SET auth_token = $1 WHERE user_id = $2 AND gsbrcd = $2`
	InsertUserLogin    = `INSERT INTO logins (auth_token, user_id, gsbrcd) VALUES ($1, $2, $3)`
	GetNASUserLogin    = `SELECT user_id, gsbrcd FROM logins WHERE auth_token = $1 LIMIT 1`
	GetUserAuthToken   = `SELECT auth_token FROM logins WHERE user_id = $1 AND gsbrcd = $2`
)

func GenerateAuthToken(pool *pgxpool.Pool, ctx context.Context, userId int, gsbrcd string) string {
	var authToken string
	err := pool.QueryRow(ctx, GetUserAuthToken, userId, gsbrcd).Scan(&authToken)

	if err == nil {
		// Temporary(?) workaround for multiple sessions with the same user ID (i.e. multiple Dolphin instances).
		// Just don't change the user's auth token... ever.
		// TODO: What do we actually do here? Do we even care about proper authentication at this stage?
		return authToken
	}

	exists := false
	authToken = "NDS" + common.RandomString(80)
	for {
		// We must make sure that the auth token doesn't exist before attempting to insert it into the database.
		var exists bool
		err := pool.QueryRow(ctx, DoesAuthTokenExist, authToken).Scan(&exists)
		if err != nil {
			panic(err)
		}

		if !exists {
			break
		}

		authToken = "NDS" + common.RandomString(80)
	}

	if exists {
		// UPDATE rather than INSERT
		_, err = pool.Exec(ctx, UpdateUserLogin, authToken, userId, gsbrcd)
		if err != nil {
			panic(err)
		}
	} else {
		_, err = pool.Exec(ctx, InsertUserLogin, authToken, userId, gsbrcd)
		if err != nil {
			panic(err)
		}
	}

	return authToken
}

func GetNASLogin(pool *pgxpool.Pool, ctx context.Context, authToken string) (int, string) {
	var userId int
	var gsbrcd string
	err := pool.QueryRow(ctx, GetNASUserLogin, authToken).Scan(&userId, &gsbrcd)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, ""
		} else {
			panic(err)
		}
	}

	return userId, gsbrcd
}

func LoginUserToGPCM(pool *pgxpool.Pool, ctx context.Context, authToken string) (User, bool) {
	// Query login table with the auth token.
	userId, gsbrcd := GetNASLogin(pool, ctx, authToken)
	if userId == 0 {
		logging.Notice("DATABASE", "Invalid auth token:", aurora.Cyan(authToken).String())
		return User{}, false
	}

	var exists bool
	err := pool.QueryRow(ctx, DoesUserExist, userId, gsbrcd).Scan(&exists)
	if err != nil {
		panic(err)
	}

	uniqueNickname := common.Base32Encode(userId) + gsbrcd

	user := User{
		UserId:     userId,
		GsbrCode:   gsbrcd,
		Password:   "troll",
		Email:      uniqueNickname + "@nds",
		UniqueNick: uniqueNickname,
	}

	if !exists {
		// Create the GPCM account
		user.CreateUser(pool, ctx)
	} else {
		err := pool.QueryRow(ctx, GetUserProfileID, userId, gsbrcd).Scan(&user.ProfileId)
		if err != nil {
			panic(err)
		}
	}

	return user, true
}
