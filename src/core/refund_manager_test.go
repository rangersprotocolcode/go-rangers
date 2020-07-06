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

package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/utility"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"testing"

	"crypto/sha256"
	"golang.org/x/crypto/sha3"
)

func TestRefundInfoList_AddRefundInfo(t *testing.T) {
	list := RefundInfoList{}

	list.AddRefundInfo(utility.UInt64ToByte(100), big.NewInt(2000))
	fmt.Println(string(list.TOJSON()))
	list.AddRefundInfo(utility.UInt64ToByte(100), big.NewInt(2000))
	fmt.Println(string(list.TOJSON()))
	list.AddRefundInfo(utility.UInt64ToByte(100), big.NewInt(2000))
	fmt.Println(string(list.TOJSON()))

	list.AddRefundInfo(utility.UInt64ToByte(200), big.NewInt(9999))
	fmt.Println(string(list.TOJSON()))

}

func TestRefundInfoList_TOJSON(t *testing.T) {
	str := `{"List":[{"Value":6000,"Id":"AAAAAAAAAGQ="},{"Value":9999,"Id":"AAAAAAAAAMg="}]}`

	var refundInfoList RefundInfoList
	err := json.Unmarshal([]byte(str), &refundInfoList)
	if err != nil {
		fmt.Println(err.Error())
	}

	for i, refundInfo := range refundInfoList.List {
		fmt.Printf("%d: value: %d, id: %s\n", i, refundInfo.Value, common.ToHex(refundInfo.Id))
	}
}

func TestDismissHeightList_Len(t *testing.T) {
	dismissHeightList := DismissHeightList{}
	dismissHeightList = append(dismissHeightList, 1000)
	dismissHeightList = append(dismissHeightList, 200)
	dismissHeightList = append(dismissHeightList, 2000)

	fmt.Println(dismissHeightList)

	sort.Sort(dismissHeightList)
	fmt.Println(dismissHeightList)
	fmt.Println(dismissHeightList[0])

	addr_buf := sha3.Sum256([]byte("12345"))
	fmt.Println(addr_buf)
	addr_buf = sha256.Sum256([]byte("12345"))
	fmt.Println(addr_buf)
}
