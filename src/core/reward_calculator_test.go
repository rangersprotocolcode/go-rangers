package core

import (
	"testing"
	"x/src/common"
	"fmt"
	"x/src/utility"
	"math/big"
)

func TestGetYear(t *testing.T) {
	year := getYear(1)
	if 0 != year {
		t.Fatalf("year error for 1")
	}

	year = getYear(100000)
	if 0 != year {
		t.Fatalf("year error for 100000")
	}

	year = getYear(common.BlocksPerYear)
	if 1 != year {
		t.Fatalf("year error for BlocksPerYear")
	}

	year = getYear(common.BlocksPerYear + 1)
	if 1 != year {
		t.Fatalf("year error for BlocksPerYear+1")
	}
}

func TestGetTotalReward(t *testing.T) {
	reward := getTotalReward(1)
	if 15.9 != reward {
		t.Fatalf("reward error for 1")
	}
	reward = getTotalReward(1000000)
	if 15.9 != reward {
		t.Fatalf("reward error for 1000000")
	}
	reward = getTotalReward(common.BlocksPerYear)
	if 16.695 != reward {
		t.Fatalf("reward error for BlocksPerYear, %v", reward)
	}
	reward = getTotalReward(common.BlocksPerYear + 1)
	if 16.695 != reward {
		t.Fatalf("reward error for BlocksPerYear+1")
	}
	reward = getTotalReward(common.BlocksPerYear * 2)
	if 17.52975 != reward {
		t.Fatalf("reward error for BlocksPerYear*2, %v", reward)
	}

}

func TestFloat64Stake(t *testing.T) {
	total := uint64(1111111111)
	stake := uint64(99)
	money := float64(stake) / float64(total)
	prop := utility.Float64ToBigInt(money)
	fmt.Println(money)
	fmt.Println(prop)
}

func TestAddReward(t *testing.T) {
	all := make(map[common.Address]*big.Int)
	addr := common.CommunityAddress
	delta := big.NewInt(1000)
	fmt.Println(all[addr])

	addReward(all, addr, delta)
	fmt.Println(all[addr])

	addReward(all, addr, delta)
	fmt.Println(all[addr])

}

func TestProposerReward(t *testing.T) {
	stake := uint64(190000)
	fmt.Println(int(stake / common.ValidatorStake))
}
