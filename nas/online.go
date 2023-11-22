package nas

import (
  "strconv"
  "wwfc/common"
  "net/http"
)

func returnOnlineStats(w http.ResponseWriter) {
  w.Header().Set("Content-Type", "text/plain")
  w.Header().Set("Content-Length", strconv.Itoa(len(strconv.Itoa(common.OnlineUsers))))
  w.Write([]byte(strconv.Itoa(common.OnlineUsers)))
} 
