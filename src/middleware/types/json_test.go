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

package types

import (
	"com.tuntun.rocket/node/src/common"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
)

func TestJSONObject_Put(t *testing.T) {
	obj := NewJSONObject()
	obj.Put("1", "a")

	fmt.Println(obj.TOJSONString())
}

func TestJSONObject_Merge(t *testing.T) {
	obj := NewJSONObject()
	obj.Put("1", big.NewInt(10))
	fmt.Println(obj.TOJSONString())

	obj2 := NewJSONObject()
	obj2.Put("1", big.NewInt(100))
	fmt.Println(obj2.TOJSONString())

	obj.Merge(&obj2, ReplaceBigInt)
	fmt.Println(obj.TOJSONString())
	fmt.Println(obj2.TOJSONString())
}

func TestJSONObject_Put2(t *testing.T) {
	obj := NewJSONObject()
	obj.Put("1", big.NewInt(10))

	mobj := NewJSONObject()
	mobj.Put("ft", obj.TOJSONString())

	fmt.Println(mobj.TOJSONString())
	fmt.Println(mobj.Remove("ft"))
}

func TestJSONObject_Put3(t *testing.T) {
	obj := NewJSONObject()
	obj.Put("1", big.NewInt(10))

	mobj := NewJSONObject()
	mobj.Put("ft", obj)

	fmt.Println(mobj.TOJSONString())
	fmt.Println(mobj.Remove("ft"))
}

func TestJsonReceipt(t *testing.T) {
	a := Receipt{}
	aBytes, _ := json.Marshal(a)
	fmt.Printf("a json marchal:%s\n", aBytes)

	b := Receipt{}
	b.Logs = nil
	bBytes, _ := json.Marshal(b)
	fmt.Printf("b json marchal:%s\n", bBytes)

	c := Receipt{}
	c.Logs = make([]*Log, 0)
	cBytes, _ := json.Marshal(c)
	fmt.Printf("c json marchal:%s\n", cBytes)

	logMap := make(map[common.Hash][]*Log)
	d := Receipt{}
	d.Logs = logMap[common.Hash{}]
	dBytes, _ := json.Marshal(d)
	fmt.Printf("d json marchal:%s\n", dBytes)
}
