package nas

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"github.com/logrusorgru/aurora/v3"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

func getStage1(r *Response, fields map[string]string) map[string]string {
	dat, err := os.ReadFile("payload/stage1.bin")
	if err != nil {
		panic(err)
	}

	r.payload = dat
	return map[string]string{}
}

func handlePayloadRequest(w http.ResponseWriter, r *http.Request) {
	// Example request:
	// GET /payload?g=RMCPD00&s=30f75b5ec08c159cb66493d5ae889090012a87d8a113696704ade9b610f6f33c

	u, err := url.Parse(r.URL.String())
	if err != nil {
		log.Printf(aurora.Red("failed to parse URL").String())
		return
	}

	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		log.Printf(aurora.Red("failed to parse URL query").String())
		return
	}

	// TODO: Read payload ID (g) from URL
	dat, err := os.ReadFile("payload/binary/RMCPD00.bin")
	if err != nil {
		log.Printf(aurora.Red("failed to read payload file").String())
		return
	}

	salt, ok := query["s"]
	if ok {
		if len(salt[0]) != 64 {
			log.Printf(aurora.Red("invalid salt length").String())
			return
		}

		salt_data, err := hex.DecodeString(salt[0])
		if err != nil {
			log.Printf(aurora.Red("invalid salt hex string").String())
			return
		}

		dat = append(append(dat[:0x110], salt_data...), dat[0x130:]...)
	}

	rsaData, err := os.ReadFile("payload/stage1-private-temp.pem")
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
		log.Fatalf("got unexpected key type: %T", parsedKey)
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

	log.Printf(aurora.White("write data").String())
	w.Header().Set("Content-Length", strconv.Itoa(len(dat)))
	w.Write(dat)
}
