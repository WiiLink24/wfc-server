package nas

import "net/http"

type Response struct {
	request *http.Request
	writer  *http.ResponseWriter
	payload []byte
}
