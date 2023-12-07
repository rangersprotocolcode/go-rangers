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

package core

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/middleware/types"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
)

func TestBigInt(t *testing.T) {
	bigInt := new(big.Int).SetUint64(111)
	str := bigInt.String()
	fmt.Printf("big int str:%s\n", str)

	bi, _ := new(big.Int).SetString(str, 10)
	bi = bi.Add(bi, new(big.Int).SetUint64(222))
	u := bi.Uint64()
	fmt.Printf("big int uint64:%d\n", u)
}

func TestStrToJson(t *testing.T) {
	a := []string{"1111", "222", "333"}
	b, err := json.Marshal(a)
	if err != nil {
		fmt.Printf("Json marshal []string err:%s", err.Error())
		return
	}
	str := string(b)
	fmt.Println(str)

	var c []string
	err = json.Unmarshal(b, &c)
	if err != nil {
		fmt.Printf("Json Unmarshal []string err:%s", err.Error())
	}
	fmt.Printf("C:%v\n", c)
}

func TestStrToJson1(t *testing.T) {
	//a := []string{"1111"}
	//b, err := json.Marshal(a)
	//if err != nil {
	//	fmt.Printf("Json marshal []string err:%s", err.Error())
	//	return
	//}
	//str := string(b)
	//fmt.Println(str)

	b := "[\"yyy\"]"
	var c []string
	err := json.Unmarshal([]byte(b), &c)
	if err != nil {
		fmt.Printf("Json Unmarshal []string err:%s", err.Error())
	}
	fmt.Printf("C:%v\n", c)
}

func TestResponse(t *testing.T) {
	res := response{
		Data:   "{\"status\":0,\"payload\":\"{\"code\":0,\"data\":{},\"status\":0}\"}",
		Id:     "hash",
		Status: "0",
	}

	data, err := json.Marshal(res)
	if err != nil {
		fmt.Errorf(err.Error())
	}
	fmt.Printf("data:%s", data)
}

func TestAddr(t *testing.T) {
	s := "TAD5ZbvETHrNobKa41hGkCkB37jEXCEQss"
	addr := common.HexToAddress(s)
	fmt.Printf("Addr:%v", addr)
}

func TestJSONTransferData(t *testing.T) {
	s := "{\"address1\":{\"balance\":\"127\",\"ft\":{\"name1\":\"189\",\"name2\":\"1\"},\"nft\":[\"id1\",\"sword2\"]}, \"address2\":{\"balance\":\"1\"}}"
	mm := make(map[string]types.TransferData, 0)
	if err := json.Unmarshal([]byte(s), &mm); nil != err {
		fmt.Errorf("fail to unmarshal")
	}

	fmt.Printf("length: %d\n", len(mm))
	fmt.Printf("length: %s", mm)
}

func TestJSONNFTID(t *testing.T) {
	a := []types.NFTID{}
	id1 := types.NFTID{Id: "1", SetId: "s1"}
	//id2 := types.NFTID{Id: "2", SetId: "s2"}
	a = append(a, id1)
	//a = append(a, id2)

	transferData := types.TransferData{NFT: a}
	mm := make(map[string]types.TransferData, 0)
	mm["address1"] = transferData
	data, _ := json.Marshal(mm)
	fmt.Printf("data:%s\n", data)

	m2 := make(map[string]types.TransferData, 0)
	err := json.Unmarshal([]byte(data), &m2)
	if err != nil {
		fmt.Printf("Unmarshal error:%s\n", err.Error())
	}
	fmt.Printf("m2:%v\n", m2)
}
