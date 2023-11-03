package nas

import (
	"strconv"
	"wwfc/database"
)

func acctcreate(r *Response, fields map[string]string) map[string]string {
	return map[string]string{
		"retry":    "0",
		"returncd": "002",
		"userid":   strconv.FormatInt(database.GetUniqueUserID(), 10),
	}
}
