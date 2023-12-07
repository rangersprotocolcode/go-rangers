// Copyright 2020 The RangersProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RangersProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RangersProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package common

import (
	"com.tuntun.rangers/node/src/utility"
	"fmt"
	"math/big"
	"strconv"
)

func GenerateCallDataAddress(addr Address) string {
	addrString := addr.String()[2:]
	padding := 64 - len(addrString)
	for i := 0; i < padding; i++ {
		addrString = "0" + addrString
	}
	return addrString
}

func GenerateCallDataString(chainName string) string {
	length := GenerateCallDataUint(uint64(len(chainName)))

	data := Bytes2Hex([]byte(chainName))
	padding := 64 - len(data)%64
	for i := 0; i < padding; i++ {
		data += "0"
	}

	return fmt.Sprintf("%s%s", length, data)
}

func GenerateCallDataUint(data uint64) string {
	result := strconv.FormatUint(data, 16)
	padding := 64 - len(result)
	for i := 0; i < padding; i++ {
		result = "0" + result
	}

	return result
}

func GenerateCallDataBigInt(data *big.Int) string {
	result := utility.BigIntBase10toN(data, 16)
	padding := 64 - len(result)
	for i := 0; i < padding; i++ {
		result = "0" + result
	}

	return result
}
