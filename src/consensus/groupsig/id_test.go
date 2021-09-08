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

package groupsig

import (
	"com.tuntun.rocket/node/src/common"
	"fmt"
	"testing"
)

//
//import (
//	"testing"
//	"fmt"
//	"math/big"
//)
//
////测试从big.Int生成ID，以及ID的序列化
//func TestID(t *testing.T) {
//	t.Log("testString")
//	fmt.Printf("\nbegin test ID...\n")
//	b := new(big.Int)
//	b.SetString("001234567890abcdef", 16)
//	c := new(big.Int)
//	c.SetString("1234567890abcdef", 16)
//	idc := NewIDFromBigInt(c)
//	id1 := NewIDFromBigInt(b) //从big.Int生成ID
//	if id1.IsEqual(*idc) {
//		fmt.Println("id1 is equal to idc")
//	}
//	if id1 == nil {
//		t.Error("NewIDFromBigInt")
//	} else {
//		buf := id1.Serialize()
//		fmt.Printf("id Serialize, len=%v, data=%v.\n", len(buf), buf)
//	}
//
//	str := id1.GetHexString()
//	fmt.Printf("ID export, len=%v, data=%v.\n", len(str), str)
//
//	str0 := id1.value.GetHexString()
//	fmt.Printf("str0 =%v\n", str0)
//
//	{
//		var id2 ID
//		err := id2.SetHexString(id1.GetHexString()) //测试ID的十六进制导出和导入功能
//		if err != nil || !id1.IsEqual(id2) {
//			t.Errorf("not same\n%s\n%s", id1.GetHexString(), id2.GetHexString())
//		}
//	}
//
//	{
//		var id2 ID
//		err := id2.Deserialize(id1.Serialize()) //测试ID的序列化和反序列化
//		fmt.Printf("id2:%v", id2.GetHexString())
//		if err != nil || !id1.IsEqual(id2) {
//			t.Errorf("not same\n%s\n%s", id1.GetHexString(), id2.GetHexString())
//		}
//	}
//	fmt.Printf("end test ID.\n")
//}

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
