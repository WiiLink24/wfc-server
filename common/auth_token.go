package common

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"strings"
	"time"
)

func generateRandom(n int) []byte {
	key := make([]byte, n)

	read, err := rand.Read(key)
	if err != nil {
		panic(err)
	}

	if read != n {
		panic("short rand.Read()")
	}

	return key
}

var (
	authTokenKey   = generateRandom(16)
	authTokenIV    = generateRandom(16)
	authTokenMagic = generateRandom(15)

	loginTicketKey   = generateRandom(16)
	loginTicketIV    = generateRandom(16)
	loginTicketMagic = generateRandom(4)
)

func appendString(blob []byte, value string, maxlen int) []byte {
	if len([]byte(value)) < maxlen {
		blob = append(blob, append([]byte(value), make([]byte, maxlen-len(value))...)...)
	} else {
		blob = append(blob, []byte(value)[:maxlen]...)
	}

	return blob
}

func MarshalNASAuthToken(gamecd string, userid uint64, gsbrcd string, cfc uint64, region byte, lang byte, ingamesn string, isLocalhost bool) (string, string) {
	blob := binary.LittleEndian.AppendUint64([]byte{}, uint64(time.Now().Unix()))

	blob = appendString(blob, gamecd, 4)

	blob = append(blob, binary.LittleEndian.AppendUint64([]byte{}, userid)[:6]...)

	blob = append(blob, byte(min(len([]byte(gsbrcd)), 16)))
	blob = appendString(blob, gsbrcd, 16)

	blob = append(blob, binary.LittleEndian.AppendUint64([]byte{}, cfc)[:7]...)
	blob = append(blob, region, lang)

	blob = append(blob, byte(min(len([]byte(ingamesn)), 75)))
	blob = appendString(blob, ingamesn, 75)

	challenge := RandomString(8)
	blob = append(blob, []byte(challenge)...)

	if isLocalhost {
		blob = append(blob, 0x01)
	} else {
		blob = append(blob, 0x00)
	}

	blob = append(blob, authTokenMagic...)

	block, err := aes.NewCipher(authTokenKey)
	if err != nil {
		panic(err)
	}

	cipher.NewCBCEncrypter(block, authTokenIV).CryptBlocks(blob, blob)
	return "NDS" + Base64DwcEncoding.EncodeToString(blob), challenge
}

func UnmarshalNASAuthToken(token string) (err error, gamecd string, issuetime time.Time, userid uint64, gsbrcd string, cfc uint64, region byte, lang byte, ingamesn string, challenge string, isLocalhost bool) {
	if !strings.HasPrefix(token, "NDS") {
		err = errors.New("invalid auth token prefix")
		return
	}

	blob, err := Base64DwcEncoding.DecodeString(token[3:])
	if err != nil {
		return
	}

	if len(blob) != 0x90 {
		err = errors.New("invalid auth token length")
		return
	}

	block, err := aes.NewCipher(authTokenKey)
	if err != nil {
		panic(err)
	}

	cipher.NewCBCDecrypter(block, authTokenIV).CryptBlocks(blob, blob)

	if !bytes.Equal(blob[0x90-len(authTokenMagic):0x90], authTokenMagic) {
		err = errors.New("invalid auth token magic")
		return
	}

	issuetime = time.Unix(int64(binary.LittleEndian.Uint64(blob[0x0:0x8])), 0)
	gamecd = string(blob[0x8:0xC])
	userid = binary.LittleEndian.Uint64(append(bytes.Clone(blob[0xC:0x12]), 0, 0))
	gsbrcd = string(blob[0x13 : 0x13+min(blob[0x12], 16)])
	cfc = binary.LittleEndian.Uint64(append(bytes.Clone(blob[0x23:0x2A]), 0))
	region = blob[0x2A]
	lang = blob[0x2B]
	ingamesn = string(blob[0x2D : 0x2D+min(blob[0x2C], 75)])
	challenge = string(blob[0x78:0x80])
	isLocalhost = blob[0x80] == 0x01
	return
}

func MarshalGPCMLoginTicket(profileId uint32) string {
	blob := binary.LittleEndian.AppendUint64([]byte{}, uint64(time.Now().Unix()))
	blob = binary.LittleEndian.AppendUint32(blob, profileId)
	blob = append(blob, loginTicketMagic...)

	block, err := aes.NewCipher(loginTicketKey)
	if err != nil {
		panic(err)
	}

	cipher.NewCBCEncrypter(block, loginTicketIV).CryptBlocks(blob, blob)
	return Base64DwcEncoding.EncodeToString(blob)
}

func UnmarshalGPCMLoginTicket(ticket string) (err error, profileId uint32, issuetime time.Time) {
	blob, err := Base64DwcEncoding.DecodeString(ticket)
	if err != nil {
		return
	}

	if len(blob) != 0x10 {
		err = errors.New("invalid login ticket length")
		return
	}

	block, err := aes.NewCipher(loginTicketKey)
	if err != nil {
		panic(err)
	}

	cipher.NewCBCDecrypter(block, loginTicketIV).CryptBlocks(blob, blob)

	if !bytes.Equal(blob[0xC:0x10], loginTicketMagic) {
		err = errors.New("invalid login ticket magic")
		return
	}

	issuetime = time.Unix(int64(binary.LittleEndian.Uint64(blob[0x0:0x8])), 0)
	profileId = binary.LittleEndian.Uint32(blob[0x8:0xC])
	return
}
