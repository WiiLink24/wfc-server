package database

import (
	"context"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/logrusorgru/aurora/v3"
	"wwfc/common"
	"wwfc/logging"
)

func LoginUserToGPCM(pool *pgxpool.Pool, ctx context.Context, userId uint64, gsbrcd string) (User, bool) {
	var exists bool
	err := pool.QueryRow(ctx, DoesUserExist, userId, gsbrcd).Scan(&exists)
	if err != nil {
		panic(err)
	}

	uniqueNickname := common.Base32Encode(userId) + gsbrcd

	user := User{
		UserId:     userId,
		GsbrCode:   gsbrcd,
		Email:      uniqueNickname + "@nds",
		UniqueNick: uniqueNickname,
	}

	if !exists {
		// Create the GPCM account
		user.CreateUser(pool, ctx)

		logging.Notice("DATABASE", "Created new GPCM user:", aurora.Cyan(userId), aurora.Cyan(gsbrcd), "-", aurora.Cyan(user.ProfileId))
	} else {
		err := pool.QueryRow(ctx, GetUserProfileID, userId, gsbrcd).Scan(&user.ProfileId)
		if err != nil {
			panic(err)
		}

		logging.Notice("DATABASE", "Log in GPCM user:", aurora.Cyan(userId), aurora.Cyan(user.GsbrCode), "-", aurora.Cyan(user.ProfileId))
	}

	return user, true
}
