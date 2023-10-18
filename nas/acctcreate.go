package nas

import (
	"encoding/base64"
	"net/url"
	"strconv"
	"strings"
	"wwfc/database"
)

func acctcreate(r *Response) {
	param := url.Values{}
	param.Set("retry", strings.Replace(base64.StdEncoding.EncodeToString([]byte("0")), "=", "*", -1))
	param.Set("returncd", strings.Replace(base64.StdEncoding.EncodeToString([]byte("002")), "=", "*", -1))
    param.Set("userid", strings.Replace(base64.StdEncoding.EncodeToString([]byte(strconv.FormatInt(database.GetUniqueUserID(), 10))), "=", "*", -1))

	// Encode and send off to be written!
	r.payload = []byte(param.Encode())
	r.payload = []byte(strings.Replace(string(r.payload), "%2A", "*", -1))
}
