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

func TestFormatNumberString(t *testing.T) {
	str := "1.23000"
	fmt.Println(formatNumberString(str, 9))

	str = "123456.0123"
	fmt.Println(formatNumberString(str, 30))

	str = "1234560"
	fmt.Println(formatNumberString(str, 30))

	str = "123456.123456789"
	fmt.Println(formatNumberString(str, 9))
}

func TestStrToBigInt(t *testing.T) {
	str := "1.23000"
	fmt.Println(str)
	bigInt, _ := StrToBigInt(str)
	fmt.Println(bigInt.String())
	fmt.Println(BigIntToStr(bigInt))
}

func TestStrToBigInt2(t *testing.T) {
	str := "10000100001000010000100001000.23000"
	fmt.Println(str)
	bigInt, _ := StrToBigInt(str)
	fmt.Println(bigInt.String())
	fmt.Println(BigIntToStr(bigInt))
}
