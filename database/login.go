package database

import (
	"context"
	"github.com/jackc/pgx/v4/pgxpool"
	"wwfc/common"
)

const (
	DoesAuthTokenExist = `SELECT EXISTS(SELECT 1 FROM logins WHERE auth_token = $1)`
	DoesNASUserExist   = `SELECT EXISTS(SELECT 1 FROM logins WHERE user_id = $1)`
	UpdateUserLogin    = `UPDATE logins SET auth_token = $1 WHERE user_id = $2`
	InsertUserLogin    = `INSERT INTO logins (auth_token, user_id, gsbrcd) VALUES ($1, $2, $3)`
	GetNASUserLogin    = `SELECT user_id, gsbrcd FROM logins WHERE auth_token = $1 LIMIT 1`
)

func GenerateAuthToken(pool *pgxpool.Pool, ctx context.Context, userId int, gsbrcd string) string {
	authToken := "NDS" + common.RandomString(80)
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

	var exists bool
	err := pool.QueryRow(ctx, DoesNASUserExist, userId).Scan(&exists)
	if err != nil {
		panic(err)
	}

	if exists {
		// UPDATE rather than INSERT
		_, err = pool.Exec(ctx, UpdateUserLogin, authToken, userId)
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
		panic(err)
	}

	return userId, gsbrcd
}

func LoginUserToGCPM(pool *pgxpool.Pool, ctx context.Context, authToken string) User {
	// Query login table with the auth token.
	userId, gsbrcd := GetNASLogin(pool, ctx, authToken)

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
		// TODO get the profile ID!!!!!
		user.ProfileId = 4
	}

	return user
}
