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
	"com.tuntun.rangers/node/src/middleware/db"
	"com.tuntun.rangers/node/src/middleware/types"
	"fmt"
	"testing"
)

func TestBlockChain_GenerateHeightKey(t *testing.T) {
	result := generateHeightKey(10)
	fmt.Println(len(result))
	fmt.Printf("%v", result)
}

func TestDB(t *testing.T) {
	db1, _ := db.NewDatabase(blockForkDBPrefix)
	db1.Put([]byte("1"), []byte("1"))
	db1.Put([]byte("2"), []byte("2"))

	db1.Delete([]byte("1"))

	iterator1 := db1.NewIterator()
	for iterator1.Next() {
		key := iterator1.Key()
		realKey := key[9:]
		fmt.Printf("key:%v,realkey:%v\n", key, realKey)
		db1.Delete(realKey)
		//db1.Delete(key)
	}
	//for iterator1.Next() {
	//	key := iterator1.Key()
	//	fmt.Printf("key2:%v\n", key)
	//	//append(keyList, key)
	//}
	//
	//db2, _ := db.NewDatabase(blockForkDBPrefix)
	//iterator2 := db2.NewIterator()
	//for iterator2.Next() {
	//	key := iterator2.Key()
	//	realKey := key[9:]
	//	fmt.Printf("second key:%v\n", realKey)
	//}
}

func TestCalStateRoot(t *testing.T) {
	receipt := types.Receipt{}
	receipt.PostState = common.CopyBytes(nil)
	receipt.Status = 1
	receipt.CumulativeGasUsed = 0
	receipt.Height = 43871249
	receipt.ContractAddress = common.Address{}
	receipt.Result = ""
	receipt.TxHash = common.HexToHash("0xe17882a7e7e7239573cd7b9da08541841418238933bb80d3e4fa52406353e4b3")

	logs := make([]*types.Log, 0)
	log0 := &types.Log{
		//BlockHash: common.HexToHash("0xfc75ed89451acb523616a724f8beff4816431cc0c287cbbabeebe9bfef2d69f9"),
		Address:     common.HexToAddress("0x616fc92ce6ea765b3cbd7a03dfd7e707fbb81851"),
		BlockNumber: 43871249,
		Index:       0,
		TxIndex:     0,
		Removed:     false,
		TxHash:      common.HexToHash("0xe17882a7e7e7239573cd7b9da08541841418238933bb80d3e4fa52406353e4b3"),
		Topics: []common.Hash{common.HexToHash("0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"),
			common.HexToHash("0x000000000000000000000000ad984f65b508740740b8aef909d589553b2bf3ef"),
			common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
			common.HexToHash("0x000200030000000000000000000000000000000000649fa63900000000000037")},
	}
	logs = append(logs, log0)

	log1 := &types.Log{
		//BlockHash: common.HexToHash("0xfc75ed89451acb523616a724f8beff4816431cc0c287cbbabeebe9bfef2d69f9"),
		Address:     common.HexToAddress("0x616fc92ce6ea765b3cbd7a03dfd7e707fbb81851"),
		BlockNumber: 43871249,
		Index:       1,
		TxIndex:     0,
		Removed:     false,
		TxHash:      common.HexToHash("0xe17882a7e7e7239573cd7b9da08541841418238933bb80d3e4fa52406353e4b3"),
		Topics: []common.Hash{common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
			common.HexToHash("0x000000000000000000000000ad984f65b508740740b8aef909d589553b2bf3ef"),
			common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
			common.HexToHash("0x000200030000000000000000000000000000000000649fa63900000000000037")},
	}
	logs = append(logs, log1)

	log2 := &types.Log{
		//BlockHash: common.HexToHash("0xfc75ed89451acb523616a724f8beff4816431cc0c287cbbabeebe9bfef2d69f9"),
		Address:     common.HexToAddress("0xc20481d12524a8b5c51300dd95a967ccd42c98c3"),
		BlockNumber: 43871249,
		Index:       2,
		TxIndex:     0,
		Removed:     false,
		TxHash:      common.HexToHash("0xe17882a7e7e7239573cd7b9da08541841418238933bb80d3e4fa52406353e4b3"),
		Topics: []common.Hash{common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
			common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
			common.HexToHash("0x000000000000000000000000ad984f65b508740740b8aef909d589553b2bf3ef"),
			common.HexToHash("0x000038010000000000000000000000000000000200649fa63d0000000002a516")},
	}
	logs = append(logs, log2)

	log3 := &types.Log{
		//BlockHash: common.HexToHash("0xfc75ed89451acb523616a724f8beff4816431cc0c287cbbabeebe9bfef2d69f9"),
		Address:     common.HexToAddress("0xc20481d12524a8b5c51300dd95a967ccd42c98c3"),
		BlockNumber: 43871249,
		Index:       3,
		TxIndex:     0,
		Removed:     false,
		TxHash:      common.HexToHash("0xe17882a7e7e7239573cd7b9da08541841418238933bb80d3e4fa52406353e4b3"),
		Topics: []common.Hash{common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
			common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
			common.HexToHash("0x000000000000000000000000ad984f65b508740740b8aef909d589553b2bf3ef"),
			common.HexToHash("0x00008e010000000000000000000000000000000200649fa63d0000000002a517")},
	}
	logs = append(logs, log3)

	log4 := &types.Log{
		//BlockHash: common.HexToHash("0xfc75ed89451acb523616a724f8beff4816431cc0c287cbbabeebe9bfef2d69f9"),
		Address:     common.HexToAddress("0xc20481d12524a8b5c51300dd95a967ccd42c98c3"),
		BlockNumber: 43871249,
		Index:       4,
		TxIndex:     0,
		Removed:     false,
		TxHash:      common.HexToHash("0xe17882a7e7e7239573cd7b9da08541841418238933bb80d3e4fa52406353e4b3"),
		Topics: []common.Hash{common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
			common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
			common.HexToHash("0x000000000000000000000000ad984f65b508740740b8aef909d589553b2bf3ef"),
			common.HexToHash("0x000089010000000000000000000000000000000200649fa63d0000000002a518")},
	}
	logs = append(logs, log4)

	receipt.Logs = logs

	tryPrintReceipt(receipt)
	var receipts types.Receipts
	receipts = append(receipts, &receipt)
	hash := calcReceiptsTree(receipts)
	fmt.Printf("hash:%s\n", hash.String())
}

func tryPrintReceipt(receipt types.Receipt) {
	fmt.Printf("tx[%s] receipt:%d,%d,%d,%s,%s\n", receipt.TxHash.String(), receipt.Status, receipt.CumulativeGasUsed, receipt.Height, receipt.ContractAddress, receipt.Result)
	fmt.Printf("logs:\n")
	for _, log := range receipt.Logs {
		fmt.Printf("tx hash:%s,block hash:%s,address:%s,block num:%d,log index:%d,tx index:%d,removed:%v,data:%s,topics:%v\n", log.TxHash.String(), log.BlockHash.String(), log.Address.String(), log.BlockNumber, log.Index, log.TxIndex, log.Removed, common.ToHex(log.Data), log.Topics)
	}
}

func TestBytes(t *testing.T) {
	b2 := []byte{0, 0, 56, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 100, 159, 166, 61, 0, 0, 0, 0, 2, 165, 22}
	s2 := common.Bytes2Hex(b2)
	fmt.Printf("%s\n", s2)

	b3 := []byte{0, 0, 142, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 100, 159, 166, 61, 0, 0, 0, 0, 2, 165, 23}
	s3 := common.Bytes2Hex(b3)
	fmt.Printf("%s\n", s3)

	b4 := []byte{0, 0, 137, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 100, 159, 166, 61, 0, 0, 0, 0, 2, 165, 24}
	s4 := common.Bytes2Hex(b4)
	fmt.Printf("%s\n", s4)

	b200 := []byte{0, 0, 121, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 100, 159, 166, 61, 0, 0, 0, 0, 2, 165, 22}
	s200 := common.Bytes2Hex(b200)
	fmt.Printf("%s\n", s200)
}
