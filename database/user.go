package database

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"wwfc/common"
)

const (
	InsertUser        = `INSERT INTO users (user_id, gsbrcd, password, email, unique_nick) VALUES ($1, $2, $3, $4, $5) RETURNING profile_id`
	CreateUserSession = `INSERT INTO sessions (session_key, profile_id, login_ticket) VALUES ($1, $2, $3)`
	DoesUserExist     = `SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1 AND gsbrcd = $2)`
	DeleteUserSession = `DELETE FROM sessions WHERE profile_id = $1`
)

type User struct {
	ProfileId  int
	UserId     int
	GsbrCode   string
	Password   string
	Email      string
	UniqueNick string
}

func (user *User) CreateUser(pool *pgxpool.Pool, ctx context.Context) {
	err := pool.QueryRow(ctx, InsertUser, user.UserId, user.GsbrCode, user.Password, user.Email, user.UniqueNick).Scan(&user.ProfileId)
	if err != nil {
		panic(err)
	}
}

func CreateSession(pool *pgxpool.Pool, ctx context.Context, profileId int, loginTicket string) string {
	// Delete session first.
	deleteSession(pool, ctx, profileId)

	sessionKey := common.RandomString(8)
	_, err := pool.Exec(ctx, CreateUserSession, sessionKey, profileId, loginTicket)
	if err != nil {
		panic(err)
	}

	return sessionKey
}

func deleteSession(pool *pgxpool.Pool, ctx context.Context, profileId int) {
	_, err := pool.Exec(ctx, DeleteUserSession, profileId)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		panic(err)
	}
}

func GetProfile(pool *pgxpool.Pool, ctx context.Context) {

}
