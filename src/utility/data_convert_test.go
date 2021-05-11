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
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"testing"
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
	//str := "1.23000"
	str := "4.281755743"
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

	fmt.Println(bigIntToStr(value, 0))
}

func TestStrToBigInt7(t *testing.T) {
	str := "1"
	value, err := StrToBigInt(str)
	if err != nil {
		panic(err)
	}

	fmt.Println(str)
	fmt.Println(value.String())

	fmt.Println(bigIntToStr(value, 9))
}

func TestStrToBigInt11(t *testing.T) {
	str := "1000000000000000000000"
	value, err := StrToBigInt(str)
	if err != nil {
		panic(err)
	}

	fmt.Println(str)
	fmt.Println(value.String())

	fmt.Println(BigIntToStr(value))
}

func TestStrToBigInt12(t *testing.T) {
	str := "999999999999999999.999999999999999999"
	fmt.Println(str)
	bigInt, _ := StrToBigInt(str)
	fmt.Println(bigInt.String())
	fmt.Println(BigIntToStr(bigInt))
}

func TestFloat64ToBigInt(t *testing.T) {
	number := float64(11.2222222222222222)
	result := Float64ToBigInt(number)
	fmt.Println(result)
	if 11222222222 != result.Uint64() {
		t.Fatalf("11222222222 error")
	}
}

func TestUint64ToBigInt(t *testing.T) {
	number := uint64(1000009)
	fmt.Println(Uint64ToBigInt(number))
}

func TestBigInttoStr1(t *testing.T) {
	num := &big.Int{}
	num.SetUint64(17311813916080901740)
	fmt.Println(num.String())
	fmt.Println(BigIntToStr(num))

	num1 := &big.Int{}
	num1.SetString("2408246081606430596384225821300900488360223075370881439480057452", 10)
	fmt.Println(num1.String())
	fmt.Println(num1.IsUint64())

	fmt.Println(num.Cmp(num1))

}

func TestExp(t *testing.T) {
	test := new(big.Int)
	test.Exp(big.NewInt(10), big.NewInt(18), nil)
	fmt.Println(test)

	base := new(big.Float)
	base.SetInt(test)
	fmt.Println(base)
}

func TestFormatDecimalForERC20(t *testing.T) {
	number := Uint64ToBigInt(100)
	fmt.Println(number)

	fmt.Println(FormatDecimalForERC20(number, 18))
	fmt.Println(FormatDecimalForERC20(number, 0))
	fmt.Println(FormatDecimalForERC20(number, 10))
}

func TestFormat(t *testing.T) {
	numberBytes := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 8, 69, 149, 22, 20, 1, 72, 74, 0, 0, 0}
	fmt.Println(len(numberBytes))

	number := new(big.Int).SetBytes(numberBytes)
	fmt.Println(number)
}
