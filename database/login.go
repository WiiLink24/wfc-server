package database

import (
	"context"
	"errors"
	"fmt"
	"time"
	"wwfc/common"
	"wwfc/logging"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/logrusorgru/aurora/v3"
)

var (
	ErrDeviceIDMismatch = errors.New("NG device ID mismatch")
	ErrProfileBannedTOS = errors.New("profile is banned for violating the Terms of Service")
)

func LoginUserToGPCM(pool *pgxpool.Pool, ctx context.Context, userId uint64, gsbrcd string, profileId uint32, ngDeviceId uint32, ipAddress string, ingamesn string) (User, error) {
	var exists bool
	err := pool.QueryRow(ctx, DoesUserExist, userId, gsbrcd).Scan(&exists)
	if err != nil {
		return User{}, err
	}

	user := User{
		UserId:   userId,
		GsbrCode: gsbrcd,
	}

	if !exists {
		user.ProfileId = profileId
		user.NgDeviceId = []uint32{ngDeviceId}
		if ngDeviceId == 0 {
			user.NgDeviceId = []uint32{}
		}
		user.UniqueNick = common.Base32Encode(userId) + gsbrcd
		user.Email = user.UniqueNick + "@nds"

		// Create the GPCM account
		err := user.CreateUser(pool, ctx)
		if err != nil {
			logging.Error("DATABASE", "Error creating user:", aurora.Cyan(userId), aurora.Cyan(gsbrcd), aurora.Cyan(user.ProfileId), "\nerror:", err.Error())
			return User{}, err
		}

		logging.Notice("DATABASE", "Created new GPCM user:", aurora.Cyan(userId), aurora.Cyan(gsbrcd), aurora.Cyan(user.ProfileId))
	} else {
		var firstName *string
		var lastName *string
		err := pool.QueryRow(ctx, GetUserProfileID, userId, gsbrcd).Scan(&user.ProfileId, &user.NgDeviceId, &user.Email, &user.UniqueNick, &firstName, &lastName, &user.OpenHost)
		if err != nil {
			return User{}, err
		}

		if firstName != nil {
			user.FirstName = *firstName
		}

		if lastName != nil {
			user.LastName = *lastName
		}

		validDeviceId := false
		deviceIdList := ""
		for index, id := range user.NgDeviceId {
			if id == ngDeviceId {
				validDeviceId = true
				break
			}

			if id == 0 {
				// Replace the 0 with the actual device ID
				user.NgDeviceId[index] = ngDeviceId
				_, err = pool.Exec(ctx, UpdateUserNGDeviceID, user.ProfileId, user.NgDeviceId)
				validDeviceId = true
				break
			}

			deviceIdList += aurora.Cyan(fmt.Sprintf("%08x", id)).String() + ", "
		}

		if !validDeviceId && ngDeviceId != 0 {
			if len(user.NgDeviceId) > 0 && !common.GetConfig().AllowMultipleDeviceIDs {
				logging.Error("DATABASE", "NG device ID mismatch for profile", aurora.Cyan(user.ProfileId), "- expected one of {", deviceIdList[:len(deviceIdList)-2], "} but got", aurora.Cyan(fmt.Sprintf("%08x", ngDeviceId)))
				return User{}, ErrDeviceIDMismatch
			} else if len(user.NgDeviceId) > 0 {
				logging.Warn("DATABASE", "Adding NG device ID", aurora.Cyan(fmt.Sprintf("%08x", ngDeviceId)), "to profile", aurora.Cyan(user.ProfileId))
			}

			user.NgDeviceId = append(user.NgDeviceId, ngDeviceId)
			_, err = pool.Exec(ctx, UpdateUserNGDeviceID, user.ProfileId, user.NgDeviceId)
		} else if !validDeviceId && ngDeviceId == 0 {
			if len(user.NgDeviceId) > 0 && !common.GetConfig().AllowConnectWithoutDeviceID {
				logging.Error("DATABASE", "NG device ID not provided for profile", aurora.Cyan(user.ProfileId), "- expected one of {", deviceIdList[:len(deviceIdList)-2], "} but got", aurora.Cyan("00000000"))
				return User{}, ErrDeviceIDMismatch
			}
		}

		if err != nil {
			return User{}, err
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

	// Update the user's last IP address and ingamesn
	_, err = pool.Exec(ctx, UpdateUserLastIPAddress, user.ProfileId, ipAddress, ingamesn)
	if err != nil {
		return User{}, err
	}

	// Find ban from device ID or IP address
	var banExists bool
	var banTOS bool
	var bannedDeviceIdList []uint32
	timeNow := time.Now()
	err = pool.QueryRow(ctx, SearchUserBan, user.NgDeviceId, user.ProfileId, ipAddress, timeNow).Scan(&banExists, &banTOS, &bannedDeviceIdList)
	if err != nil {
		if err != pgx.ErrNoRows {
			return User{}, err
		}

		banExists = false
	}

	if banExists {
		// Find first device ID in common
		bannedDeviceId := uint32(0)
		for _, id := range bannedDeviceIdList {
			for _, id2 := range user.NgDeviceId {
				if id == id2 {
					bannedDeviceId = id
					break
				}
			}

			if bannedDeviceId != 0 {
				break
			}
		}

		if bannedDeviceId == 0 && len(bannedDeviceIdList) > 0 {
			bannedDeviceId = bannedDeviceIdList[len(bannedDeviceIdList)-1]
		}

		if banTOS {
			logging.Warn("DATABASE", "Profile", aurora.Cyan(user.ProfileId), "is banned")
			return User{RestrictedDeviceId: bannedDeviceId}, ErrProfileBannedTOS
		}

		logging.Warn("DATABASE", "Profile", aurora.Cyan(user.ProfileId), "is restricted")
		user.Restricted = true
		user.RestrictedDeviceId = bannedDeviceId
	}

	return user, nil
}

func LoginUserToGameStats(pool *pgxpool.Pool, ctx context.Context, userId uint64, gsbrcd string) (User, error) {
	user := User{
		UserId:   userId,
		GsbrCode: gsbrcd,
	}

	var expectedNgId *uint32
	var firstName *string
	var lastName *string
	err := pool.QueryRow(ctx, GetUserProfileID, userId, gsbrcd).Scan(&user.ProfileId, &expectedNgId, &user.Email, &user.UniqueNick, &firstName, &lastName, &user.OpenHost)
	if err != nil {
		return User{}, err
	}

	if firstName != nil {
		user.FirstName = *firstName
	}

	if lastName != nil {
		user.LastName = *lastName
	}

	return user, nil
}
