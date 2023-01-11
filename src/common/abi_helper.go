package common

import (
	"com.tuntun.rocket/node/src/utility"
	"fmt"
	"math/big"
	"strconv"
)

func GenerateCallDataString(chainName string) string {
	length := GenerateCallDataUint(uint64(len(chainName)))

	data := Bytes2Hex([]byte(chainName))
	padding := 64 - len(data)
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
