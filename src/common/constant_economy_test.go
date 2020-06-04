package common

import (
	"fmt"
	"math/big"
	"testing"
)

func TestRewardBlocks(test *testing.T) {
	i := uint64(10)

	j := i + RewardBlocks

	fmt.Println(j)

	bigI := big.NewInt(10)
	bigJ := big.NewInt(100)

	bigI.Add(bigJ, bigI)
	fmt.Println(bigI)

	bigZ := new(big.Int).SetBytes(bigI.Bytes())
	bigZ.Sub(bigZ, bigJ)
	fmt.Println(bigI)
	fmt.Println(bigZ)

	stake := big.NewInt(100000 * 1000000000 * 2)
	used := big.NewInt(100000 * 1000000000)
	left := new(big.Int).SetBytes(stake.Bytes())
	left.Sub(left, used)

	fmt.Println(stake)
	fmt.Println(used)
}
