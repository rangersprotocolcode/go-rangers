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
	"log"

	"com.tuntun.rocket/node/src/common"
	bn_curve "com.tuntun.rocket/node/src/consensus/groupsig/bn256"

	"golang.org/x/crypto/sha3"
)

type Pubkey struct {
	value bn_curve.G2
}

func GeneratePubkey(sec Seckey) *Pubkey {
	pub := new(Pubkey)
	pub.value.ScalarBaseMult(sec.value.getBigInt())
	return pub
}

func ByteToPublicKey(bytes []byte) Pubkey {
	var pk Pubkey
	if err := pk.Deserialize(bytes); err != nil {
		return Pubkey{}
	}
	return pk
}

func (pub Pubkey) Serialize() []byte {
	return pub.value.Marshal()
}

func (pub *Pubkey) Deserialize(b []byte) error {
	_, error := pub.value.Unmarshal(b)
	return error
}

func (pub Pubkey) GetHexString() string {
	return PREFIX + common.Bytes2Hex(pub.value.Marshal())
}

func (pub *Pubkey) SetHexString(s string) error {
	if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
		return fmt.Errorf("arg failed")
	}
	buf := s[len(PREFIX):]

	pub.value.Unmarshal(common.Hex2Bytes(buf))
	return nil
}

func (pub Pubkey) IsEmpty() bool {
	return pub.value.IsEmpty()
}

func (pub Pubkey) IsEqual(rhs Pubkey) bool {
	return bytes.Equal(pub.value.Marshal(), rhs.value.Marshal())
}

func (pub Pubkey) IsValid() bool {
	return !pub.IsEmpty()
}

func (pub Pubkey) GetAddress() common.Address {
	h := sha3.Sum256(pub.Serialize())
	return common.BytesToAddress(h[:])
}

func AggregatePubkeys(pubs []Pubkey) *Pubkey {
	if len(pubs) == 0 {
		log.Printf("AggregatePubkeys no pubs")
		return nil
	}

	pub := new(Pubkey)
	pub.value.Set(&pubs[0].value)

	for i := 1; i < len(pubs); i++ {
		pub.add(&pubs[i])
	}

	return pub
}

func (pub *Pubkey) add(rhs *Pubkey) error {
	pa := &bn_curve.G2{}
	pb := &bn_curve.G2{}

	pa.Set(&pub.value)
	pb.Set(&rhs.value)

	pub.value.Add(pa, pb)
	return nil
}

func (pub *Pubkey) ShortS() string {
	str := pub.GetHexString()
	return common.ShortHex12(str)
}

func (pub Pubkey) MarshalJSON() ([]byte, error) {
	str := "\"" + pub.GetHexString() + "\""
	return []byte(str), nil
}

func (pub *Pubkey) UnmarshalJSON(data []byte) error {
	str := string(data[:])
	if len(str) < 2 {
		return fmt.Errorf("data size less than min.")
	}
	str = str[1 : len(str)-1]
	return pub.SetHexString(str)
}
