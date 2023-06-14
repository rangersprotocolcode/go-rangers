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
	"testing"
)

func TestPubkey(t *testing.T) {
	fmt.Printf("\nbegin test pub key...\n")
	t.Log("testPubkey")
	r := base.NewRand()

	fmt.Printf("size of rand = %v\n.", len(r))
	sec := NewSeckeyFromRand(r.Deri(1))
	if sec == nil {
		t.Fatal("NewSeckeyFromRand")
	}

	pub := GeneratePubkey(*sec)
	if pub == nil {
		t.Log("NewPubkeyFromSeckey")
	}

	{
		var pub2 Pubkey
		err := pub2.SetHexString(pub.GetHexString())
		if err != nil || !pub.IsEqual(pub2) {
			t.Log("pub != pub2")
		}
	}
	{
		var pub2 Pubkey
		err := pub2.Deserialize(pub.Serialize())
		if err != nil || !pub.IsEqual(pub2) {
			t.Log("pub != pub2")
		}
	}
	fmt.Printf("\nend test pub key.\n")
}

func BenchmarkPubkeyFromSeckey(b *testing.B) {
	b.StopTimer()

	r := base.NewRand()

	//var sec Seckey
	for n := 0; n < b.N; n++ {
		//sec.SetByCSPRNG()
		sec := NewSeckeyFromRand(r.Deri(1))
		b.StartTimer()
		GeneratePubkey(*sec)
		b.StopTimer()
	}
}
