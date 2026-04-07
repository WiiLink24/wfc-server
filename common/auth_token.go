package common

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
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
	authTokenMagic = generateRandom(8)

	loginTicketKey   = generateRandom(16)
	loginTicketIV    = generateRandom(16)
	loginTicketMagic = generateRandom(4)
)

var (
	ErrTokenMagic   = errors.New("invalid auth token or login ticket magic")
	ErrTokenExpired = errors.New("auth token or login ticket expired")
	ErrTokenLength  = errors.New("invalid auth token or login ticket length")
)

type NASAuthToken struct {
	IssueTime         uint64
	UserID            uint64
	ConsoleFriendCode uint64
	Region            byte
	Lang              byte
	UnitCode          byte
	GameCode          [4]byte
	GsbrCode          [16]byte
	Challenge         [8]byte
	InGameScreenName  [64]byte
	Magic             [8]byte
}

var nasAuthTokenSize = (binary.Size(NASAuthToken{}) + aes.BlockSize - 1) & ^(aes.BlockSize - 1)

func (t NASAuthToken) Marshal() string {
	t.IssueTime = uint64(time.Now().UTC().Unix())
	copy(t.Magic[:], authTokenMagic)

	var buf bytes.Buffer
	ShouldNotError(binary.Write(&buf, binary.LittleEndian, t))

	// Pad to CBC block size
	data := append(buf.Bytes(), make([]byte, nasAuthTokenSize-len(buf.Bytes()))...)

	block, err := aes.NewCipher(authTokenKey)
	ShouldNotError(err)
	cipher.NewCBCEncrypter(block, authTokenIV).CryptBlocks(data, data)

	return "NDS" + Base64DwcEncoding.EncodeToString(data)
}

func (t *NASAuthToken) Unmarshal(data string) error {
	if !strings.HasPrefix(data, "NDS") {
		return ErrTokenLength
	}

	blob, err := Base64DwcEncoding.DecodeString(data[3:])
	if err != nil {
		return err
	}

	if len(blob) != nasAuthTokenSize {
		return ErrTokenLength
	}

	block, err := aes.NewCipher(authTokenKey)
	ShouldNotError(err)
	cipher.NewCBCDecrypter(block, authTokenIV).CryptBlocks(blob, blob)

	reader := bytes.NewReader(blob)
	if err := binary.Read(reader, binary.LittleEndian, t); err != nil {
		return err
	}

	if !bytes.Equal(t.Magic[:], authTokenMagic) {
		return ErrTokenMagic
	}

	currentTime := time.Now().UTC()
	issueTime := time.Unix(int64(t.IssueTime), 0)
	if issueTime.Before(currentTime.Add(-10*time.Minute)) || issueTime.After(currentTime) {
		return ErrTokenExpired
	}

	return nil
}

type GPCMLoginTicket struct {
	IssueTime uint64
	ProfileID uint32
	Magic     [4]byte
}

var gpcmLoginTicketSize = (binary.Size(GPCMLoginTicket{}) + aes.BlockSize - 1) & ^(aes.BlockSize - 1)

func (t GPCMLoginTicket) Marshal() string {
	t.IssueTime = uint64(time.Now().UTC().Unix())
	copy(t.Magic[:], loginTicketMagic)

	var buf bytes.Buffer
	ShouldNotError(binary.Write(&buf, binary.LittleEndian, t))

	// Pad to CBC block size
	data := append(buf.Bytes(), make([]byte, gpcmLoginTicketSize-len(buf.Bytes()))...)

	block, err := aes.NewCipher(loginTicketKey)
	ShouldNotError(err)
	cipher.NewCBCEncrypter(block, loginTicketIV).CryptBlocks(data, data)

	return base64.StdEncoding.EncodeToString(data)
}

func (t *GPCMLoginTicket) Unmarshal(ticket string) error {
	blob, err := base64.StdEncoding.DecodeString(ticket)
	if err != nil {
		return err
	}

	if len(blob) != gpcmLoginTicketSize {
		return ErrTokenLength
	}

	block, err := aes.NewCipher(loginTicketKey)
	ShouldNotError(err)
	cipher.NewCBCDecrypter(block, loginTicketIV).CryptBlocks(blob, blob)

	reader := bytes.NewReader(blob)
	if err := binary.Read(reader, binary.LittleEndian, t); err != nil {
		return err
	}

	if !bytes.Equal(t.Magic[:], loginTicketMagic) {
		return ErrTokenMagic
	}

	currentTime := time.Now().UTC()
	issueTime := time.Unix(int64(t.IssueTime), 0)
	if issueTime.Before(currentTime.Add(-48*time.Hour)) || issueTime.After(currentTime) {
		return ErrTokenExpired
	}

	return nil
}
