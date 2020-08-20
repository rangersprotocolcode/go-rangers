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
	"bytes"
	"encoding/json"
	"math/big"
)

type RefundInfo struct {
	Value *big.Int
	Id    []byte
}

type RefundInfoList struct {
	List []*RefundInfo
}

func (refundInfoList *RefundInfoList) AddRefundInfo(id []byte, value *big.Int) {
	found := false
	i := 0

	for ; i < len(refundInfoList.List); i++ {
		target := refundInfoList.List[i]
		if bytes.Compare(id, target.Id) == 0 {
			found = true
			break
		}
	}

	if found {
		target := refundInfoList.List[i]
		target.Value.Add(target.Value, value)
	} else {
		nb := &big.Int{}
		nb.SetBytes(value.Bytes())
		refundInfo := &RefundInfo{Value: nb, Id: id}
		refundInfoList.List = append(refundInfoList.List, refundInfo)
	}
}

func (refundInfoList *RefundInfoList) TOJSON() []byte {
	data, _ := json.Marshal(refundInfoList)
	return data
}

func (refundInfoList *RefundInfoList) IsEmpty() bool {
	return 0 == len(refundInfoList.List)
}

func GetRefundInfo(context map[string]interface{}) map[uint64]RefundInfoList {
	raw := context["refund"]
	return raw.(map[uint64]RefundInfoList)
}