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

package groupsig

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus/base"
	bn_curve "com.tuntun.rangers/node/src/consensus/groupsig/bn256"
	"log"
	"math/big"
)

// Curve and Field order
var curveOrder = bn_curve.Order
var fieldOrder = bn_curve.P
var bitLength = curveOrder.BitLen()

// Seckey -- represented by a big.Int modulo curveOrder
type Seckey struct {
	value BnInt
}

func NewSeckeyFromRand(seed base.Rand) *Seckey {
	return newSeckeyFromByte(seed.Bytes())
}

func NewSeckeyFromBigInt(b *big.Int) *Seckey {
	nb := &big.Int{}
	nb.Set(b)
	b.Mod(nb, curveOrder)

	sec := new(Seckey)
	sec.value.setBigInt(b)

	return sec
}

func (sec Seckey) Serialize() []byte {
	return sec.value.serialize()
}

func (sec *Seckey) Deserialize(b []byte) error {
	return sec.value.deserialize(b)
}

func (sec Seckey) GetBigInt() (s *big.Int) {
	s = new(big.Int)
	s.Set(sec.value.getBigInt())
	return s
}

func (sec Seckey) GetHexString() string {
	return sec.value.getHexString()
}

func (sec *Seckey) SetHexString(s string) error {
	return sec.value.setHexString(s)
}

func (sec Seckey) IsEqual(rhs Seckey) bool {
	return sec.value.isEqual(&rhs.value)
}

func (sec Seckey) IsValid() bool {
	bi := sec.GetBigInt()
	return bi.Cmp(big.NewInt(0)) != 0
}

func (sec *Seckey) ShortS() string {
	str := sec.GetHexString()
	return common.ShortHex12(str)
}

func AggregateSeckeys(secs []Seckey) *Seckey {
	if len(secs) == 0 {
		log.Printf("AggregateSeckeys no secs")
		return nil
	}
	sec := new(Seckey)
	sec.value.setBigInt(secs[0].value.getBigInt())
	for i := 1; i < len(secs); i++ {
		sec.value.add(&secs[i].value)
	}

	x := new(big.Int)
	x.Set(sec.value.getBigInt())
	sec.value.setBigInt(x.Mod(x, curveOrder))
	return sec
}

func ShareSeckey(msec []Seckey, id ID) *Seckey {
	secret := big.NewInt(0)
	k := len(msec) - 1

	// evaluate polynomial f(x) with coefficients c0, ..., ck
	secret.Set(msec[k].GetBigInt())
	x := id.GetBigInt()
	new_b := &big.Int{}

	for j := k - 1; j >= 0; j-- {
		new_b.Set(secret)
		secret.Mul(new_b, x)

		new_b.Set(secret)
		secret.Add(new_b, msec[j].GetBigInt())

		new_b.Set(secret)
		secret.Mod(new_b, curveOrder)
	}

	return NewSeckeyFromBigInt(secret)
}

func newSeckeyFromByte(b []byte) *Seckey {
	sec := new(Seckey)
	err := sec.Deserialize(b[:32])
	if err != nil {
		log.Printf("NewSeckeyFromByte %s\n", err)
		return nil
	}

	sec.value.mod()
	return sec
}
