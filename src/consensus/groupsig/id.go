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
	"fmt"
	"golang.org/x/crypto/sha3"
	"log"
	"math/big"
)

const ID_LENGTH = 32

// ID -- id for secret sharing, represented by big.Int
type ID struct {
	value BnInt
}

func NewIDFromPubkey(pk Pubkey) *ID {
	h := sha3.Sum256(pk.Serialize()) //取得公钥的SHA3 256位哈希
	bi := new(big.Int).SetBytes(h[:])
	return newIDFromBigInt(bi)
}

func DeserializeID(bs []byte) ID {
	var id ID
	if err := id.Deserialize(bs); err != nil {
		return ID{}
	}
	return id
}

func (id ID) GetHexString() string {
	bs := id.Serialize()
	return common.ToHex(bs)
}

func (id *ID) SetHexString(s string) error {
	return id.value.setHexString(s)
}

func (id ID) GetBigInt() *big.Int {
	x := new(big.Int)
	x.Set(id.value.getBigInt())
	return x
}

func (id *ID) SetBigInt(b *big.Int) error {
	id.value.setBigInt(b)
	return nil
}

func (id ID) Serialize() []byte {
	idBytes := id.value.serialize()
	if len(idBytes) == ID_LENGTH {
		return idBytes
	}
	if len(idBytes) > ID_LENGTH {
		panic("ID Serialize error: ID bytes is more than IDLENGTH")
	}
	buff := make([]byte, ID_LENGTH)
	copy(buff[ID_LENGTH-len(idBytes):ID_LENGTH], idBytes)
	return buff
}

func (id *ID) Deserialize(b []byte) error {
	return id.value.deserialize(b)
}

func (id ID) IsEqual(rhs ID) bool {
	return id.value.isEqual(&rhs.value)
}

func (id ID) IsValid() bool {
	bi := id.GetBigInt()
	return bi.Cmp(big.NewInt(0)) != 0

}

func (id ID) ShortS() string {
	str := id.GetHexString()
	return common.ShortHex12(str)
}

func (id ID) ToAddress() common.Address {
	return common.BytesToAddress(id.Serialize())
}

func newIDFromBigInt(b *big.Int) *ID {
	id := new(ID)
	err := id.value.setBigInt(b) //bn_curve C库函数
	if err != nil {
		log.Printf("NewIDFromBigInt %s\n", err)
		return nil
	}
	return id
}
func (id ID) MarshalJSON() ([]byte, error) {
	str := "\"" + id.GetHexString() + "\""
	return []byte(str), nil
}

func (id *ID) UnmarshalJSON(data []byte) error {
	str := string(data[:])
	if len(str) < 2 {
		return fmt.Errorf("data size less than min.")
	}
	str = str[1 : len(str)-1]
	return id.SetHexString(str)
}
