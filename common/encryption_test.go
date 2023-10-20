package common

import (
	"fmt"
	"testing"
)

func TestEncryption(t *testing.T) {
	encrypted := EncryptTypeX([]byte("key"), []byte("challenge"), []byte("data"))
	for i := 0; i < len(encrypted); i++ {
		fmt.Printf("%02x ", encrypted[i])
	}
	fmt.Printf("\n")
}
