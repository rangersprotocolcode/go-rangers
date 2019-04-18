package utility

import (
	"fmt"
	"testing"
	"encoding/hex"
)

func TestByteToInt(t *testing.T) {

	var a uint32
	a = 16
	bytes := UInt32ToByte(a)
	i := ByteToInt(bytes)

	if i == 16 {
		fmt.Printf("OK")
	} else {
		fmt.Errorf("Failed")
	}
}

func TestHex(t *testing.T) {
	b, _ := hex.DecodeString("80000001")
	fmt.Printf("%v\n", b)
}
