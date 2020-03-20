package common

import (
	"fmt"
	"golang.org/x/crypto/sha3"
	"testing"
)

func TestSha256(t *testing.T) {
	hash := sha3.Sum256([]byte{})
	fmt.Println(Bytes2Hex(hash[:]))

	hash2 := sha3.Sum256([]byte("test"))
	fmt.Println(Bytes2Hex(hash2[:]))
}
