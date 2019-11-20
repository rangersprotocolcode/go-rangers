package utility

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"fmt"
	"strings"
)

const zeroString = "0"

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
	// 空字符串，默认返回0
	if 0 == len(s) {
		return big.NewInt(0), nil
	}

	target, _, err := big.ParseFloat(s, 10, 256, big.ToNearestEven)
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

// 11220000000->"11.220000000"
func BigIntToStr(number *big.Int) string {
	if nil == number || 0 == number.Sign() {
		return zeroString
	}

	// 默认保留小数点9位
	return bigIntToStr(number, 9)
}

func bigIntToStr(n *big.Int, precision int) string {
	if nil == n || precision < 0 {
		return zeroString
	}

	// 绝对值字符串
	number := n.Abs(n).String()

	var starter, first, last string

	// 负数
	if n.Sign() < 0 {
		starter = "-"
	}

	length := len(number)
	// 小于1的数
	if length <= precision {
		first = zeroString
		last = fmt.Sprintf("%s%s", strings.Repeat(zeroString, precision-length), number)
	} else {
		first = number[:length-precision]
		last = number[length-precision : length]
	}

	if 0 == precision {
		return fmt.Sprintf("%s%s", starter, first)
	}
	return fmt.Sprintf("%s%s.%s", starter, first, last)
}
