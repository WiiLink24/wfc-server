package database

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"math/rand"
	"wwfc/common"
)

const (
	InsertUser        = `INSERT INTO users (user_id, gsbrcd, password, email, unique_nick) VALUES ($1, $2, $3, $4, $5) RETURNING profile_id`
	UpdateUserTable   = `UPDATE users SET firstname = CASE WHEN $3 THEN $2 ELSE firstname END, lastname = CASE WHEN $5 THEN $4 ELSE lastname END WHERE profile_id = $1 RETURNING user_id, gsbrcd, password, email, unique_nick, firstname, lastname`
	GetUser           = `SELECT user_id, gsbrcd, password, email, unique_nick, firstname, lastname FROM users WHERE profile_id = $1`
	CreateUserSession = `INSERT INTO sessions (session_key, profile_id, login_ticket) VALUES ($1, $2, $3)`
	DoesUserExist     = `SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1 AND gsbrcd = $2)`
	DeleteUserSession = `DELETE FROM sessions WHERE profile_id = $1`
	GetUserProfileID  = `SELECT profile_id FROM users WHERE user_id = $1 AND gsbrcd = $2`
)

type User struct {
	ProfileId  uint32
	UserId     int64
	GsbrCode   string
	Password   string
	Email      string
	UniqueNick string
	FirstName  string
	LastName   string
}

func (user *User) CreateUser(pool *pgxpool.Pool, ctx context.Context) {
	err := pool.QueryRow(ctx, InsertUser, user.UserId, user.GsbrCode, user.Password, user.Email, user.UniqueNick).Scan(&user.ProfileId)
	if err != nil {
		panic(err)
	}
}

func GetUniqueUserID() int64 {
	// Not guaranteed unique but doesn't matter in practice if multiple people have the same user ID.
	return rand.Int63n(0x80000000000)
}

func UpdateProfile(pool *pgxpool.Pool, ctx context.Context, profileId uint32, data map[string]string) User {
	firstName, firstNameExists := data["firstname"]
	lastName, lastNameExists := data["lastname"]

	user := User{}
	row := pool.QueryRow(ctx, UpdateUserTable, profileId, firstName, firstNameExists, lastName, lastNameExists)
	err := row.Scan(&user.UserId, &user.GsbrCode, &user.Password, &user.Email, &user.UniqueNick, &user.FirstName, &user.LastName)
	if err != nil {
		panic(err)
	}

	user.ProfileId = profileId
	return user
}

func CreateSession(pool *pgxpool.Pool, ctx context.Context, profileId uint32, loginTicket string) string {
	sessionKey := common.RandomString(8)
	_, err := pool.Exec(ctx, CreateUserSession, sessionKey, profileId, loginTicket)
	if err != nil {
		panic(err)
	}

	return sessionKey
}

func deleteSession(pool *pgxpool.Pool, ctx context.Context, profileId uint32) {
	_, err := pool.Exec(ctx, DeleteUserSession, profileId)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		panic(err)
	}
}

func GetProfile(pool *pgxpool.Pool, ctx context.Context, profileId uint32) (User, bool) {
	user := User{}
	row := pool.QueryRow(ctx, GetUser, profileId)
	err := row.Scan(&user.UserId, &user.GsbrCode, &user.Password, &user.Email, &user.UniqueNick, &user.FirstName, &user.LastName)
	if err != nil {
		return User{}, false
	}

	user.ProfileId = profileId
	return user, true
}
