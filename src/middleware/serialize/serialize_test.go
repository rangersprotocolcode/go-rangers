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

package serialize

import (
	"com.tuntun.rocket/node/src/common"
	"fmt"
	"math/big"
	"testing"
)

type Account struct {
	Nonce    uint64
	Balance  *big.Int
	Root     common.Hash
	CodeHash []byte
}

func TestSerialize(t *testing.T) {
	a := Account{Nonce: 100, Root: common.BytesToHash([]byte{1, 2, 3}), CodeHash: []byte{4, 5, 6},Balance:new(big.Int)}
	accountDump(a)
	byte, err := EncodeToBytes(a)
	if err != nil {
		fmt.Printf("encode error\n" + err.Error())
	}
	fmt.Println(byte)

	var b  = Account{}
	decodeErr := DecodeBytes(byte, &b)
	if decodeErr != nil {
		fmt.Printf("decode error\n" + decodeErr.Error())
	}
	accountDump(b)
}

func accountDump(a Account) {
	fmt.Printf("Account nounce:%d,Root:%s,CodeHash:%v,Balance:%v\n", a.Nonce, a.Root.String(), a.CodeHash,a.Balance.Sign())
}
