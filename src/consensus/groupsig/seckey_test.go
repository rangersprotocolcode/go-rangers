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

package groupsig

import (
	"com.tuntun.rocket/node/src/consensus/base"
	"fmt"
	"math/big"
	"testing"
)

func TestSeckey(t *testing.T) {
	fmt.Printf("\nbegin test sec key...\n")
	t.Log("testSeckey")
	s := "401035055535747319451436327113007154621327258807739504261475863403006987855"
	var b = new(big.Int)
	b.SetString(s, 10)

	sec := NewSeckeyFromBigInt(b)
	str := sec.GetHexString()
	fmt.Printf("sec key export, len=%v, data=%v.\n", len(str), str)

	{
		var sec2 Seckey
		err := sec2.SetHexString(str)
		if err != nil || !sec.IsEqual(sec2) {
			t.Error("bad SetHexString")
		}
		str = sec2.GetHexString()
		fmt.Printf("sec key import and export again, len=%v, data=%v.\n", len(str), str)
	}

	{
		var sec2 Seckey
		err := sec2.Deserialize(sec.Serialize())
		if err != nil || !sec.IsEqual(sec2) {
			t.Error("bad Serialize")
		}
	}
	fmt.Printf("end test sec key.\n")
}

func TestShareSeckey(t *testing.T) {
	fmt.Printf("\nbegin testShareSeckey...\n")
	t.Log("testShareSeckey")
	n := 100
	msec := make([]Seckey, n)
	r := base.NewRand()
	for i := 0; i < n; i++ {
		msec[i] = *NewSeckeyFromRand(r.Deri(i))
	}
	id := *newIDFromInt64(123)
	s2 := ShareSeckey(msec, id)

	fmt.Printf("Share piece seckey:%s", s2.GetHexString())
	fmt.Printf("end testShareSeckey.\n")
}

func TestAggregateSeckeys(t *testing.T) {
	fmt.Printf("\nbegin test Aggregation...\n")
	t.Log("testAggregation")
	n := 100
	r := base.NewRand()
	seckeyContributions := make([]Seckey, n)
	for i := 0; i < n; i++ {
		seckeyContributions[i] = *NewSeckeyFromRand(r.Deri(i))
	}
	groupSeckey := AggregateSeckeys(seckeyContributions)
	groupPubkey := GeneratePubkey(*groupSeckey)
	t.Log("Group pubkey:", groupPubkey.GetHexString())
	fmt.Printf("end test Aggregation.\n")
}

func newIDFromInt64(i int64) *ID {
	return newIDFromBigInt(big.NewInt(i))
}
