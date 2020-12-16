package common

import (
	"fmt"
	"testing"
)

func TestGenerateLotteryAddress(t *testing.T) {
	addr := "0x98c2d62e3475207c6ddb424b8398e62630158a6c3fa9fadc05e46ef19c5c6919"
	for i := uint64(0); i < 100; i++ {
		fmt.Println(GenerateLotteryAddress(addr, i))
	}

}
