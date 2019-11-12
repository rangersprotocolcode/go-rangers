package utility

import (
	"fmt"
	"testing"
	"encoding/hex"
	"strconv"
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

func TestStrconv(t *testing.T) {
	f := float64(1.5) / 100000
	s := strconv.FormatFloat(f, 'f', -1, 64)
	fmt.Printf("s:%s\n", s)
}

func TestStrToBigInt(t *testing.T) {
	s := "100000000000000000000"
	b, _ := StrToBigInt(s)
	floatData := float64(b.Int64()) / 1000000000
	fmt.Printf("float:%v\n", floatData)
	str := strconv.FormatFloat(floatData, 'f', -1, 64)
	fmt.Printf("float str:%s\n", str)
}
