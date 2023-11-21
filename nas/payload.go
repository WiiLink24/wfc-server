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

	r.payload = append([]byte{0x01, 0x2C}, dat...)
	return map[string]string{}
}

func handlePayloadRequest(w http.ResponseWriter, r *http.Request) {
	// Example request:
	// GET /payload?g=RMCPD00&s=4e44b095817f8cfb62e6cffd57e9cfd411004a492784039ea4b2b7ca64717c91&h=9fdb6f60

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

		_, err := hex.DecodeString(salt[0])
		if err != nil {
			log.Printf(aurora.Red("invalid salt hex string").String())
			return
		}

		saltHashTest, ok := query["h"]
		if !ok {
			log.Printf(aurora.Red("Request is missing salt hash").String())
			return
		}

		if len(saltHashTest[0]) != 8 {
			log.Printf(aurora.Red("Invalid salt hash length").String())
			return
		}

		saltHashTestData, err := hex.DecodeString(saltHashTest[0])
		if err != nil {
			log.Printf(aurora.Red("Invalid salt hash hex string").String())
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
			log.Printf(aurora.Red("Invalid salt hash").String())
			return
		}

		dat = append(append(dat[:0x110], saltHash...), dat[0x130:]...)
	}

	// TODO: Cache the request for a zero salt

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
