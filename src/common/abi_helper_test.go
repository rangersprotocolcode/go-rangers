package common

import (
	"fmt"
	"testing"
)

func TestGenerateCallDataAddress(t *testing.T) {
	addr := HexToAddress("0x1111111111111111111111111111111111111111")
	fmt.Println(GenerateCallDataAddress(addr))
}
