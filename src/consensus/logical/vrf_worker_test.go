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

package logical

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/model"
	"com.tuntun.rocket/node/src/consensus/vrf"
	cryptorand "crypto/rand"
	"fmt"
	"io"
	"math/big"
	"testing"
	"time"
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

func TestBigIntMod(t *testing.T) {
	seed := big.NewInt(761351)
	length := big.NewInt(3)
	index := seed.Mod(seed, length).Uint64()
	fmt.Println(index)
}

func TestProve(t *testing.T) {
	castTime, _ := time.Parse("2006-01-02 15:04:05 -0700 MST", "2023-03-21 14:26:06.280524597 +0800 CST")
	fmt.Println(castTime.String())
	beforeTime, _ := time.Parse("2006-01-02 15:04:05 -0700 MST", "2023-03-21 13:59:38.502854066 +0800 CST")
	fmt.Println(beforeTime.String())

	delta := CalDeltaByTime(castTime, beforeTime)
	random := common.FromHex("0x5c71b5c58e1484476ca895de7bb8f74e3e2c8a0d67d71fef53fef77cf678288d7b2e4ac4534e2cc5d65bee15c2351a70d1e39be17379cb200bced19b3a16686f")
	vrfMsg := genVrfMsg(random, delta)
	vrfPK := vrf.Hex2VRFPublicKey("0x009f3b76f3e49dcdd6d2ee8421f077fd4c68c176b18e1e602a3c1f09f9272250")
	vrfSK := vrf.Hex2VRFPrivateKey("0xcf11281bb181c0f44191e415555767ba9b66f6f97195b54405b82688e4bffc24009f3b76f3e49dcdd6d2ee8421f077fd4c68c176b18e1e602a3c1f09f9272250")
	prove, err := vrf.VRFGenProve(vrfPK, vrfSK, vrfMsg)
	if err != nil {
		panic(err)
	}
	fmt.Println(prove.Big())
	fmt.Println(vrf.VRFProof2Hash(vrf.VRFProve(prove.Big().Bytes())).Big())
}

func TestProve1(t *testing.T) {
	castTime, _ := time.Parse("2006-01-02 15:04:05 -0700 MST", "2023-03-21 13:59:38.502854066 +0800 CST")
	fmt.Println(castTime.String())
	beforeTime, _ := time.ParseInLocation("2006-01-02 15:04:05 -0700 MST", "2023-03-21 13:59:37.499894284 +0800 CST", time.Local)
	fmt.Println(beforeTime.String())

	delta := CalDeltaByTime(castTime, beforeTime)
	random := common.FromHex("0x5695bab150bdb457c8b81b41d38aa6d0b705172d668ddb02791dc9e467cee6a41d8a78ad35de62dd82e61b1fa47b3b14d02e9b2c22679c1fc29a1c9c04971506")
	vrfMsg := genVrfMsg(random, delta)
	vrfPK := vrf.Hex2VRFPublicKey("0x009f3b76f3e49dcdd6d2ee8421f077fd4c68c176b18e1e602a3c1f09f9272250")
	vrfSK := vrf.Hex2VRFPrivateKey("0xcf11281bb181c0f44191e415555767ba9b66f6f97195b54405b82688e4bffc24009f3b76f3e49dcdd6d2ee8421f077fd4c68c176b18e1e602a3c1f09f9272250")
	prove, err := vrf.VRFGenProve(vrfPK, vrfSK, vrfMsg)
	if err != nil {
		panic(err)
	}
	fmt.Println(prove.Big())
	fmt.Println(vrf.VRFProof2Hash(vrf.VRFProve(prove.Big().Bytes())).Big())
}
