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
	"bytes"
	"fmt"
	"math/big"
	"sort"

	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/base"
	bn_curve "com.tuntun.rocket/node/src/consensus/groupsig/bn256"
)

type Signature struct {
	value bn_curve.G1
}

func DeserializeSign(b []byte) *Signature {
	sig := &Signature{}
	sig.Deserialize(b)
	return sig
}

func (sig Signature) Serialize() []byte {
	if sig.IsNil() {
		return []byte{}
	}
	return sig.value.Marshal()
}

func (sig *Signature) Deserialize(b []byte) error {
	if len(b) == 0 {
		return fmt.Errorf("signature Deserialized failed.")
	}
	sig.value.Unmarshal(b)
	return nil
}

func (sig Signature) GetHexString() string {
	return PREFIX + common.Bytes2Hex(sig.value.Marshal())
}

func (sig *Signature) SetHexString(s string) error {
	if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
		return fmt.Errorf("arg failed")
	}
	buf := s[len(PREFIX):]

	if sig.value.IsNil() {
		sig.value = bn_curve.G1{}
	}

	sig.value.Unmarshal(common.Hex2Bytes(buf))
	return nil
}

func (sig *Signature) IsNil() bool {
	return sig.value.IsNil()
}

func (sig Signature) IsEqual(rhs Signature) bool {
	return bytes.Equal(sig.value.Marshal(), rhs.value.Marshal())
}

func (sig Signature) IsValid() bool {
	s := sig.Serialize()
	if len(s) == 0 {
		return false
	}

	return sig.value.IsValid()
}

func Sign(sec Seckey, msg []byte) (sig Signature) {
	bg := hashToG1(string(msg))
	sig.value.ScalarMult(bg, sec.GetBigInt())
	return sig
}

func VerifySig(pub Pubkey, msg []byte, sig Signature) bool {
	if sig.IsNil() || !sig.IsValid() {
		return false
	}
	if !pub.IsValid() {
		return false
	}
	if sig.value.IsNil() {
		return false
	}
	bQ := bn_curve.GetG2Base()
	p1 := bn_curve.Pair(&sig.value, bQ)

	Hm := hashToG1(string(msg))
	p2 := bn_curve.Pair(Hm, &pub.value)

	return bn_curve.PairIsEuqal(p1, p2)
}

func RecoverGroupSignature(memberSignMap map[string]Signature, thresholdValue int) *Signature {
	if thresholdValue < len(memberSignMap) {
		memberSignMap = getRandomKSignInfo(memberSignMap, thresholdValue)
	}
	ids := make([]ID, thresholdValue)
	sigs := make([]Signature, thresholdValue)
	i := 0
	for s_id, si := range memberSignMap {
		var id ID
		id.SetHexString(s_id)
		ids[i] = id
		sigs[i] = si
		i++
		if i >= thresholdValue {
			break
		}
	}
	return recoverSignature(sigs, ids)
}

func (sig Signature) ShortS() string {
	str := sig.GetHexString()
	return common.ShortHex12(str)
}

func getRandomKSignInfo(memberSignMap map[string]Signature, k int) map[string]Signature {
	indexs := base.NewRand().RandomPerm(len(memberSignMap), k)
	sort.Ints(indexs)
	ret := make(map[string]Signature)

	i := 0
	j := 0
	for key, sign := range memberSignMap {
		if i == indexs[j] {
			ret[key] = sign
			j++
			if j >= k {
				break
			}
		}
		i++
	}
	return ret
}

func recoverSignature(sigs []Signature, ids []ID) *Signature {
	k := len(sigs)
	xs := make([]*big.Int, len(ids))
	for i := 0; i < len(xs); i++ {
		xs[i] = ids[i].GetBigInt()
	}
	// need len(ids) = k > 0
	sig := &Signature{}
	new_sig := &Signature{}
	for i := 0; i < k; i++ {
		var delta, num, den, diff *big.Int = big.NewInt(1), big.NewInt(1), big.NewInt(1), big.NewInt(0)
		for j := 0; j < k; j++ { //ID遍历
			if j != i { //不是自己
				num.Mul(num, xs[j])
				num.Mod(num, curveOrder)
				diff.Sub(xs[j], xs[i])
				den.Mul(den, diff)
				den.Mod(den, curveOrder)
			}
		}

		den.ModInverse(den, curveOrder)
		delta.Mul(num, den)
		delta.Mod(delta, curveOrder)

		new_sig.value.Set(&sigs[i].value)
		new_sig.mul(delta)

		if i == 0 {
			sig.value.Set(&new_sig.value)
		} else {
			sig.add(new_sig)
		}
	}
	return sig
}

func (sig *Signature) add(sig1 *Signature) error {
	new_sig := &Signature{}
	new_sig.value.Set(&sig.value)
	sig.value.Add(&new_sig.value, &sig1.value)

	return nil
}

func (sig *Signature) mul(bi *big.Int) error {
	g1 := new(bn_curve.G1)
	g1.Set(&sig.value)
	sig.value.ScalarMult(g1, bi)
	return nil
}
