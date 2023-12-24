package database

import (
	"context"
	"fmt"
	"wwfc/common"
	"wwfc/logging"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/logrusorgru/aurora/v3"
)

func LoginUserToGPCM(pool *pgxpool.Pool, ctx context.Context, userId uint64, gsbrcd string, profileId uint32, ngDeviceId uint32) (User, bool) {
	var exists bool
	err := pool.QueryRow(ctx, DoesUserExist, userId, gsbrcd).Scan(&exists)
	if err != nil {
		return User{}, false
	}

	user := User{
		UserId:   userId,
		GsbrCode: gsbrcd,
	}

	if !exists {
		user.ProfileId = profileId
		user.NgDeviceId = ngDeviceId
		user.UniqueNick = common.Base32Encode(userId) + gsbrcd
		user.Email = user.UniqueNick + "@nds"

		// Create the GPCM account
		err := user.CreateUser(pool, ctx)
		if err != nil {
			logging.Error("DATABASE", "Error creating user:", aurora.Cyan(userId), aurora.Cyan(gsbrcd), aurora.Cyan(user.ProfileId), "\nerror:", err.Error())
			return User{}, false
		}

		logging.Notice("DATABASE", "Created new GPCM user:", aurora.Cyan(userId), aurora.Cyan(gsbrcd), aurora.Cyan(user.ProfileId))
	} else {
		var expectedNgId *uint32
		var firstName *string
		var lastName *string
		err := pool.QueryRow(ctx, GetUserProfileID, userId, gsbrcd).Scan(&user.ProfileId, &expectedNgId, &user.Email, &user.UniqueNick, &firstName, &lastName)
		if err != nil {
			panic(err)
		}

		if firstName != nil {
			user.FirstName = *firstName
		}

		if lastName != nil {
			user.LastName = *lastName
		}

		if expectedNgId != nil && user.NgDeviceId != 0 {
			user.NgDeviceId = *expectedNgId
			if ngDeviceId != 0 && user.NgDeviceId != ngDeviceId {
				logging.Error("DATABASE", "NG device ID mismatch for profile", aurora.Cyan(user.ProfileId), "- expected", aurora.Cyan(fmt.Sprintf("%08x", user.NgDeviceId)), "but got", aurora.Cyan(fmt.Sprintf("%08x", ngDeviceId)))
				return User{}, false
			}
		} else if ngDeviceId != 0 {
			user.NgDeviceId = ngDeviceId
			_, err := pool.Exec(ctx, UpdateUserNGDeviceID, user.ProfileId, ngDeviceId)
			if err != nil {
				panic(err)
			}
		}

		if profileId != 0 && user.ProfileId != profileId {
			err := user.UpdateProfileID(pool, ctx, profileId)
			if err != nil {
				logging.Warn("DATABASE", "Could not update", aurora.Cyan(userId), aurora.Cyan(gsbrcd), "profile ID from", aurora.Cyan(user.ProfileId), "to", aurora.Cyan(profileId))
			} else {
				logging.Notice("DATABASE", "Updated GPCM user profile ID:", aurora.Cyan(userId), aurora.Cyan(gsbrcd), aurora.Cyan(user.ProfileId))
			}
		}

		logging.Notice("DATABASE", "Log in GPCM user:", aurora.Cyan(userId), aurora.Cyan(user.GsbrCode), "-", aurora.Cyan(user.ProfileId))
	}

	// This should be set if the user already knows its own profile ID
	if profileId != 0 && user.LastName == "" {
		user.UpdateProfile(pool, ctx, map[string]string{
			"lastname": "000000000" + gsbrcd,
		})
	}

	return user, true
}
