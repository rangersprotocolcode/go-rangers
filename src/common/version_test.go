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
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

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
	initChainConfig(ENV_MAINNET)
	chainId1 := ChainId(1000000)
	fmt.Printf("chain id 1:%s\n", chainId1)

	chainId2 := ChainId(1000000 + 1)
	fmt.Printf("chain id 2:%s\n", chainId2)

	chainId3 := ChainId(1000000 - 1)
	fmt.Printf("chain id 3:%s\n", chainId3)
}

func TestGetGenesis(t *testing.T) {
	if nil == getGenesisConf("1.json") {
		t.Fatal()
	}
}
