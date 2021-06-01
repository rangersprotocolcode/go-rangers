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

package service

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/utility"
	"fmt"
	"math"
	"math/big"
	"testing"
)

func TestGetEpoch(t *testing.T) {
	f := big.NewInt(92)
	f.Exp(f, big.NewInt(20), nil)
	s := new(big.Int).Exp(big.NewInt(100), big.NewInt(20), nil)
	fmt.Println(f)
	fmt.Println(s.String())

	ff, _, _ := big.ParseFloat(f.String(), 10, 256, big.ToNearestEven)
	sf, _, _ := big.ParseFloat(s.String(), 10, 256, big.ToNearestEven)
	fmt.Println(ff.Text('f',256))
	fmt.Println(sf.Text('f',256))
	r := new(big.Float).Quo(ff, sf)
	fmt.Println(r.Text('f',256))

	year := getEpoch(1)
	if 0 != year {
		t.Fatalf("year error for 1")
	}

	year = getEpoch(100000)
	if 0 != year {
		t.Fatalf("year error for 100000")
	}

	year = getEpoch(common.BlocksPerEpoch)
	if 1 != year {
		t.Fatalf("year error for BlocksPerYear")
	}

	year = getEpoch(common.BlocksPerEpoch + 1)
	if 1 != year {
		t.Fatalf("year error for BlocksPerYear+1")
	}
}

func TestGetTotalReward(t *testing.T) {
	reward := getTotalReward(1)
	if 0.052932098765432097 != reward {
		t.Fatalf("reward error for 1")
	}
	reward = getTotalReward(1000000)
	if 0.052932098765432097 != reward {
		t.Fatalf("reward error for 1000000")
	}
	reward = getTotalReward(common.BlocksPerEpoch)
	if 0.04869753086419753 != reward {
		t.Fatalf("reward error for BlocksPerYear, %v", reward)
	}
	reward = getTotalReward(common.BlocksPerEpoch + 1)
	if 0.04869753086419753 != reward {
		t.Fatalf("reward error for BlocksPerYear+1")
	}
	reward = getTotalReward(common.BlocksPerEpoch * 2)
	if 0.044801728395061725 != reward {
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
	fmt.Println(math.Ceil(float64(stake) / float64(common.ValidatorStake)))
}

func TestRewardCalculator_NextRewardHeight(t *testing.T) {
	height := uint64(11)
	fmt.Println(height / common.RewardBlocks)
	next := math.Ceil(float64(height) / float64(common.RewardBlocks))
	fmt.Println(next)

	nextblock := uint64(0)
	nextblock = uint64(next) * common.RewardBlocks
	fmt.Println(nextblock)

}
