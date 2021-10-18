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

package logical

import (
	"com.tuntun.rocket/node/src/consensus/model"
	"com.tuntun.rocket/node/src/consensus/vrf"
	cryptorand "crypto/rand"
	"fmt"
	"io"
	"math/big"
	"testing"
)

func TestCalcPotentialProposal(t *testing.T) {
	param := model.ConsensusParam{
		PotentialProposalIndex: 30,
		PotentialProposal:      3,
		PotentialProposalMax:   8,
	}

	fmt.Println(calcPotentialProposal(20, param))  // 6
	fmt.Println(calcPotentialProposal(7, param))   // 3
	fmt.Println(calcPotentialProposal(200, param)) // 8
}

func TestBigInt(t *testing.T) {
	t.Log(max256, max256.String(), max256.FloatString(10))
}

func TestBigIntDiv(t *testing.T) {
	a, _ := new(big.Int).SetString("ffffffffffff", 16)
	b := new(big.Rat).SetInt(a)
	v := new(big.Rat).Quo(b, max256)
	t.Log(a, b, max256, v)
	t.Log(v.FloatString(5))

	a1 := new(big.Rat).SetInt64(10)
	a2 := new(big.Rat).SetInt64(30)
	v2 := a1.Quo(a1, a2)
	t.Log(v2.Float64())
	t.Log(v2.FloatString(5))
}

func TestCMP(t *testing.T) {
	rat := new(big.Rat).SetInt64(1)

	i := 1
	for i < 1000 {
		i++
		v := new(big.Rat).SetFloat64(1.66666666666666666666667)
		if v.Cmp(rat) > 0 {
			v = rat
		}
		t.Log(v.Quo(v, new(big.Rat).SetFloat64(0.5)), rat)
	}
}

func TestMax256(t *testing.T) {
	t1 := new(big.Int)
	t1.SetString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)
	m := new(big.Rat).SetInt(t1)
	f, _ := m.Float64()
	fmt.Printf("%v,%v", m.String(), f)
}

func TestVrfValueRatio(t *testing.T) {
	b := make([]byte, 80)
	io.ReadFull(cryptorand.Reader, b[:])
	fmt.Printf("%v\n", b)

	rat := vrfValueRatio(vrf.VRFProve(b))
	f, e := rat.Float64()
	fmt.Printf("%v,%v\n", f, e)
}

func TestBigIntBytes(t *testing.T) {
	a := []byte{0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	bigA := big.NewInt(0).SetBytes(a)
	b := bigA.Bytes()
	fmt.Printf("%v\n", b)
}
