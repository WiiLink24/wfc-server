package api

import (
	"wwfc/database"
)

type UserActionResponse struct {
	User    database.User
	Success bool
	Error   string
}
