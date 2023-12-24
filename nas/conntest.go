package nas

import (
	"net/http"
	"strconv"
)

func handleConnectionTest(w http.ResponseWriter) {
	response := "\n"
	response += `            <!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">` + "\n"
	response += `            <html>` + "\n"
	response += `            <head>` + "\n"
	response += `            <title>HTML Page</title>` + "\n"
	response += `            </head>` + "\n"
	response += `            <body bgcolor="#FFFFFF">` + "\n"
	response += `            This is test.html page` + "\n"
	response += `            </body>` + "\n"
	response += `            </html>` + "\n"
	response += `          `

	w.Header().Set("Content-type", "text/html")
	w.Header().Set("Content-Length", strconv.Itoa(len(response)))
	w.Header().Set("X-Organization", "Nintendo")
	w.Header().Set("Connection", "Keep-Alive")
	w.WriteHeader(200)
	w.Write([]byte(response))
}
