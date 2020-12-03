package common

import (
	"fmt"
	"testing"
)

func TestGenerateLotteryAddress(t *testing.T) {
	addr := "0xca6c233c63deb259c919f3de5b6aecf59023d1ee38d95dabd348709aa9460329"
	for i := uint64(0); i < 100; i++ {
		fmt.Println(GenerateLotteryAddress(addr, i))
	}

}
