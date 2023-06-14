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
	"com.tuntun.rocket/node/src/common"
	"fmt"
	"testing"
)

func TestID(t *testing.T) {
	for {
		sk := common.GenerateKey("")
		//secretSeed := base.RandFromBytes(sk.PrivKey.D.Bytes())
		//secKey := NewSeckeyFromRand(secretSeed)
		//pubKey := GeneratePubkey(*secKey)
		idBytes := sk.GetPubKey().GetID()
		fmt.Printf("id bytes:%v\n", idBytes)
		//big int deserialize
		id := DeserializeID(idBytes[:])

		//big int serialize
		idStr := id.GetHexString()
		fmt.Printf("id string:%s\n", idStr)
		fmt.Printf("send to id:%v\n", common.FromHex(idStr))
		fmt.Printf("set net id:%v\n", id.Serialize())

		if idBytes[0] == 0 {
			break
		}
	}
}
