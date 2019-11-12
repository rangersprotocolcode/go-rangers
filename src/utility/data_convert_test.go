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

func TestStateMachineManager_GetType(t *testing.T) {
	str := "1.23456"
	fmt.Println(formatNumberString(str, 1))

	str = "123456.0123"
	fmt.Println(formatNumberString(str, 30))

	str = "1234560"
	fmt.Println(formatNumberString(str, 30))

	str = "123456.123456789"
	fmt.Println(formatNumberString(str, 9))
}
