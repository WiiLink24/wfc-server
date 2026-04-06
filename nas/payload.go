package nas

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

func downloadStage1(w http.ResponseWriter, r *http.Request) {
	stage1Ver := r.URL.Path[len(r.URL.Path)-1] - '0'

	path := "payload/stage1.bin"
	if stage1Ver != 0 {
		path = "payload/stage1v" + strconv.Itoa(int(stage1Ver)) + ".bin"
	}
	dat, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	payload := append([]byte{0x01, 0x2C}, dat...)

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
	_, err = w.Write(payload)
	if err != nil {
		logging.Error("NAS", "Error writing stage 1 payload:", err)
	}
}

func forwardPayloadRequest(w http.ResponseWriter, r *http.Request) {
	moduleName := getModuleName(r)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	r.URL.Scheme = "http"
	r.URL.Host = payloadServerAddress
	r.RequestURI = ""
	r.Host = payloadServerAddress

	resp, err := client.Do(r)
	if err != nil {
		logging.Error(moduleName, "Error forwarding payload request:", err)
		replyHTTPError(w, http.StatusBadGateway, "502 Bad Gateway")
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Copy the response headers and status code
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// Copy the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logging.Error(moduleName, "Error reading payload response body:", err)
		replyHTTPError(w, http.StatusInternalServerError, "500 Internal Server Error")
		return
	}
	_, err = w.Write(body)
	if err != nil {
		logging.Error(moduleName, "Error writing payload response body:", err)
	}
}

func handlePayloadRequest(w http.ResponseWriter, r *http.Request) {
	// Example request:
	// GET /payload?g=RMCPD00&s=4e44b095817f8cfb62e6cffd57e9cfd411004a492784039ea4b2b7ca64717c91&h=9fdb6f60

	moduleName := getModuleName(r)

	u, err := url.Parse(r.URL.String())
	if err != nil {
		logging.Error(moduleName, "Failed to parse URL")
		return
	}

	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		logging.Error(moduleName, "Failed to parse URL query")
		return
	}

	// Read payload ID (g) from URL
	game := query.Get("g")
	if len(game) != 7 && len(game) != 9 {
		logging.Error(moduleName, "Invalid or missing game ID:", aurora.Cyan(game))
		return
	}

	if (len(game) == 7 && game[4] != 'D') || (len(game) == 9 && game[4] != 'N') {
		logging.Error(moduleName, "Invalid game ID:", aurora.Cyan(game))
		return
	}

	for i := 0; i < 4; i++ {
		if (game[i] >= 'A' && game[i] <= 'Z') || (game[i] >= '0' && game[i] <= '9') {
			continue
		}

		logging.Error(moduleName, "Invalid game ID char:", aurora.Cyan(game))
		return
	}

	for i := 5; i < len(game); i++ {
		if (game[i] >= '0' && game[i] <= '9') || (game[i] >= 'a' && game[i] <= 'f') {
			continue
		}

		logging.Error(moduleName, "Invalid game ID version:", aurora.Cyan(game))
		return
	}

	dat, err := os.ReadFile("payload/binary/payload." + game + ".bin")
	if err != nil {
		logging.Error(moduleName, "Failed to read payload file")
		return
	}

	salt, ok := query["s"]
	if ok {
		if len(salt[0]) != 64 {
			logging.Error(moduleName, "Invalid salt length:", aurora.BrightCyan(len(salt[0])))
			return
		}

		_, err := hex.DecodeString(salt[0])
		if err != nil {
			logging.Error(moduleName, "Invalid salt hex string")
			return
		}

		saltHashTest, ok := query["h"]
		if !ok {
			logging.Error(moduleName, "Request is missing salt hash")
			return
		}

		if len(saltHashTest[0]) != 8 {
			logging.Error(moduleName, "Invalid salt hash length:", aurora.BrightCyan(len(saltHashTest[0])))
			return
		}

		saltHashTestData, err := hex.DecodeString(saltHashTest[0])
		if err != nil {
			logging.Error(moduleName, "Invalid salt hash hex string")
			return
		}

		// Generate the salt hash
		saltHashData := "payload?g=" + query["g"][0] + "&s=" + query["s"][0]

		hashCtx := sha256.New()
		_, err = hashCtx.Write([]byte(saltHashData))
		if err != nil {
			panic(err)
		}

		saltHash := hashCtx.Sum(nil)
		if !bytes.Equal(saltHashTestData, saltHash[:4]) {
			logging.Error(moduleName, "Salt hash mismatch")
			return
		}

		dat = append(append(dat[:0x110], saltHash...), dat[0x130:]...)
	}

	// TODO: Cache the request for a zero salt

	rsaData, err := os.ReadFile("payload/private-key.pem")
	if err != nil {
		panic(err)
	}

	rsaBlock, _ := pem.Decode(rsaData)
	parsedKey, err := x509.ParsePKCS8PrivateKey(rsaBlock.Bytes)
	if err != nil {
		panic(err)
	}

	rsaKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		panic("Unexpected key type")
	}

	// Hash our data then sign
	hash := sha256.New()
	_, err = hash.Write(dat[0x110:])
	if err != nil {
		panic(err)
	}

	contentsHashSum := hash.Sum(nil)

	reader := rand.Reader
	signature, err := rsa.SignPKCS1v15(reader, rsaKey, crypto.SHA256, contentsHashSum)
	if err != nil {
		panic(err)
	}

	dat = append(append(dat[:0x10], signature...), dat[0x110:]...)

	w.Header().Set("Content-Length", strconv.Itoa(len(dat)))
	_, err = w.Write(dat)
	if err != nil {
		logging.Error("NAS", "Error writing payload response body:", err)
	}
}
