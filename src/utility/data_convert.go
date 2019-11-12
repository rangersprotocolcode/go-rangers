package utility

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"strings"
	"fmt"
)

func UInt32ToByte(i uint32) []byte {
	buf := bytes.NewBuffer([]byte{})
	binary.Write(buf, binary.BigEndian, i)
	return buf.Bytes()
}

func ByteToUInt32(b []byte) uint32 {
	buf := bytes.NewBuffer(b)
	var x uint32
	binary.Read(buf, binary.BigEndian, &x)
	return x
}

func IntToByte(i int) []byte {
	buf := bytes.NewBuffer([]byte{})
	binary.Write(buf, binary.BigEndian, i)
	return buf.Bytes()
}

func ByteToInt(b []byte) int {
	buf := bytes.NewBuffer(b)
	var x int
	binary.Read(buf, binary.BigEndian, &x)
	return x
}

func UInt64ToByte(i uint64) []byte {
	buf := bytes.NewBuffer([]byte{})
	binary.Write(buf, binary.BigEndian, i)
	return buf.Bytes()
}

func ByteToUInt64(b []byte) uint64 {
	buf := bytes.NewBuffer(b)
	var x uint64
	binary.Read(buf, binary.BigEndian, &x)
	return x
}

//"11.22"->11220000000
func StrToBigInt(s string) (*big.Int, error) {
	target, _, err := big.ParseFloat(s, 10, 128, big.ToNearestEven)
	if err != nil {
		return nil, err
	}

	base := new(big.Float)
	base.SetInt(big.NewInt(1000000000))

	target.Mul(target, base)
	result := new(big.Int)
	target.Int(result)

	return result, nil
}

// 11220000000->"11.22"
func BigIntToStr(number *big.Int) string {
	base := new(big.Float)
	base.SetInt(big.NewInt(1000000000))

	target := new(big.Float)
	target.SetPrec(128)
	target.SetMode(big.ToNearestEven)
	target.SetInt(number)

	target.Quo(target, base)

	return formatNumberString(target.Text('f', 128), 9)
}

func formatNumberString(x string, precision int) string {
	lastIndex := strings.Index(x, ".")
	if lastIndex < 0 {
		return x
	}

	first := x[:lastIndex]

	length := lastIndex + precision + 1
	if length > len(x) {
		length = len(x)
	}
	last := x[lastIndex+1 : length]

	return fmt.Sprintf("%s.%s", first, last)
}
