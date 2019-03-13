package base

import (
    "encoding/hex"
    "strconv"
)

//把unicode字符集转化为字符串
//如输入xi := `\u5bb6\u65cf`
func MapHexToBytes(x []string) [][]byte {
    y := make([][]byte, len(x))
    for k, xi := range x {
        // TODO handle errors
        y[k], _ = hex.DecodeString(xi)
    }
    return y
}

func MapStringToBytes(x []string) [][]byte {
    y := make([][]byte, len(x))
    for k, xi := range x {
        y[k] = []byte(xi)
    }
    return y
}

//整数数组转化为字符串数组
func MapItoa(x []int) []string {
    y := make([]string, len(x))
    for k, xi := range x {
        y[k] = strconv.Itoa(xi)
    }
    return y
}

func MapShortIDToInt(x []ShortID) []int {
    y := make([]int, len(x))
    for k, xi := range x {
        y[k] = int(xi)
    }
    return y
}
