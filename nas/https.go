package nas

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/rc4"
	"crypto/rsa"
	"crypto/sha1"
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
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

// Bare minimum TLS 1.0 server implementation for the Wii's /dev/net/ssl client
// Use this with a certificate that exploits the Wii's SSL certificate bug to impersonate naswii.nintendowifi.net
// See here: https://github.com/shutterbug2000/wii-ssl-bug

// Don't use this for anything else, it's not secure

func startHTTPSProxy(address string, nasAddr string) {
	cert, err := os.ReadFile("naswii-cert.der")
	if err != nil {
		panic(err)
	}

	rsaData, err := os.ReadFile("naswii-key.pem")
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
		panic("unexpected key type")
	}

	serverCertsRecord := []byte{0x16, 0x03, 0x01}

	// Length of the record
	certLen := uint32(len(cert))
	serverCertsRecord = append(serverCertsRecord, []byte{
		byte((certLen + 10) >> 8),
		byte(certLen + 10),
	}...)

	serverCertsRecord = append(serverCertsRecord, 0xB)

	serverCertsRecord = append(serverCertsRecord, []byte{
		byte((certLen + 6) >> 16),
		byte((certLen + 6) >> 8),
		byte(certLen + 6),
	}...)

	serverCertsRecord = append(serverCertsRecord, []byte{
		byte((certLen + 3) >> 16),
		byte((certLen + 3) >> 8),
		byte(certLen + 3),
	}...)

	serverCertsRecord = append(serverCertsRecord, []byte{
		byte(certLen >> 16),
		byte(certLen >> 8),
		byte(certLen),
	}...)

	serverCertsRecord = append(serverCertsRecord, cert...)

	serverCertsRecord = append(serverCertsRecord, []byte{
		0x16, 0x03, 0x01, 0x00, 0x04, 0x0E, 0x00, 0x00, 0x00,
	}...)

	logging.Notice("NAS-TLS", "Starting HTTPS server on", address)
	l, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			panic(err)
		}

		logging.Notice("NAS-TLS", "Receiving HTTPS request from", aurora.BrightCyan(conn.RemoteAddr()))
		moduleName := "NAS-TLS:" + conn.RemoteAddr().String()

		go func() {
			defer conn.Close()

			buf := make([]byte, 0x1000)
			index := 0

			// Read client hello
			// fmt.Printf("Client Hello:\n")
			for {
				n, err := conn.Read(buf[index:])
				if err != nil {
					logging.Error(moduleName, "Failed to read from client:", err)
					return
				}

				// fmt.Printf("% X ", buf[index:index+n])
				index += n

				if !bytes.HasPrefix([]byte{
					0x80, 0x2B, 0x01, 0x03, 0x01, 0x00, 0x12, 0x00, 0x00, 0x00, 0x10, 0x00,
					0x00, 0x35, 0x00, 0x00, 0x2F, 0x00, 0x00, 0x0A, 0x00, 0x00, 0x09, 0x00,
					0x00, 0x05, 0x00, 0x00, 0x04,
				}, buf[:min(index, 0x1D)]) {
					logging.Error(moduleName, "Invalid client hello:", aurora.Cyan(fmt.Sprintf("% X ", buf[:min(index, 0x1D)])))
					return
				}

				if index == 0x2D {
					buf = buf[:index]
					break
				}

				if index > 0x2D {
					logging.Error(moduleName, "Invalid client hello length:", aurora.BrightCyan(index))
					return
				}
			}
			// fmt.Printf("\n")

			clientHello := buf

			finishHash := newFinishedHash()
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
			finishHash.Write(serverHello[0x34 : 0x34+(certLen+10)])
			finishHash.Write(serverHello[0x34+(certLen+10)+5 : 0x34+(certLen+10)+5+4])

			_, err = conn.Write(serverHello)
			if err != nil {
				logging.Error(moduleName, "Failed to write to client:", err)
				return
			}

			// fmt.Printf("Client key exchange:\n")
			buf = make([]byte, 0x1000)
			index = 0
			// Read client key exchange (+ change cipher spec + finished)
			for {
				n, err := conn.Read(buf[index:])
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
					return
				}

				if index > 0x8B {
					// Check change cipher spec + finished header
					if !bytes.HasPrefix(buf[0x8B:min(index, 0x8B+0x0B)], []byte{
						0x14, 0x03, 0x01, 0x00, 0x01, 0x01, 0x16, 0x03, 0x01, 0x00, 0x20,
					}) {
						logging.Error(moduleName, "Invalid client change cipher spec + finished header:", aurora.Cyan(fmt.Sprintf("%X ", buf[0x8B:min(index, 0x8B+0x0B)])))
						return
					}
				}

				if index == 0xB6 {
					buf = buf[:index]
					break
				}

				if index > 0xB6 {
					logging.Error(moduleName, "Invalid client key exchange length:", aurora.BrightCyan(index))
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
				return
			}

			if !bytes.Equal(preMasterSecret[:2], []byte{0x03, 0x01}) {
				logging.Error(moduleName, "Invalid TLS version in pre master secret:", aurora.BrightCyan(preMasterSecret[:2]))
				return
			}

			clientServerRandom := append(bytes.Clone(clientRandom), serverRandom[:0x20]...)

			masterSecret := make([]byte, 48)
			prf10(masterSecret, preMasterSecret, []byte("master secret"), clientServerRandom)

			// fmt.Printf("Master secret:\n% X\n", masterSecret)

			_, serverMAC, clientKey, serverKey, _, _ := keysFromMasterSecret(masterSecret, clientRandom, serverRandom, 16, 16, 16)

			// fmt.Printf("Client MAC:\n% X\n", clientMAC)
			// fmt.Printf("Server MAC:\n% X\n", serverMAC)
			// fmt.Printf("Client key:\n% X\n", clientKey)
			// fmt.Printf("Server key:\n% X\n", serverKey)
			// fmt.Printf("Client IV:\n% X\n", clientIV)
			// fmt.Printf("Server IV:\n% X\n", serverIV)

			// Create the server RC4 cipher
			cipher, err := rc4.NewCipher(serverKey)
			if err != nil {
				panic(err)
			}

			// Create the client RC4 cipher
			clientCipher, err := rc4.NewCipher(clientKey)
			if err != nil {
				panic(err)
			}

			// Create the hmac cipher
			macFn := hmac.New(md5.New, serverMAC)

			// Create the hmac cipher
			// clientMacFn := hmac.New(md5.New, clientMAC)

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

			// Open a connection to NAS
			newConn, err := net.Dial("tcp", nasAddr)
			if err != nil {
				panic(err)
			}

			defer newConn.Close()

			// Read bytes from the HTTP server and forward them through the TLS connection
			go func() {
				buf := make([]byte, 0x1000)

				seq := uint64(1)
				index := 0
				for {
					n, err := newConn.Read(buf[index:])
					if err != nil {
						if errors.Is(err, io.EOF) || strings.Contains(err.Error(), "use of closed network connection") {
							return
						}

						logging.Error(moduleName, "Failed to read from HTTP server:", err)
						return
					}

					// fmt.Printf("Sent:\n% X ", buf[index:index+n])
					var record []byte
					record, seq = encryptTLS(macFn, cipher, buf[index:index+n], seq, []byte{0x17, 0x03, 0x01, byte(n >> 8), byte(n)})

					_, err = conn.Write(record)
					if err != nil {
						logging.Error(moduleName, "Failed to write to client:", err)
						return
					}
				}
			}()

			// Read encrypted content from the client and forward it to the HTTP server
			index = 0
			total := 0
			buf = make([]byte, 0x1000)
			for {
				n, err := conn.Read(buf[index:])
				if err != nil {
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

					if buf[1] != 0x03 || buf[2] != 0x01 {
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
						if buf[0] == 0x15 && buf[5] == 0x01 && buf[6] == 0x00 {
							return
						}
						logging.Error(moduleName, "Non-application data received")
						return
					}

					// Send the decrypted content to the HTTP server
					_, err = newConn.Write(buf[5 : 5+recordLength-16])
					if err != nil {
						logging.Error(moduleName, "Failed to write to HTTP server:", err)
						return
					}

					buf = buf[5+recordLength:]
					buf = append(buf, make([]byte, 0x1000-len(buf))...)
					index -= 5 + int(recordLength)
				}
			}
		}()
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

// keysFromMasterSecret generates the connection keys from the master
// secret, given the lengths of the MAC key, cipher key and IV, as defined in
// RFC 2246, Section 6.3.
func keysFromMasterSecret(masterSecret, clientRandom, serverRandom []byte, macLen, keyLen, ivLen int) (clientMAC, serverMAC, clientKey, serverKey, clientIV, serverIV []byte) {
	seed := make([]byte, 0, len(serverRandom)+len(clientRandom))
	seed = append(seed, serverRandom...)
	seed = append(seed, clientRandom...)

	n := 2*macLen + 2*keyLen + 2*ivLen
	keyMaterial := make([]byte, n)
	prf10(keyMaterial, masterSecret, []byte("key expansion"), seed)
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

func newFinishedHash() finishedHash {
	return finishedHash{sha1.New(), sha1.New(), md5.New(), md5.New(), prf10}
}

// A finishedHash calculates the hash of a set of handshake messages suitable
// for including in a Finished message.
type finishedHash struct {
	client hash.Hash
	server hash.Hash

	// Prior to TLS 1.2, an additional MD5 hash is required.
	clientMD5 hash.Hash
	serverMD5 hash.Hash

	prf func(result, secret, label, seed []byte)
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

// clientSum returns the contents of the verify_data member of a client's
// Finished message.
func (h finishedHash) clientSum(masterSecret []byte) []byte {
	out := make([]byte, 12)
	h.prf(out, masterSecret, []byte("client finished"), h.Sum())
	return out
}

// serverSum returns the contents of the verify_data member of a server's
// Finished message.
func (h finishedHash) serverSum(masterSecret []byte) []byte {
	out := make([]byte, 12)
	h.prf(out, masterSecret, []byte("server finished"), h.Sum())
	return out
}

func encryptTLS(macFn hash.Hash, cipher *rc4.Cipher, payload []byte, seq uint64, record []byte) ([]byte, uint64) {
	mac := tls10MAC(macFn, []byte{}, binary.BigEndian.AppendUint64([]byte{}, seq), record[:5], payload, nil)
	record = append(append(bytes.Clone(record[:5]), payload...), mac...)
	cipher.XORKeyStream(record[5:], record[5:])

	// Update length to include nonce, MAC and any block padding needed.
	n := len(record) - 5
	record[3] = byte(n >> 8)
	record[4] = byte(n)

	return record, seq + 1
}

// tls10MAC implements the TLS 1.0 MAC function. RFC 2246, Section 6.2.3.
func tls10MAC(h hash.Hash, out, seq, header, data, extra []byte) []byte {
	h.Reset()
	h.Write(seq)
	h.Write(header)
	h.Write(data)
	res := h.Sum(out)
	if extra != nil {
		h.Write(extra)
	}
	return res
}
