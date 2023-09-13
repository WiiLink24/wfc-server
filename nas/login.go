package nas

import (
	"encoding/base64"
	"net/url"
	"strconv"
	"strings"
	"wwfc/database"
)

// TODO: Generate and store in database!!!
const Challenge = "0qUekMb4"

func login(r *Response) {
	// Validate the user id. It must be an integer.
	strUserId, _ := base64.StdEncoding.DecodeString(strings.Replace(r.request.PostForm.Get("userid"), "*", "=", -1))
	userId, err := strconv.Atoi(string(strUserId))
	if err != nil {
		panic(err)
	}

	gsbrcd, _ := base64.StdEncoding.DecodeString(strings.Replace(r.request.PostForm.Get("gsbrcd"), "*", "=", -1))
	authToken := database.GenerateAuthToken(pool, ctx, userId, string(gsbrcd))

	param := url.Values{}
	// param.Set("datetime", strings.Replace(base64.StdEncoding.EncodeToString([]byte("20230911232518")), "=", "*", -1))
	param.Set("retry", strings.Replace(base64.StdEncoding.EncodeToString([]byte("0")), "=", "*", -1))
	param.Set("returncd", strings.Replace(base64.StdEncoding.EncodeToString([]byte("001")), "=", "*", -1))
	param.Set("locator", strings.Replace(base64.StdEncoding.EncodeToString([]byte("gamespy.com")), "=", "*", -1))
	param.Set("challenge", strings.Replace(base64.StdEncoding.EncodeToString([]byte(Challenge)), "=", "*", -1))
	param.Set("token", strings.Replace(base64.StdEncoding.EncodeToString([]byte(authToken)), "=", "*", -1))

	// Encode and send off to be written!
	r.payload = []byte(param.Encode())
	r.payload = []byte(param.Encode())
	r.payload = []byte(strings.Replace(string(r.payload), "%2A", "*", -1))
}
