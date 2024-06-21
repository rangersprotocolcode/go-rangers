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
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package common

import (
	"fmt"
	"math/big"
	"testing"
)

func TestRewardBlocks(test *testing.T) {
	i := uint64(10)

	j := i + GetRewardBlocks()

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
