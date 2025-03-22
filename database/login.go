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

const (
	SearchUserBan = `SELECT has_ban, ban_tos, ng_device_id, ban_reason
		FROM users
		WHERE has_ban = true
			AND (profile_id = $2
				OR (ng_device_id && (SELECT array_agg(the_id) FROM UNNEST($1::bigint[]) AS x(the_id) WHERE x.the_id != 67349608))
				OR last_ip_address = $3
				OR ($3 != '' AND last_ip_address = $4)
				OR $6 && csnum)
			AND (ban_expires IS NULL OR ban_expires > $5)
			ORDER BY ban_tos DESC LIMIT 1`
)

var (
	ErrDeviceIDMismatch = errors.New("NG device ID mismatch")
	ErrProfileBannedTOS = errors.New("profile is banned for violating the Terms of Service")
	ErrCsnumMismatch    = errors.New("csnum mismatch")
)

func handleCsnum(pool *pgxpool.Pool, ctx context.Context, user *User, csnum string, lastIPAddress *string, ipAddress string) (bool, error) {
	success := false
	csnumList := ""
	var err error

	for i, validCsnum := range user.Csnum {
		if validCsnum == csnum {
			success = true
			break
		}

		if validCsnum == "" {
			user.Csnum[i] = csnum
			_, err = pool.Exec(ctx, UpdateUserCsnum, user.ProfileId, csnum)
			success = true
			break
		}

		csnumList += aurora.Cyan(validCsnum).String() + ", "
	}

	if !success && csnum != "" {
		if len(user.Csnum) > 0 && common.GetConfig().AllowMultipleCsnums != "always" {
			if common.GetConfig().AllowMultipleCsnums == "SameIPAddress" && (lastIPAddress == nil || ipAddress != *lastIPAddress) {
				logging.Error("DATABASE", "Csnum mismatch for profile", aurora.Cyan(user.ProfileId), "- expected one of {", csnumList[:len(csnumList)-2], "} but got", aurora.Cyan(csnum))
				return success, ErrCsnumMismatch
			}
		}

		if len(user.Csnum) > 0 {
			logging.Warn("DATABASE", "Adding csnum", aurora.Cyan(csnum), "to profile", aurora.Cyan(user.ProfileId))
		}

		user.Csnum = append(user.Csnum, csnum)
		_, err = pool.Exec(ctx, UpdateUserCsnum, user.ProfileId, user.Csnum)

		success = err == nil
	}

	return success, err
}

func LoginUserToGPCM(pool *pgxpool.Pool, ctx context.Context, userId uint64, gsbrcd string, profileId uint32, ngDeviceId uint32, ipAddress string, ingamesn string, deviceAuth bool, csnum string) (User, error) {
	var exists bool
	err := pool.QueryRow(ctx, DoesUserExist, userId, gsbrcd).Scan(&exists)
	if err != nil {
		return User{}, err
	}

	user := User{
		UserId:   userId,
		GsbrCode: gsbrcd,
	}

	var lastIPAddress *string

	if !exists {
		user.ProfileId = profileId
		user.NgDeviceId = []uint32{ngDeviceId}
		if ngDeviceId == 0 {
			user.NgDeviceId = []uint32{}
		}
		user.UniqueNick = common.Base32Encode(userId) + gsbrcd
		user.Email = user.UniqueNick + "@nds"
		user.Csnum = []string{csnum}

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

		err := pool.QueryRow(ctx, GetUserProfileID, userId, gsbrcd).Scan(&user.ProfileId, &user.NgDeviceId, &user.Email, &user.UniqueNick, &firstName, &lastName, &user.OpenHost, &lastIPAddress, &user.Csnum)
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
			if len(user.NgDeviceId) > 0 && common.GetConfig().AllowMultipleDeviceIDs != "always" {
				if common.GetConfig().AllowMultipleDeviceIDs == "SameIPAddress" && (lastIPAddress == nil || ipAddress != *lastIPAddress) {
					logging.Error("DATABASE", "NG device ID mismatch for profile", aurora.Cyan(user.ProfileId), "- expected one of {", deviceIdList[:len(deviceIdList)-2], "} but got", aurora.Cyan(fmt.Sprintf("%08x", ngDeviceId)))
					return User{}, ErrDeviceIDMismatch
				}
			}

			if len(user.NgDeviceId) > 0 {
				logging.Warn("DATABASE", "Adding NG device ID", aurora.Cyan(fmt.Sprintf("%08x", ngDeviceId)), "to profile", aurora.Cyan(user.ProfileId))
			}

			user.NgDeviceId = append(user.NgDeviceId, ngDeviceId)
			_, err = pool.Exec(ctx, UpdateUserNGDeviceID, user.ProfileId, user.NgDeviceId)
		} else if deviceAuth && !validDeviceId && ngDeviceId == 0 {
			if len(user.NgDeviceId) > 0 && !common.GetConfig().AllowConnectWithoutDeviceID {
				logging.Error("DATABASE", "NG device ID not provided for profile", aurora.Cyan(user.ProfileId), "- expected one of {", deviceIdList[:len(deviceIdList)-2], "} but got", aurora.Cyan("00000000"))
				return User{}, ErrDeviceIDMismatch
			}
		}

		if err != nil {
			return User{}, err
		}

		success, err := handleCsnum(pool, ctx, &user, csnum, lastIPAddress, ipAddress)

		if !success {
			if err != nil {
				return User{}, err
			}

			return User{}, ErrCsnumMismatch
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
	if deviceAuth {
		_, err = pool.Exec(ctx, UpdateUserLastIPAddress, user.ProfileId, ipAddress, ingamesn)
		if err != nil {
			return User{}, err
		}
	}

	emptyString := ""
	if lastIPAddress == nil {
		lastIPAddress = &emptyString
	}

	// Find ban from device ID or IP address
	var banExists bool
	var banTOS bool
	var bannedDeviceIdList []uint32
	var banReason string

	timeNow := time.Now()
	err = pool.QueryRow(ctx, SearchUserBan, user.NgDeviceId, user.ProfileId, ipAddress, *lastIPAddress, timeNow, user.Csnum).Scan(&banExists, &banTOS, &bannedDeviceIdList, &banReason)

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
			return User{RestrictedDeviceId: bannedDeviceId, BanReason: banReason}, ErrProfileBannedTOS
		}

		logging.Warn("DATABASE", "Profile", aurora.Cyan(user.ProfileId), "is restricted")
		user.Restricted = true
		user.RestrictedDeviceId = bannedDeviceId
		user.BanReason = banReason
	}

	return user, nil
}

func LoginUserToGameStats(pool *pgxpool.Pool, ctx context.Context, userId uint64, gsbrcd string) (User, error) {
	user := User{
		UserId:   userId,
		GsbrCode: gsbrcd,
	}

	var firstName *string
	var lastName *string
	var lastIPAddress *string
	err := pool.QueryRow(ctx, GetUserProfileID, userId, gsbrcd).Scan(&user.ProfileId, &user.NgDeviceId, &user.Email, &user.UniqueNick, &firstName, &lastName, &user.OpenHost, &lastIPAddress, &user.Csnum)
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
