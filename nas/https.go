package nas

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/rc4"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"hash"
	"io"
	"net"
	"os"
	"strings"
	"time"
	"wwfc/common"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

// Buffered conn for passing to regular TLS after peeking the client hello
type bufferedConn struct {
	r *bufio.Reader
	net.Conn
}

func newBufferedConn(c net.Conn) bufferedConn {
	return bufferedConn{bufio.NewReader(c), c}
}

func (b bufferedConn) Peek(n int) ([]byte, error) {
	return b.r.Peek(n)
}

func (b bufferedConn) Read(p []byte) (int, error) {
	return b.r.Read(p)
}

// Bare minimum TLS 1.0 server implementation for the Wii's /dev/net/ssl client
// Use this with a certificate that exploits the Wii's SSL certificate bug to impersonate naswii.nintendowifi.net
// See here: https://github.com/shutterbug2000/wii-ssl-bug
// https://github.com/KaeruTeam/nds-constraint

// Don't use this for anything else, it's not secure

func startHTTPSProxy(config common.Config) {
	address := *config.NASAddressHTTPS + ":" + config.NASPortHTTPS
	nasAddr := *config.NASAddress + ":" + config.NASPort
	privKeyPath := config.KeyPath
	certsPath := config.CertPath
	exploitWii := *config.EnableHTTPSExploitWii
	exploitDS := *config.EnableHTTPSExploitDS

	logging.Notice("NAS-TLS", "Starting HTTPS server on", aurora.BrightCyan(address))
	l, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}

	setupRealTLS(privKeyPath, certsPath)
	// Reread the private key and certs on a regular interval
	go func() {
		for {
			time.Sleep(24 * time.Hour)
			setupRealTLS(privKeyPath, certsPath)
		}
	}()

	if !(exploitWii || exploitDS) {
		// Only handle real TLS requests
		for {
			conn, err := l.Accept()
			if err != nil {
				panic(err)
			}

			go func() {
				moduleName := "NAS-TLS:" + conn.RemoteAddr().String()

				conn.SetDeadline(time.Now().Add(25 * time.Second))

				handleRealTLS(moduleName, conn, nasAddr)
			}()
		}
	}

	// Handle requests from Wii, DS and regular TLS
	var rsaKeyWii *rsa.PrivateKey
	var serverCertsRecordWii []byte
	if exploitWii {
		certWii, err := os.ReadFile(config.CertPathWii)
		if err != nil {
			panic(err)
		}

		rsaDataWii, err := os.ReadFile(config.KeyPathWii)
		if err != nil {
			panic(err)
		}

		rsaBlockWii, _ := pem.Decode(rsaDataWii)
		parsedKeyWii, err := x509.ParsePKCS8PrivateKey(rsaBlockWii.Bytes)
		if err != nil {
			panic(err)
		}

		var ok bool
		rsaKeyWii, ok = parsedKeyWii.(*rsa.PrivateKey)
		if !ok {
			panic("unexpected key type")
		}

		serverCertsRecordWii = []byte{0x16, 0x03, 0x01}

		// Length of the record
		certLenWii := uint32(len(certWii))
		serverCertsRecordWii = append(serverCertsRecordWii, []byte{
			byte((certLenWii + 10) >> 8),
			byte(certLenWii + 10),
		}...)

		serverCertsRecordWii = append(serverCertsRecordWii, 0xB)

		serverCertsRecordWii = append(serverCertsRecordWii, []byte{
			byte((certLenWii + 6) >> 16),
			byte((certLenWii + 6) >> 8),
			byte(certLenWii + 6),
		}...)

		serverCertsRecordWii = append(serverCertsRecordWii, []byte{
			byte((certLenWii + 3) >> 16),
			byte((certLenWii + 3) >> 8),
			byte(certLenWii + 3),
		}...)

		serverCertsRecordWii = append(serverCertsRecordWii, []byte{
			byte(certLenWii >> 16),
			byte(certLenWii >> 8),
			byte(certLenWii),
		}...)

		serverCertsRecordWii = append(serverCertsRecordWii, certWii...)

		serverCertsRecordWii = append(serverCertsRecordWii, []byte{
			0x16, 0x03, 0x01, 0x00, 0x04, 0x0E, 0x00, 0x00, 0x00,
		}...)
	}

	var rsaKeyDS *rsa.PrivateKey
	var serverCertsRecordDS []byte
	if exploitDS {
		certDS, err := os.ReadFile(config.CertPathDS)
		if err != nil {
			panic(err)
		}

		rsaDataDS, err := os.ReadFile(config.KeyPathDS)
		if err != nil {
			panic(err)
		}

		rsaBlockDS, _ := pem.Decode(rsaDataDS)
		parsedKeyDS, err := x509.ParsePKCS8PrivateKey(rsaBlockDS.Bytes)
		if err != nil {
			panic(err)
		}

		var ok bool
		rsaKeyDS, ok = parsedKeyDS.(*rsa.PrivateKey)
		if !ok {
			panic("unexpected key type")
		}

		wiiCertDS, err := os.ReadFile(config.WiiCertPathDS)
		if err != nil {
			panic(err)
		}

		serverCertsRecordDS = []byte{0x16, 0x03, 0x00}

		// Length of the record
		certLenDS := uint32(len(certDS))
		wiiCertLenDS := uint32(len(wiiCertDS))
		serverCertsRecordDS = append(serverCertsRecordDS, []byte{
			byte((certLenDS + wiiCertLenDS + 13) >> 8),
			byte(certLenDS + wiiCertLenDS + 13),
		}...)

		serverCertsRecordDS = append(serverCertsRecordDS, 0xB)

		serverCertsRecordDS = append(serverCertsRecordDS, []byte{
			byte((certLenDS + wiiCertLenDS + 9) >> 16),
			byte((certLenDS + wiiCertLenDS + 9) >> 8),
			byte(certLenDS + wiiCertLenDS + 9),
		}...)

		serverCertsRecordDS = append(serverCertsRecordDS, []byte{
			byte((certLenDS + wiiCertLenDS + 6) >> 16),
			byte((certLenDS + wiiCertLenDS + 6) >> 8),
			byte(certLenDS + wiiCertLenDS + 6),
		}...)

		serverCertsRecordDS = append(serverCertsRecordDS, []byte{
			byte(certLenDS >> 16),
			byte(certLenDS >> 8),
			byte(certLenDS),
		}...)

		serverCertsRecordDS = append(serverCertsRecordDS, certDS...)

		serverCertsRecordDS = append(serverCertsRecordDS, []byte{
			byte(wiiCertLenDS >> 16),
			byte(wiiCertLenDS >> 8),
			byte(wiiCertLenDS),
		}...)

		serverCertsRecordDS = append(serverCertsRecordDS, wiiCertDS...)

		serverCertsRecordDS = append(serverCertsRecordDS, []byte{
			0x16, 0x03, 0x00, 0x00, 0x04, 0x0E, 0x00, 0x00, 0x00,
		}...)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			panic(err)
		}

		go func() {
			// logging.Info("NAS-TLS", "Receiving HTTPS request from", aurora.BrightCyan(conn.RemoteAddr()))

			moduleName := "NAS-TLS:" + conn.RemoteAddr().String()

			conn.SetDeadline(time.Now().Add(5 * time.Second))

			handleTLS(moduleName, conn, nasAddr, serverCertsRecordWii, rsaKeyWii, serverCertsRecordDS, rsaKeyDS)
		}()
	}
}

// handleTLS handles the TLS request from the Wii or the DS. It may call handleRealTLS if the request is from a modern web browser.
func handleTLS(moduleName string, rawConn net.Conn, nasAddr string, serverCertsRecordWii []byte, rsaKeyWii *rsa.PrivateKey, serverCertsRecordDS []byte, rsaKeyDS *rsa.PrivateKey) {
	// Recover from panics
	defer func() {
		if r := recover(); r != nil {
			logging.Error(moduleName, "Panic:", r)
		}
	}()

	conn := newBufferedConn(rawConn)

	defer conn.Close()

	// Read client hello
	// fmt.Printf("Client Hello:\n")
	var helloBytes []byte
	if rsaKeyWii != nil {
		index := 0
		for index = 0; index < 0x1D; index++ {
			var err error
			helloBytes, err = conn.Peek(index + 1)
			if err != nil {
				logging.Error(moduleName, "Failed to peek from client:", err)
				return
			}

			if helloBytes[index] != []byte{
				0x80, 0x2B, 0x01, 0x03, 0x01, 0x00, 0x12, 0x00, 0x00, 0x00, 0x10, 0x00,
				0x00, 0x35, 0x00, 0x00, 0x2F, 0x00, 0x00, 0x0A, 0x00, 0x00, 0x09, 0x00,
				0x00, 0x05, 0x00, 0x00, 0x04,
			}[index] {
				break
			}
		}
		if index == 0x1D {
			macFn, cipher, clientCipher, err := handleWiiTLSHandshake(moduleName, conn, serverCertsRecordWii, rsaKeyWii)
			if err == nil && macFn != nil && cipher != nil && clientCipher != nil {
				proxyConsoleTLS(moduleName, conn, nasAddr, VersionTLS10, macFn, cipher, clientCipher)
			}
			return
		}
	}

	if rsaKeyDS != nil {
		index := 0
		for index = 0; index < 0x0B; index++ {
			var err error
			helloBytes, err = conn.Peek(index + 1)
			if err != nil {
				logging.Error(moduleName, "Failed to peek from client:", err)
				return
			}

			if helloBytes[index] != []byte{
				0x16, 0x03, 0x00, 0x00, 0x2F, 0x01, 0x00, 0x00, 0x2B, 0x03, 0x00,
			}[index] {
				break
			}
		}
		if index == 0x0B {
			macFn, cipher, clientCipher, err := handleDSSSLHandshake(moduleName, conn, serverCertsRecordDS, rsaKeyDS)
			if err == nil && macFn != nil && cipher != nil && clientCipher != nil {
				proxyConsoleTLS(moduleName, conn, nasAddr, VersionSSL30, macFn, cipher, clientCipher)
			}
			return
		}
	}

	conn.SetDeadline(time.Now().Add(25 * time.Second))

	// logging.Info(moduleName, "Forwarding client hello:", aurora.Cyan(fmt.Sprintf("% X ", helloBytes)))
	handleRealTLS(moduleName, conn, nasAddr)
}

func handleWiiTLSHandshake(moduleName string, conn bufferedConn, serverCertsRecord []byte, rsaKey *rsa.PrivateKey) (macFn macFunction, cipher *rc4.Cipher, clientCipher *rc4.Cipher, err error) {
	// fmt.Printf("\n")

	clientHello := make([]byte, 0x2D)
	_, err = io.ReadFull(conn.r, clientHello)
	if err != nil {
		logging.Error(moduleName, "Failed to read from client:", err)
		return
	}

	finishHash := newFinishedHash(VersionTLS10)
	finishHash.Write(clientHello[0x2:0x2D])

	// The random bytes are padded to 32 bytes with 0x00 (data is right justified)
	clientRandom := append(make([]byte, 16), clientHello[0x1D:0x1D+0x10]...)

	serverHello := []byte{0x16, 0x03, 0x01, 0x00, 0x2A, 0x02, 0x00, 0x00, 0x26, 0x03, 0x01}

	serverRandom := make([]byte, 0x20)
	_, err = rand.Read(serverRandom)
	if err != nil {
		logging.Error(moduleName, "Failed to generate random bytes:", err)
		return
	}

	serverHello = append(serverHello, serverRandom...)

	// Send an empty session ID
	serverHello = append(serverHello, 0x00)

	// Select cipher suite TLS_RSA_WITH_RC4_128_MD5 (0x0004)
	serverHello = append(serverHello, []byte{
		0x00, 0x04, 0x00,
	}...)

	// Append the certs record to the server hello buffer
	serverHello = append(serverHello, serverCertsRecord...)

	// fmt.Printf("Server Hello:\n% X\n", serverHello)

	finishHash.Write(serverHello[0x5:0x2F])
	finishHash.Write(serverHello[0x34 : 0x34+(len(serverCertsRecord)-14)])
	finishHash.Write(serverHello[0x34+(len(serverCertsRecord)-14)+5 : 0x34+(len(serverCertsRecord)-14)+5+4])

	_, err = conn.Write(serverHello)
	if err != nil {
		logging.Error(moduleName, "Failed to write to client:", err)
		return
	}

	// fmt.Printf("Client key exchange:\n")
	buf := make([]byte, 0x1000)
	index := 0
	// Read client key exchange (+ change cipher spec + finished)
	for {
		var n int
		n, err = conn.Read(buf[index:])
		if err != nil {
			logging.Error(moduleName, "Failed to read from client:", err)
			return
		}

		// fmt.Printf("% X ", buf[index:index+n])
		index += n

		// Check client key exchange header
		if !bytes.HasPrefix([]byte{
			0x16, 0x03, 0x01, 0x00, 0x86, 0x10, 0x00, 0x00, 0x82, 0x00, 0x80,
		}, buf[:min(index, 0x0B)]) {
			logging.Error(moduleName, "Invalid client key exchange header:", aurora.Cyan(fmt.Sprintf("% X ", buf[:min(index, 0x0B)])))
			err = errors.New("invalid client key exchange header")
			return
		}

		if index > 0x8B {
			// Check change cipher spec + finished header
			if !bytes.HasPrefix(buf[0x8B:min(index, 0x8B+0x0B)], []byte{
				0x14, 0x03, 0x01, 0x00, 0x01, 0x01, 0x16, 0x03, 0x01, 0x00, 0x20,
			}) {
				logging.Error(moduleName, "Invalid client change cipher spec + finished header:", aurora.Cyan(fmt.Sprintf("%X ", buf[0x8B:min(index, 0x8B+0x0B)])))
				err = errors.New("invalid client change cipher spec + finished header")
				return
			}
		}

		if index == 0xB6 {
			buf = buf[:index]
			break
		}

		if index > 0xB6 {
			logging.Error(moduleName, "Invalid client key exchange length:", aurora.BrightCyan(index))
			err = errors.New("invalid client key exchange length")
			return
		}
	}
	// fmt.Printf("\n")

	encryptedPreMasterSecret := buf[0x0B : 0x0B+0x80]
	clientFinish := buf[0x96 : 0x96+0x20]

	finishHash.Write(buf[0x5 : 0x5+0x86])

	// Decrypt the pre master secret using our RSA key
	preMasterSecret, err := rsa.DecryptPKCS1v15(rand.Reader, rsaKey, encryptedPreMasterSecret)
	if err != nil {
		logging.Error(moduleName, "Failed to decrypt pre master secret:", err)
		return
	}

	// fmt.Printf("Pre master secret:\n% X\n", preMasterSecret)

	if len(preMasterSecret) != 48 {
		logging.Error(moduleName, "Invalid pre master secret length:", aurora.BrightCyan(len(preMasterSecret)))
		err = errors.New("invalid pre master secret length")
		return
	}

	if !bytes.Equal(preMasterSecret[:2], []byte{0x03, 0x01}) {
		logging.Error(moduleName, "Invalid TLS version in pre master secret:", aurora.BrightCyan(preMasterSecret[:2]))
		err = errors.New("invalid TLS version in pre master secret")
		return
	}

	clientServerRandom := append(bytes.Clone(clientRandom), serverRandom[:0x20]...)

	masterSecret := make([]byte, 48)
	prf10(masterSecret, preMasterSecret, []byte("master secret"), clientServerRandom)

	// fmt.Printf("Master secret:\n% X\n", masterSecret)

	_, serverMAC, clientKey, serverKey, _, _ := keysFromMasterSecret(VersionTLS10, masterSecret, clientRandom, serverRandom, 16, 16, 16)

	// fmt.Printf("Client MAC:\n% X\n", clientMAC)
	// fmt.Printf("Server MAC:\n% X\n", serverMAC)
	// fmt.Printf("Client key:\n% X\n", clientKey)
	// fmt.Printf("Server key:\n% X\n", serverKey)
	// fmt.Printf("Client IV:\n% X\n", clientIV)
	// fmt.Printf("Server IV:\n% X\n", serverIV)

	// Create the server RC4 cipher
	cipher, err = rc4.NewCipher(serverKey)
	if err != nil {
		panic(err)
	}

	// Create the client RC4 cipher
	clientCipher, err = rc4.NewCipher(clientKey)
	if err != nil {
		panic(err)
	}

	// Create the hmac cipher
	macFn = macMD5(VersionTLS10, serverMAC)

	// Decrypt client finish
	clientCipher.XORKeyStream(clientFinish, clientFinish)
	finishHash.Write(clientFinish[:0x10])

	// fmt.Printf("Client Finish:\n% X\n", clientFinish)

	// Send ChangeCipherSpec
	_, err = conn.Write([]byte{0x14, 0x03, 0x01, 0x00, 0x01, 0x01})
	if err != nil {
		panic(err)
	}

	finishedRecord := []byte{0x16, 0x03, 0x01, 0x00, 0x10}

	out := finishHash.serverSum(masterSecret)

	// Encrypt the finished record
	finishedRecord, _ = encryptTLS(macFn, cipher, append([]byte{0x14, 0x00, 0x00, 0x0C}, out[:12]...), 0, finishedRecord)

	_, err = conn.Write(finishedRecord)

	return
}

func handleDSSSLHandshake(moduleName string, conn bufferedConn, serverCertsRecord []byte, rsaKey *rsa.PrivateKey) (macFn macFunction, cipher *rc4.Cipher, clientCipher *rc4.Cipher, err error) {
	clientHello := make([]byte, 0x34)
	_, err = io.ReadFull(conn.r, clientHello)
	if err != nil {
		logging.Error(moduleName, "Failed to read from client:", err)
		return
	}

	finishHash := newFinishedHash(VersionSSL30)
	finishHash.Write(clientHello[0x5:0x34])

	clientRandom := clientHello[0x0b : 0x0b+0x20]

	serverHello := []byte{0x16, 0x03, 0x00, 0x00, 0x2A, 0x02, 0x00, 0x00, 0x26, 0x03, 0x00}

	serverRandom := make([]byte, 0x20)
	_, err = rand.Read(serverRandom)
	if err != nil {
		logging.Error(moduleName, "Failed to generate random bytes:", err)
		return
	}

	serverHello = append(serverHello, serverRandom...)

	// Send an empty session ID
	serverHello = append(serverHello, 0x00)

	// Select cipher suite TLS_RSA_WITH_RC4_128_MD5 (0x0004)
	serverHello = append(serverHello, []byte{
		0x00, 0x04, 0x00,
	}...)

	// Append the certs record to the server hello buffer
	serverHello = append(serverHello, serverCertsRecord...)

	// fmt.Printf("Server Hello:\n% X\n", serverHello)

	finishHash.Write(serverHello[0x5:0x2F])
	finishHash.Write(serverHello[0x34 : 0x34+(len(serverCertsRecord)-14)])
	finishHash.Write(serverHello[0x34+(len(serverCertsRecord)-14)+5 : 0x34+(len(serverCertsRecord)-14)+5+4])

	_, err = conn.Write(serverHello)
	if err != nil {
		logging.Error(moduleName, "Failed to write to client:", err)
		return
	}

	// fmt.Printf("Client key exchange:\n")
	buf := make([]byte, 0x1000)
	index := 0
	// Read client key exchange (+ change cipher spec + finished)
	for {
		var n int
		n, err = conn.Read(buf[index:])
		if err != nil {
			logging.Error(moduleName, "Failed to read from client:", err)
			return
		}

		// fmt.Printf("% X ", buf[index:index+n])
		index += n

		// Check client key exchange header
		if !bytes.HasPrefix([]byte{
			0x16, 0x03, 0x00, 0x00, 0x84, 0x10, 0x00, 0x00, 0x80,
		}, buf[:min(index, 0x09)]) {
			logging.Error(moduleName, "Invalid client key exchange header:", aurora.Cyan(fmt.Sprintf("% X ", buf[:min(index, 0x09)])))
			err = errors.New("invalid client key exchange header")
			return
		}

		if index > 0x8B {
			// Check change cipher spec + finished header
			if !bytes.HasPrefix(buf[0x89:min(index, 0x89+0x0B)], []byte{
				0x14, 0x03, 0x00, 0x00, 0x01, 0x01, 0x16, 0x03, 0x00, 0x00, 0x38,
			}) {
				logging.Error(moduleName, "Invalid client change cipher spec + finished header:", aurora.Cyan(fmt.Sprintf("%X ", buf[0x89:min(index, 0x89+0x0B)])))
				err = errors.New("invalid client change cipher spec + finished header")
				return
			}
		}

		if index == 0xCC {
			buf = buf[:index]
			break
		}

		if index > 0xCC {
			logging.Error(moduleName, "Invalid client key exchange length:", aurora.BrightCyan(index))
			err = errors.New("invalid client key exchange length")
			return
		}
	}
	// fmt.Printf("\n")

	encryptedPreMasterSecret := buf[0x09 : 0x09+0x80]
	clientFinish := buf[0x94 : 0x94+0x38]

	finishHash.Write(buf[0x5 : 0x5+0x84])

	// Decrypt the pre master secret using our RSA key
	preMasterSecret, err := rsa.DecryptPKCS1v15(rand.Reader, rsaKey, encryptedPreMasterSecret)
	if err != nil {
		logging.Error(moduleName, "Failed to decrypt pre master secret:", err)
		return
	}

	// fmt.Printf("Pre master secret:\n% X\n", preMasterSecret)

	if len(preMasterSecret) != 48 {
		logging.Error(moduleName, "Invalid pre master secret length:", aurora.BrightCyan(len(preMasterSecret)))
		err = errors.New("invalid pre master secret length")
		return
	}

	if !bytes.Equal(preMasterSecret[:2], []byte{0x03, 0x00}) {
		logging.Error(moduleName, "Invalid TLS version in pre master secret:", aurora.BrightCyan(preMasterSecret[:2]))
		err = errors.New("invalid TLS version in pre master secret")
		return
	}

	clientServerRandom := append(bytes.Clone(clientRandom), serverRandom[:0x20]...)

	masterSecret := make([]byte, 48)
	prf30(masterSecret, preMasterSecret, []byte("master secret"), clientServerRandom)

	// fmt.Printf("Master secret:\n% X\n", masterSecret)

	_, serverMAC, clientKey, serverKey, _, _ := keysFromMasterSecret(VersionSSL30, masterSecret, clientRandom, serverRandom, 16, 16, 16)

	// fmt.Printf("Client MAC:\n% X\n", clientMAC)
	// fmt.Printf("Server MAC:\n% X\n", serverMAC)
	// fmt.Printf("Client key:\n% X\n", clientKey)
	// fmt.Printf("Server key:\n% X\n", serverKey)
	// fmt.Printf("Client IV:\n% X\n", clientIV)
	// fmt.Printf("Server IV:\n% X\n", serverIV)

	// Create the server RC4 cipher
	cipher, err = rc4.NewCipher(serverKey)
	if err != nil {
		panic(err)
	}

	// Create the client RC4 cipher
	clientCipher, err = rc4.NewCipher(clientKey)
	if err != nil {
		panic(err)
	}

	// Create the mac function
	macFn = macMD5(VersionSSL30, serverMAC)

	// Decrypt client finish
	clientCipher.XORKeyStream(clientFinish, clientFinish)
	finishHash.Write(clientFinish[:0x28])

	// fmt.Printf("Client Finish:\n% X\n", clientFinish)

	// Send ChangeCipherSpec
	_, err = conn.Write([]byte{0x14, 0x03, 0x00, 0x00, 0x01, 0x01})
	if err != nil {
		panic(err)
	}

	finishedRecord := []byte{0x16, 0x03, 0x00, 0x00, 0x28}

	out := finishHash.serverSum(masterSecret)

	// Encrypt the finished record
	finishedRecord, _ = encryptTLS(macFn, cipher, append([]byte{0x14, 0x00, 0x00, 0x24}, out[:36]...), 0, finishedRecord)

	_, err = conn.Write(finishedRecord)

	return
}

func proxyConsoleTLS(moduleName string, conn bufferedConn, nasAddr string, version uint16, macFn macFunction, cipher *rc4.Cipher, clientCipher *rc4.Cipher) {
	// Open a connection to NAS
	newConn, err := net.Dial("tcp", nasAddr)
	if err != nil {
		panic(err)
	}

	defer newConn.Close()

	// Read bytes from the HTTP server and forward them through the TLS connection
	go func() {
		recvBuf := make([]byte, 0x100)

		seq := uint64(1)
		for {
			n, err := newConn.Read(recvBuf)
			if err != nil {
				if errors.Is(err, io.EOF) || strings.Contains(err.Error(), "use of closed network connection") {
					return
				}

				logging.Error(moduleName, "Failed to read from HTTP server:", err)
				return
			}

			// fmt.Printf("Sent:\n% X ", recvBuf[:n])
			var record []byte
			record, seq = encryptTLS(macFn, cipher, recvBuf[:n], seq, []byte{0x17, 0x03, 0x01, byte(n >> 8), byte(n)})

			_, err = conn.Write(record)
			if err != nil {
				logging.Error(moduleName, "Failed to write to client:", err)
				return
			}
		}
	}()

	// Read encrypted content from the client and forward it to the HTTP server
	index := 0
	total := 0
	buf := make([]byte, 0x1000)
	for {
		n, err := conn.Read(buf[index:])
		if err != nil {
			if errors.Is(err, io.EOF) || strings.Contains(err.Error(), "use of closed network connection") {
				logging.Info(moduleName, "Connection closed by client after", aurora.BrightCyan(total), "bytes")
				return
			}

			logging.Error(moduleName, "Failed to read from client:", err)
			return
		}

		// fmt.Printf("Received:\n% X ", buf[index:index+n])
		index += n
		total += n

		for {
			if index < 5 {
				break
			}

			if buf[0] < 0x15 || buf[0] > 0x17 {
				logging.Error(moduleName, "Invalid record type")
				return
			}

			if buf[1] != 0x03 || (version == VersionTLS10 && buf[2] != 0x01) || (version == VersionSSL30 && buf[2] != 0x00) {
				logging.Error(moduleName, "Invalid TLS version")
				return
			}

			recordLength := binary.BigEndian.Uint16(buf[3:5])
			if recordLength < 17 || (recordLength+5) > 0x1000 {
				logging.Error(moduleName, "Invalid record length")
				return
			}

			if index < int(recordLength)+5 {
				break
			}

			// Decrypt content
			clientCipher.XORKeyStream(buf[5:5+recordLength], buf[5:5+recordLength])
			// fmt.Printf("\nDecrypted content:\n% X \n", buf[5:5+recordLength])

			if buf[0] != 0x17 {
				if buf[0] == 0x15 || buf[5] == 0x01 || buf[6] == 0x00 {
					logging.Info(moduleName, "Alert connection close by client after", aurora.BrightCyan(total), "bytes")
					return
				}

				logging.Error(moduleName, "Non-application data received:", aurora.Cyan(fmt.Sprintf("% X ", buf[:5+recordLength])))
				return
			} else {
				// Send the decrypted content to the HTTP server
				_, err = newConn.Write(buf[5 : 5+recordLength-16])
				if err != nil {
					logging.Error(moduleName, "Failed to write to HTTP server:", err)
					return
				}
			}

			buf = buf[5+recordLength:]
			buf = append(buf, make([]byte, 0x1000-len(buf))...)
			index -= 5 + int(recordLength)
		}
	}
}

var realTLSConfig *tls.Config

func setupRealTLS(privKeyPath string, certsPath string) {
	// Read server key and certs

	serverKey, err := os.ReadFile(privKeyPath)
	if err != nil {
		logging.Error("NAS-TLS", "Failed to read server key:", err)
		return
	}

	serverCerts, err := os.ReadFile(certsPath)
	if err != nil {
		logging.Error("NAS-TLS", "Failed to read server certs:", err)
		return
	}

	cert, err := tls.X509KeyPair(serverCerts, serverKey)
	if err != nil {
		logging.Error("NAS-TLS", "Failed to parse server certs:", err)
		return
	}

	config := tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	realTLSConfig = &config
}

// handleRealTLS handles the TLS request legitimately using crypto/tls
func handleRealTLS(moduleName string, conn net.Conn, nasAddr string) {
	// Recover from panics
	defer func() {
		if r := recover(); r != nil {
			logging.Error(moduleName, "Panic:", r)
		}
	}()

	if realTLSConfig == nil {
		return
	}

	tlsConn := tls.Server(conn, realTLSConfig)

	err := tlsConn.Handshake()
	if err != nil {
		return
	}

	newConn, err := net.Dial("tcp", nasAddr)
	if err != nil {
		panic(err)
	}

	defer newConn.Close()

	// Read bytes from the HTTP server and forward them through the TLS connection
	go func() {
		recvBuf := make([]byte, 0x100)

		for {
			n, err := newConn.Read(recvBuf)
			if err != nil {
				return
			}

			_, err = tlsConn.Write(recvBuf[:n])
			if err != nil {
				logging.Error(moduleName, "Failed to write to client:", err)
				return
			}
		}
	}()

	// Read encrypted content from the client and forward it to the HTTP server
	buf := make([]byte, 0x1000)
	for {
		n, err := tlsConn.Read(buf)
		if err != nil {
			return
		}

		_, err = newConn.Write(buf[:n])
		if err != nil {
			logging.Error(moduleName, "Failed to write to HTTP server:", err)
			return
		}
	}
}

// The following functions are modified from the crypto standard library
//
// Copyright (c) 2009 The Go Authors. All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//    * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//    * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//    * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

const (
	VersionSSL30 = 0x0300
	VersionTLS10 = 0x0301
)

// Split a premaster secret in two as specified in RFC 4346, Section 5.
func splitPreMasterSecret(secret []byte) (s1, s2 []byte) {
	s1 = secret[0 : (len(secret)+1)/2]
	s2 = secret[len(secret)/2:]
	return
}

// pHash implements the P_hash function, as defined in RFC 4346, Section 5.
func pHash(result, secret, seed []byte, hash func() hash.Hash) {
	h := hmac.New(hash, secret)
	h.Write(seed)
	a := h.Sum(nil)

	j := 0
	for j < len(result) {
		h.Reset()
		h.Write(a)
		h.Write(seed)
		b := h.Sum(nil)
		copy(result[j:], b)
		j += len(b)

		h.Reset()
		h.Write(a)
		a = h.Sum(nil)
	}
}

// prf10 implements the TLS 1.0 pseudo-random function, as defined in RFC 2246, Section 5.
func prf10(result, secret, label, seed []byte) {
	hashSHA1 := sha1.New
	hashMD5 := md5.New
	labelAndSeed := make([]byte, len(label)+len(seed))

	copy(labelAndSeed, label)
	copy(labelAndSeed[len(label):], seed)
	s1, s2 := splitPreMasterSecret(secret)

	pHash(result, s1, labelAndSeed, hashMD5)
	result2 := make([]byte, len(result))
	pHash(result2, s2, labelAndSeed, hashSHA1)

	for i, b := range result2 {
		result[i] ^= b
	}
}

// prf30 implements the SSL 3.0 pseudo-random function, as defined in
// www.mozilla.org/projects/security/pki/nss/ssl/draft302.txt section 6.
func prf30(result, secret, label, seed []byte) {
	hashSHA1 := sha1.New()
	hashMD5 := md5.New()

	done := 0
	i := 0
	// RFC5246 section 6.3 says that the largest PRF output needed is 128
	// bytes. Since no more ciphersuites will be added to SSLv3, this will
	// remain true. Each iteration gives us 16 bytes so 10 iterations will
	// be sufficient.
	var b [11]byte
	for done < len(result) {
		for j := 0; j <= i; j++ {
			b[j] = 'A' + byte(i)
		}

		hashSHA1.Reset()
		hashSHA1.Write(b[:i+1])
		hashSHA1.Write(secret)
		hashSHA1.Write(seed)
		digest := hashSHA1.Sum(nil)

		hashMD5.Reset()
		hashMD5.Write(secret)
		hashMD5.Write(digest)

		done += copy(result[done:], hashMD5.Sum(nil))
		i++
	}
}

// keysFromMasterSecret generates the connection keys from the master
// secret, given the lengths of the MAC key, cipher key and IV, as defined in
// RFC 2246, Section 6.3.
func keysFromMasterSecret(version uint16, masterSecret, clientRandom, serverRandom []byte, macLen, keyLen, ivLen int) (clientMAC, serverMAC, clientKey, serverKey, clientIV, serverIV []byte) {
	prf := prf10
	if version == VersionSSL30 {
		prf = prf30
	}

	seed := make([]byte, 0, len(serverRandom)+len(clientRandom))
	seed = append(seed, serverRandom...)
	seed = append(seed, clientRandom...)

	n := 2*macLen + 2*keyLen + 2*ivLen
	keyMaterial := make([]byte, n)
	prf(keyMaterial, masterSecret, []byte("key expansion"), seed)
	clientMAC = keyMaterial[:macLen]
	keyMaterial = keyMaterial[macLen:]
	serverMAC = keyMaterial[:macLen]
	keyMaterial = keyMaterial[macLen:]
	clientKey = keyMaterial[:keyLen]
	keyMaterial = keyMaterial[keyLen:]
	serverKey = keyMaterial[:keyLen]
	keyMaterial = keyMaterial[keyLen:]
	clientIV = keyMaterial[:ivLen]
	keyMaterial = keyMaterial[ivLen:]
	serverIV = keyMaterial[:ivLen]
	return
}

func newFinishedHash(version uint16) finishedHash {
	return finishedHash{sha1.New(), sha1.New(), md5.New(), md5.New(), version}
}

// A finishedHash calculates the hash of a set of handshake messages suitable
// for including in a Finished message.
type finishedHash struct {
	client hash.Hash
	server hash.Hash

	// Prior to TLS 1.2, an additional MD5 hash is required.
	clientMD5 hash.Hash
	serverMD5 hash.Hash

	version uint16
}

func (h *finishedHash) Write(msg []byte) int {
	// fmt.Printf("Write finished hash: % X\n", msg)

	h.client.Write(msg)
	h.server.Write(msg)

	h.clientMD5.Write(msg)
	h.serverMD5.Write(msg)

	return len(msg)
}

func (h finishedHash) Sum() []byte {
	out := make([]byte, 0, md5.Size+sha1.Size)
	out = h.clientMD5.Sum(out)
	return h.client.Sum(out)
}

// finishedSum30 calculates the contents of the verify_data member of a SSLv3
// Finished message given the MD5 and SHA1 hashes of a set of handshake
// messages.
func finishedSum30(md5, sha1 hash.Hash, masterSecret []byte, magic [4]byte) []byte {
	md5.Write(magic[:])
	md5.Write(masterSecret)
	md5.Write(ssl30Pad1[:])
	md5Digest := md5.Sum(nil)

	md5.Reset()
	md5.Write(masterSecret)
	md5.Write(ssl30Pad2[:])
	md5.Write(md5Digest)
	md5Digest = md5.Sum(nil)

	sha1.Write(magic[:])
	sha1.Write(masterSecret)
	sha1.Write(ssl30Pad1[:40])
	sha1Digest := sha1.Sum(nil)

	sha1.Reset()
	sha1.Write(masterSecret)
	sha1.Write(ssl30Pad2[:40])
	sha1.Write(sha1Digest)
	sha1Digest = sha1.Sum(nil)

	ret := make([]byte, len(md5Digest)+len(sha1Digest))
	copy(ret, md5Digest)
	copy(ret[len(md5Digest):], sha1Digest)
	return ret
}

// serverSum returns the contents of the verify_data member of a server's
// Finished message.
func (h finishedHash) serverSum(masterSecret []byte) []byte {
	if h.version == VersionSSL30 {
		return finishedSum30(h.serverMD5, h.server, masterSecret, [4]byte{0x53, 0x52, 0x56, 0x52})
	}

	out := make([]byte, 12)
	prf10(out, masterSecret, []byte("server finished"), h.Sum())
	return out
}

func encryptTLS(macFn macFunction, cipher *rc4.Cipher, payload []byte, seq uint64, record []byte) ([]byte, uint64) {
	mac := macFn.MAC([]byte{}, binary.BigEndian.AppendUint64([]byte{}, seq), record[:5], payload, nil)

	record = append(append(bytes.Clone(record[:5]), payload...), mac...)
	cipher.XORKeyStream(record[5:], record[5:])

	// Update length to include nonce, MAC and any block padding needed.
	n := len(record) - 5
	record[3] = byte(n >> 8)
	record[4] = byte(n)

	return record, seq + 1
}

type macFunction interface {
	MAC(out, seq, header, data, extra []byte) []byte
}

func macMD5(version uint16, key []byte) macFunction {
	if version == VersionSSL30 {
		mac := ssl30MAC{
			h:   md5.New(),
			key: make([]byte, len(key)),
		}
		copy(mac.key, key)
		return mac
	}
	return tls10MAC{h: hmac.New(md5.New, key)}
}

// tls10MAC implements the TLS 1.0 MAC function. RFC 2246, Section 6.2.3.
type tls10MAC struct {
	h hash.Hash
}

func (s tls10MAC) MAC(out, seq, header, data, extra []byte) []byte {
	s.h.Reset()
	s.h.Write(seq)
	s.h.Write(header)
	s.h.Write(data)
	res := s.h.Sum(out)
	if extra != nil {
		s.h.Write(extra)
	}
	return res
}

// ssl30MAC implements the SSLv3 MAC function, as defined in
// www.mozilla.org/projects/security/pki/nss/ssl/draft302.txt section 5.2.3.1
type ssl30MAC struct {
	h   hash.Hash
	key []byte
}

var ssl30Pad1 = [48]byte{0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36, 0x36}

var ssl30Pad2 = [48]byte{0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c, 0x5c}

func (s ssl30MAC) MAC(out, seq, header, data []byte, extra []byte) []byte {
	padLength := 48
	if s.h.Size() == 20 {
		padLength = 40
	}

	s.h.Reset()
	s.h.Write(s.key)
	s.h.Write(ssl30Pad1[:padLength])
	s.h.Write(seq)
	s.h.Write(header[:1])
	s.h.Write(header[3:5])
	s.h.Write(data)
	out = s.h.Sum(out[:0])

	s.h.Reset()
	s.h.Write(s.key)
	s.h.Write(ssl30Pad2[:padLength])
	s.h.Write(out)
	return s.h.Sum(out[:0])
}
