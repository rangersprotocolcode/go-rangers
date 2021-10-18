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

package service

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/db"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"sort"
	"testing"

	"crypto/sha256"
	"golang.org/x/crypto/sha3"
)

var (
	leveldb *db.LDBDatabase
	triedb  account.AccountDatabase
)

func TestRefundInfoList_AddRefundInfo(t *testing.T) {
	list := types.RefundInfoList{}

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

	var refundInfoList types.RefundInfoList
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

func TestRefundManager_Add(t *testing.T) {
	os.RemoveAll("logs")
	defer os.Remove("1.ini")
	defer os.RemoveAll("storage0")

	common.InitConf("1.ini")
	InitRefundManager(nil, nil)

	data := make(map[uint64]types.RefundInfoList, 2)
	refundInfoList := types.RefundInfoList{}
	refundInfoList.AddRefundInfo([]byte{0, 0, 0, 1}, big.NewInt(10090))
	refundInfoList.AddRefundInfo([]byte{0, 0, 2, 1}, big.NewInt(90090))
	data[10086] = refundInfoList

	db := getTestAccountDB()
	RefundManagerImpl.Add(data, db)

	data = make(map[uint64]types.RefundInfoList, 2)
	refundInfoList = types.RefundInfoList{}
	refundInfoList.AddRefundInfo([]byte{0, 0, 0, 1}, big.NewInt(20090))
	refundInfoList.AddRefundInfo([]byte{0, 0, 3, 1}, big.NewInt(190090))
	data[10086] = refundInfoList
	RefundManagerImpl.Add(data, db)
}

func TestRefundManager_CheckAndMove(t *testing.T) {
	os.RemoveAll("logs")
	defer os.Remove("1.ini")
	defer os.RemoveAll("storage0")

	common.InitConf("1.ini")
	InitRefundManager(nil, nil)

	addr := common.HexToAddress("0x4c1a42165e9009d747e4bcedc5654252b6bc9b8b")

	db := getTestAccountDB()
	if 0 != db.GetBalance(addr).Int64() {
		t.Fatalf("fail to get DB")
	}

	data := make(map[uint64]types.RefundInfoList, 2)
	refundInfoList := types.RefundInfoList{}
	refundInfoList.AddRefundInfo([]byte{0, 0, 0, 1}, big.NewInt(10090))
	refundInfoList.AddRefundInfo([]byte{0, 0, 2, 1}, big.NewInt(90090))
	refundInfoList.AddRefundInfo(addr.Bytes(), big.NewInt(290090))
	data[10086] = refundInfoList

	RefundManagerImpl.Add(data, db)

	RefundManagerImpl.CheckAndMove(10086, db)

	if 10090 != db.GetBalance(common.BytesToAddress([]byte{0, 0, 0, 1})).Int64() {
		t.Fatalf("fail to checkAnd move, 0001")
	}
	if 290090 != db.GetBalance(addr).Int64() {
		t.Fatalf("fail to checkAnd move, %s", addr.GetHexString())
	}
}

func getTestAccountDB() *account.AccountDB {
	if nil == leveldb {
		leveldb, _ = db.NewLDBDatabase("test", 0, 0)
		triedb = account.NewDatabase(leveldb)
	}

	accountdb, _ := account.NewAccountDB(common.Hash{}, triedb)
	return accountdb
}
