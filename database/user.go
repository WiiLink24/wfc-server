package database

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v4/pgxpool"
	"math/rand"
)

const (
	InsertUser              = `INSERT INTO users (user_id, gsbrcd, password, ng_device_id, email, unique_nick) VALUES ($1, $2, $3, $4, $5, $6) RETURNING profile_id`
	InsertUserWithProfileID = `INSERT INTO users (profile_id, user_id, gsbrcd, password, ng_device_id, email, unique_nick) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	UpdateUserTable         = `UPDATE users SET firstname = CASE WHEN $3 THEN $2 ELSE firstname END, lastname = CASE WHEN $5 THEN $4 ELSE lastname END WHERE profile_id = $1`
	UpdateUserProfileID     = `UPDATE users SET profile_id = $3 WHERE user_id = $1 AND gsbrcd = $2`
	UpdateUserNGDeviceID    = `UPDATE users SET ng_device_id = $2 WHERE profile_id = $1`
	GetUser                 = `SELECT user_id, gsbrcd, email, unique_nick, firstname, lastname FROM users WHERE profile_id = $1`
	DoesUserExist           = `SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1 AND gsbrcd = $2)`
	IsProfileIDInUse        = `SELECT EXISTS(SELECT 1 FROM users WHERE profile_id = $1)`
	DeleteUserSession       = `DELETE FROM sessions WHERE profile_id = $1`
	GetUserProfileID        = `SELECT profile_id, ng_device_id, email, unique_nick, firstname, lastname FROM users WHERE user_id = $1 AND gsbrcd = $2`

	GetMKWFriendInfoQuery    = `SELECT mariokartwii_friend_info FROM users WHERE profile_id = $1`
	UpdateMKWFriendInfoQuery = `UPDATE users SET mariokartwii_friend_info = $2 WHERE profile_id = $1`
)

type User struct {
	ProfileId  uint32
	UserId     uint64
	GsbrCode   string
	NgDeviceId uint32
	Email      string
	UniqueNick string
	FirstName  string
	LastName   string
}

var (
	ErrProfileIDInUse         = errors.New("profile ID is already in use")
	ErrReservedProfileIDRange = errors.New("profile ID is in reserved range")
)

func (user *User) CreateUser(pool *pgxpool.Pool, ctx context.Context) error {
	if user.ProfileId == 0 {
		return pool.QueryRow(ctx, InsertUser, user.UserId, user.GsbrCode, "", user.NgDeviceId, user.Email, user.UniqueNick).Scan(&user.ProfileId)
	}

	if user.ProfileId >= 1000000000 {
		return ErrReservedProfileIDRange
	}

	var exists bool
	err := pool.QueryRow(ctx, IsProfileIDInUse, user.ProfileId).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		return ErrProfileIDInUse
	}

	_, err = pool.Exec(ctx, InsertUserWithProfileID, user.ProfileId, user.UserId, user.GsbrCode, "", user.NgDeviceId, user.Email, user.UniqueNick)
	return err
}

func (user *User) UpdateProfileID(pool *pgxpool.Pool, ctx context.Context, newProfileId uint32) error {
	if newProfileId >= 1000000000 {
		return ErrReservedProfileIDRange
	}

	var exists bool
	err := pool.QueryRow(ctx, IsProfileIDInUse, newProfileId).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		return ErrProfileIDInUse
	}

	_, err = pool.Exec(ctx, UpdateUserProfileID, user.UserId, user.GsbrCode, newProfileId)
	if err == nil {
		user.ProfileId = newProfileId
	}

	return err
}

func GetUniqueUserID() uint64 {
	// Not guaranteed unique but doesn't matter in practice if multiple people have the same user ID.
	return uint64(rand.Int63n(0x80000000000))
}

func (user *User) UpdateProfile(pool *pgxpool.Pool, ctx context.Context, data map[string]string) {
	firstName, firstNameExists := data["firstname"]
	lastName, lastNameExists := data["lastname"]

	_, err := pool.Exec(ctx, UpdateUserTable, user.ProfileId, firstName, firstNameExists, lastName, lastNameExists)
	if err != nil {
		panic(err)
	}

	if firstNameExists {
		user.FirstName = firstName
	}

	if lastNameExists {
		user.LastName = lastName
	}
}

func GetProfile(pool *pgxpool.Pool, ctx context.Context, profileId uint32) (User, bool) {
	user := User{}
	row := pool.QueryRow(ctx, GetUser, profileId)
	err := row.Scan(&user.UserId, &user.GsbrCode, &user.Email, &user.UniqueNick, &user.FirstName, &user.LastName)
	if err != nil {
		return User{}, false
	}

	user.ProfileId = profileId
	return user, true
}

func GetMKWFriendInfo(pool *pgxpool.Pool, ctx context.Context, profileId uint32) string {
	var info string
	err := pool.QueryRow(ctx, GetMKWFriendInfoQuery, profileId).Scan(&info)
	if err != nil {
		return ""
	}

	return info
}

func UpdateMKWFriendInfo(pool *pgxpool.Pool, ctx context.Context, profileId uint32, info string) {
	_, err := pool.Exec(ctx, UpdateMKWFriendInfoQuery, profileId, info)
	if err != nil {
		panic(err)
	}
}
