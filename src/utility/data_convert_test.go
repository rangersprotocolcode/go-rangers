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

func TestStrToBigInt3(t *testing.T) {
	str := "1000000000000000000000000001000000000.1234567891"
	value, err := StrToBigInt(str)
	if err != nil {
		panic(err)
	}

	fmt.Println(str)
	fmt.Println(value.String())

	fmt.Println(BigIntToStr(value))
}

func TestStrToBigInt4(t *testing.T) {
	str := "0.1234567891"
	value, err := StrToBigInt(str)
	if err != nil {
		panic(err)
	}

	fmt.Println(str)
	fmt.Println(value.String())

	fmt.Println(BigIntToStr(value))
}

func TestStrToBigInt5(t *testing.T) {
	str := "-0.0092"
	value, err := StrToBigInt(str)
	if err != nil {
		panic(err)
	}

	fmt.Println(str)
	fmt.Println(value.String())

	fmt.Println(BigIntToStr(value))
}

func TestStrToBigInt6(t *testing.T) {
	str := "0.0"
	value, err := StrToBigInt(str)
	if err != nil {
		panic(err)
	}

	fmt.Println(str)
	fmt.Println(value.String())

	fmt.Println(bigIntToStr(value,0))
}
