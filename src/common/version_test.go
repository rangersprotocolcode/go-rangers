package common

import (
	"com.tuntun.rocket/node/src/utility"
	"fmt"
	"testing"
)

func TestUint64(t *testing.T) {
	var a uint64 = utility.MaxUint64
	fmt.Println(a)
	fmt.Println(a > 1000000)
	a++
	fmt.Println(a)
}

func TestChainId(t *testing.T) {
	InitChainConfig(ENV_MAINNET)
	chainId1 := ChainId(1000000)
	fmt.Printf("chain id 1:%s\n", chainId1)

	chainId2 := ChainId(1000000 + 1)
	fmt.Printf("chain id 2:%s\n", chainId2)

	chainId3 := ChainId(1000000 - 1)
	fmt.Printf("chain id 3:%s\n", chainId3)
}
