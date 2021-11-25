// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package utility

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"
	"strings"
)

const (
	zeroString     = "0"
	prec           = 512
	baseNumber     = 1000000000000000000
	defaultDecimal = 18
)

var ten = big.NewInt(10)

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

func BigIntBytesToStr(value []byte) string {
	amount := new(big.Int)
	amount.SetBytes(value)

	return BigIntToStr(amount)
}

//"11.22"->11220000000
func StrToBigInt(s string) (*big.Int, error) {
	return strToBigInt(s, defaultDecimal)
}

//"11.22"->11220000000
func strToBigInt(s string, decimal int64) (*big.Int, error) {
	// 空字符串，默认返回0
	if 0 == len(s) {
		return big.NewInt(0), nil
	}

	target, _, err := big.ParseFloat(s, 10, prec, big.AwayFromZero)
	if err != nil {
		return nil, err
	}

	exp := new(big.Int)
	exp.Exp(ten, big.NewInt(decimal), nil)
	base := new(big.Float)
	base.SetInt(exp)

	target.Mul(target, base)
	result := new(big.Int)
	target.Int(result)

	return result, nil
}

// 11220000000000000000->"11.220000000"
func BigIntToStr(number *big.Int) string {
	if nil == number || 0 == number.Sign() {
		return zeroString
	}

	// 默认保留小数点9位
	return bigIntToStr(number, defaultDecimal)
}

func bigIntToStr(n *big.Int, precision int) string {
	if nil == n || precision < 0 {
		return zeroString
	}

	// 绝对值字符串
	var starter, first, last string
	// 负数
	if n.Sign() < 0 {
		starter = "-"
	}

	numCopied := new(big.Int).Set(n)
	number := numCopied.Abs(numCopied).String()
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

//11.22->11220000000
func Float64ToBigInt(number float64) *big.Int {
	base := new(big.Float)
	base.SetInt(big.NewInt(baseNumber))

	target := new(big.Float)
	target.SetPrec(prec)
	target.SetFloat64(number)
	target.Mul(target, base)

	result := new(big.Int)
	target.Int(result)

	return result
}

func Uint64ToBigInt(number uint64) *big.Int {
	base := big.NewInt(baseNumber)
	result := new(big.Int)
	result.SetUint64(number)
	result.Mul(result, base)

	return result
}

// FormatDecimalForERC20 火箭协议默认18位decimal，
// 而ERC20通常会自己定义decimal，需要转换
func FormatDecimalForERC20(number *big.Int, decimal int64) *big.Int {
	if nil == number || 0 == number.Sign() {
		return big.NewInt(0)
	}

	numberString := BigIntToStr(number)
	result, _ := strToBigInt(numberString, decimal)
	return result
}

// FormatDecimalForRocket 火箭协议默认18位decimal，
// 而ERC20通常会自己定义decimal，需要转换
func FormatDecimalForRocket(number *big.Int, decimal int64) *big.Int {
	if nil == number || 0 == number.Sign() {
		return big.NewInt(0)
	}

	numberString := bigIntToStr(number, int(decimal))
	result, _ := StrToBigInt(numberString)
	return result
}
