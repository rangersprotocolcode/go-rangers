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

package account

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/storage/rlp"
	"fmt"
	"testing"
)

func Test_RLP_account(t *testing.T) {
	account := Account{}

	data, err := rlp.EncodeToBytes(account)
	if err != nil {
		t.Fatalf("%s", err.Error())
	}

	fmt.Println(data)
	fmt.Println(common.ToHex(data))

	accountBeta := &Account{}
	err = rlp.DecodeBytes(data, accountBeta)

}
