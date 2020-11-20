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
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
	"encoding/json"
	"math/big"
	"strings"
	"testing"
)

// 正常发nftSet
func testVMExecutorPublishNFTSet(t *testing.T) {
	block := generateBlock()

	txString := `{"data":"{\"conditions\":{\"ft\":{\"0x10086-abc\":\"0.1\",\"0x10086-123\":\"0.4\"},\"coin\":{\"ETH.ETH \":\"1 \",\"ONT \":\"0.5 \"}},\"symbol\":\"testNFTSetSymbol\",\"createTime\":\"1585791972730\",\"name\":\"testNFTSetName\",\"setId\":\"1c9b03bf-5975-417f-a6fa-dc098ba8ff2e\",\"maxSupply\":100}","extraData":"","hash":"0xdf3d6b5ade4bd0de884bbf85000d9fa6f7ea91eeb99592f9cf9e825f428c2305","nonce":0,"sign":"0x275eb9cfdc5a850625cad2710fe16cb9d2383c2b9494069721db5660bae033a61ecbde189f03605ebc03916937a54e9942a01c0953cb3a833df86b2e8bdbeb8100","socketRequestId":"6044708977077341138-111","source":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","target":"","time":"1585791972709","type":111}`
	var txJson types.TxJson
	err := json.Unmarshal([]byte(txString), &txJson)
	if nil != err {
		t.Fatalf(err.Error())
	}
	tx1 := txJson.ToTransaction()

	block.Transactions = append(block.Transactions, &tx1)
	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress(tx1.Source), big.NewInt(1000000000000))

	executor := newVMExecutor(accountDB, block, "testing")
	stateRoot, evictedTxs, transactions, receipts := executor.Execute()

	if 0 != strings.Compare("98cc35916b602e5685fa405a9581d1b8940a6fd1713255743e4600c3b45546c1", common.Bytes2Hex(stateRoot[:])) {
		t.Fatalf("fail to get stateRoot. %s", common.Bytes2Hex(stateRoot[:]))
	}
	if 0 != len(evictedTxs) {
		t.Fatalf("fail to get evictedTxs")
	}
	if 1 != len(transactions) {
		t.Fatalf("fail to get transactions")
	}
	if 1 != len(receipts) || 0 != strings.Compare(receipts[0].Msg, "nft publish successful, setId: 1c9b03bf-5975-417f-a6fa-dc098ba8ff2e") {
		t.Fatalf("fail to get receipts. %s", receipts[0].Msg)
	}

	root, err := accountDB.Commit(true)
	if nil != err {
		t.Fatalf("fail to commit accountDB")
	}
	err = accountDB.Database().TrieDB().Commit(root, false)
	if nil != err {
		t.Fatalf("fail to commit TrieDB, %s", err.Error())
	}

	accountDB, _ = account.NewAccountDB(root, accountDB.Database())
	nftSet := service.NFTManagerInstance.GetNFTSet("1c9b03bf-5975-417f-a6fa-dc098ba8ff2e", accountDB)
	if nil == nftSet || 0 != strings.Compare(nftSet.Owner, tx1.Source) || 100 != nftSet.MaxSupply {
		t.Fatalf("fail to get nftSet")
	}

	balance := accountDB.GetBalance(common.HexToAddress(tx1.Source))
	if nil == balance || 0 != balance.Cmp(big.NewInt(999999900000)) {
		t.Fatalf("fee error")
	}
}

// 不正常发nftSet
// 手续费不退
func testVMExecutorPublishNFTSetError(t *testing.T) {
	block := generateBlock()

	txString := `{"data":"{\"symbol\":\"testNFTSetSymbol\",\"createTime\":\"1585791972730\",\"name\":\"testNFTSetName\",\"setId\":\"1c9b03bf-5975-417f-a6fa-dc098ba8ff2e\",\"maxSupply\":-100}","extraData":"","hash":"0xdf3d6b5ade4bd0de884bbf85000d9fa6f7ea91eeb99592f9cf9e825f428c2305","nonce":0,"sign":"0x275eb9cfdc5a850625cad2710fe16cb9d2383c2b9494069721db5660bae033a61ecbde189f03605ebc03916937a54e9942a01c0953cb3a833df86b2e8bdbeb8100","socketRequestId":"6044708977077341138-111","source":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","target":"","time":"1585791972709","type":111}`
	var tx1 types.Transaction
	json.Unmarshal([]byte(txString), &tx1)

	block.Transactions = append(block.Transactions, &tx1)
	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress(tx1.Source), big.NewInt(1000000000000))

	executor := newVMExecutor(accountDB, block, "testing")
	stateRoot, evictedTxs, transactions, receipts := executor.Execute()

	if 0 != strings.Compare("ae0e307c7e33b2f79522a061c4e392976fce33df40f62e4e658231282943a3b7", common.Bytes2Hex(stateRoot[:])) {
		t.Fatalf("fail to get stateRoot. %s", common.Bytes2Hex(stateRoot[:]))
	}
	if 1 != len(evictedTxs) {
		t.Fatalf("fail to get evictedTxs")
	}
	if 1 != len(transactions) {
		t.Fatalf("fail to get transactions")
	}
	if 1 != len(receipts) || 0 != strings.Compare(receipts[0].Msg, "Publish NFT Set Bad Format") {
		t.Fatalf("fail to get receipts, %s", receipts[0].Msg)
	}

	root, err := accountDB.Commit(true)
	if nil != err {
		t.Fatalf("fail to commit accountDB")
	}
	err = accountDB.Database().TrieDB().Commit(root, false)
	if nil != err {
		t.Fatalf("fail to commit TrieDB, %s", err.Error())
	}

	accountDB, _ = account.NewAccountDB(root, accountDB.Database())
	nftSet := service.NFTManagerInstance.GetNFTSet("1c9b03bf-5975-417f-a6fa-dc098ba8ff2e", accountDB)
	if nil != nftSet {
		t.Fatalf("fail to get nftSet")
	}

	balance := accountDB.GetBalance(common.HexToAddress(tx1.Source))
	if nil == balance || 0 != balance.Cmp(big.NewInt(999999900000)) {
		t.Fatalf("fee error, %s", balance)
	}
}

// 正常mintnft
// 一个nft
func testVMExecutorMintNFT(t *testing.T) {
	block := generateBlock()

	txString := `{"data":"{\"symbol\":\"testNFTSetSymbol\",\"createTime\":\"1585791972730\",\"name\":\"testNFTSetName\",\"setId\":\"1c9b03bf-5975-417f-a6fa-dc098ba8ff2e\",\"maxSupply\":100}","extraData":"","hash":"0xdf3d6b5ade4bd0de884bbf85000d9fa6f7ea91eeb99592f9cf9e825f428c2305","nonce":0,"sign":"0x275eb9cfdc5a850625cad2710fe16cb9d2383c2b9494069721db5660bae033a61ecbde189f03605ebc03916937a54e9942a01c0953cb3a833df86b2e8bdbeb8100","socketRequestId":"6044708977077341138-111","source":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","target":"","time":"1585791972709","type":111}`
	var tx1 types.Transaction
	json.Unmarshal([]byte(txString), &tx1)
	block.Transactions = append(block.Transactions, &tx1)

	tx2String := `{"data":"{\"data\":\"5.99\",\"createTime\":\"1556076659050692000\",\"setId\":\"1c9b03bf-5975-417f-a6fa-dc098ba8ff2e\",\"id\":\"123450\",\"target\":\"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443\"}","extraData":"","hash":"0xe923663d6bb41b47755febb3984e9154050b788f0bd9f6ca41b77c3a3d7ed863","nonce":0,"sign":"0x2c4f232f809273f84d686dc0b08690070cdea95982ccd7e8740240acb81b74625c48b9e4e7ed6574adc31d0d700fff79f692d16017f851d996d81a800b0bdc1901","socketRequestId":"-7139970467356776184-8107300104116841842-100","source":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","target":"","time":"1585796407753","type":117}`
	var tx2 types.Transaction
	json.Unmarshal([]byte(tx2String), &tx2)
	block.Transactions = append(block.Transactions, &tx2)

	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress(tx1.Source), big.NewInt(1000000000000))

	executor := newVMExecutor(accountDB, block, "testing")
	stateRoot, evictedTxs, transactions, receipts := executor.Execute()

	if 0 != strings.Compare("12834f4392a98fceeab8036655794099a7528d9c8f65802416bb8584b335eb16", common.Bytes2Hex(stateRoot[:])) {
		t.Fatalf("fail to get stateRoot. %s", common.Bytes2Hex(stateRoot[:]))
	}
	if 0 != len(evictedTxs) {
		t.Fatalf("fail to get evictedTxs")
	}
	if 2 != len(transactions) {
		t.Fatalf("fail to get transactions")
	}
	if 2 != len(receipts) ||
		0 != strings.Compare(receipts[0].Msg, "nft publish successful, setId: 1c9b03bf-5975-417f-a6fa-dc098ba8ff2e") ||
		0 != strings.Compare(receipts[1].Msg, "nft mint successful. setId: 1c9b03bf-5975-417f-a6fa-dc098ba8ff2e,id: 123450") {
		t.Fatalf("fail to get receipts. %s", receipts[0].Msg)
	}

	root, err := accountDB.Commit(true)
	if nil != err {
		t.Fatalf("fail to commit accountDB")
	}
	err = accountDB.Database().TrieDB().Commit(root, false)
	if nil != err {
		t.Fatalf("fail to commit TrieDB, %s", err.Error())
	}

	accountDB, _ = account.NewAccountDB(root, accountDB.Database())
	nftSet := service.NFTManagerInstance.GetNFTSet("1c9b03bf-5975-417f-a6fa-dc098ba8ff2e", accountDB)
	if nil == nftSet || 0 != strings.Compare(nftSet.Owner, tx1.Source) || 100 != nftSet.MaxSupply ||
		1 != nftSet.TotalSupply || 1 != len(nftSet.OccupiedID) ||
		0 != strings.Compare(nftSet.OccupiedID["123450"].GetHexString(), "0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443") {
		t.Fatalf("fail to get nftSet, %s, %s", nftSet.OccupiedID["123450"].GetHexString(), tx2.Target)
	}

	nft := accountDB.GetNFTById("1c9b03bf-5975-417f-a6fa-dc098ba8ff2e", "123450")
	if nil == nft || 0 != strings.Compare("5.99", nft.GetData("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443")) {
		t.Fatalf("fail to get nft")
	}
	balance := accountDB.GetBalance(common.HexToAddress(tx1.Source))
	if nil == balance || 0 != balance.Cmp(big.NewInt(999999800000)) {
		t.Fatalf("fee error")
	}
}

// 正常mintnft maxsupply=0
// 两个nft
func testVMExecutorMintNFTWithoutLimit(t *testing.T) {
	block := generateBlock()

	txString := `{"data":"{\"symbol\":\"testNFTSetSymbol\",\"createTime\":\"1585791972730\",\"name\":\"testNFTSetName\",\"setId\":\"1c9b03bf-5975-417f-a6fa-dc098ba8ff2e\",\"maxSupply\":0}","extraData":"","hash":"0xdf3d6b5ade4bd0de884bbf85000d9fa6f7ea91eeb99592f9cf9e825f428c2305","nonce":0,"sign":"0x275eb9cfdc5a850625cad2710fe16cb9d2383c2b9494069721db5660bae033a61ecbde189f03605ebc03916937a54e9942a01c0953cb3a833df86b2e8bdbeb8100","socketRequestId":"6044708977077341138-111","source":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","target":"","time":"1585791972709","type":111}`
	var tx1 types.Transaction
	json.Unmarshal([]byte(txString), &tx1)
	block.Transactions = append(block.Transactions, &tx1)

	tx2String := `{"data":"{\"data\":\"5.99\",\"createTime\":\"1556076659050692000\",\"setId\":\"1c9b03bf-5975-417f-a6fa-dc098ba8ff2e\",\"id\":\"123450\",\"target\":\"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443\"}","extraData":"","hash":"0xe923663d6bb41b47755febb3984e9154050b788f0bd9f6ca41b77c3a3d7ed863","nonce":0,"sign":"0x2c4f232f809273f84d686dc0b08690070cdea95982ccd7e8740240acb81b74625c48b9e4e7ed6574adc31d0d700fff79f692d16017f851d996d81a800b0bdc1901","socketRequestId":"-7139970467356776184-8107300104116841842-100","source":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","target":"","time":"1585796407753","type":117}`
	var tx2 types.Transaction
	json.Unmarshal([]byte(tx2String), &tx2)
	block.Transactions = append(block.Transactions, &tx2)

	tx3String := `{"data":"{\"data\":\"6.99\",\"createTime\":\"1556076659050692000\",\"setId\":\"1c9b03bf-5975-417f-a6fa-dc098ba8ff2e\",\"id\":\"123451\",\"target\":\"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443\"}","extraData":"","hash":"0xe923663d6bb41b47755febb3984e9154050b788f0bd9f6ca41b77c3a3d7ed863","nonce":0,"sign":"0x2c4f232f809273f84d686dc0b08690070cdea95982ccd7e8740240acb81b74625c48b9e4e7ed6574adc31d0d700fff79f692d16017f851d996d81a800b0bdc1901","socketRequestId":"-7139970467356776184-8107300104116841842-100","source":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","target":"","time":"1585796407753","type":117}`
	var tx3 types.Transaction
	json.Unmarshal([]byte(tx3String), &tx3)
	block.Transactions = append(block.Transactions, &tx3)

	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress(tx1.Source), big.NewInt(1000000000000))

	executor := newVMExecutor(accountDB, block, "testing")
	stateRoot, evictedTxs, transactions, receipts := executor.Execute()

	if 0 != strings.Compare("6b21a37dbd2d885bd0fb255c01343e74d384d2b1a4c2a365fa3c974395af958b", common.Bytes2Hex(stateRoot[:])) {
		t.Fatalf("fail to get stateRoot. %s", common.Bytes2Hex(stateRoot[:]))
	}
	if 0 != len(evictedTxs) {
		t.Fatalf("fail to get evictedTxs")
	}
	if 3 != len(transactions) {
		t.Fatalf("fail to get transactions")
	}
	if 3 != len(receipts) ||
		0 != strings.Compare(receipts[0].Msg, "nft publish successful, setId: 1c9b03bf-5975-417f-a6fa-dc098ba8ff2e") ||
		0 != strings.Compare(receipts[1].Msg, "nft mint successful. setId: 1c9b03bf-5975-417f-a6fa-dc098ba8ff2e,id: 123450") ||
		0 != strings.Compare(receipts[2].Msg, "nft mint successful. setId: 1c9b03bf-5975-417f-a6fa-dc098ba8ff2e,id: 123451") {
		t.Fatalf("fail to get receipts. %s", receipts[0].Msg)
	}

	root, err := accountDB.Commit(true)
	if nil != err {
		t.Fatalf("fail to commit accountDB")
	}
	err = accountDB.Database().TrieDB().Commit(root, false)
	if nil != err {
		t.Fatalf("fail to commit TrieDB, %s", err.Error())
	}

	accountDB, _ = account.NewAccountDB(root, accountDB.Database())
	nftSet := service.NFTManagerInstance.GetNFTSet("1c9b03bf-5975-417f-a6fa-dc098ba8ff2e", accountDB)
	if nil == nftSet || 0 != strings.Compare(nftSet.Owner, tx1.Source) || 0 != nftSet.MaxSupply ||
		2 != nftSet.TotalSupply || 2 != len(nftSet.OccupiedID) ||
		0 != strings.Compare(nftSet.OccupiedID["123450"].GetHexString(), "0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443") ||
		0 != strings.Compare(nftSet.OccupiedID["123451"].GetHexString(), "0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443") {
		t.Fatalf("fail to get nftSet, %s, %s", nftSet.OccupiedID["123450"].GetHexString(), tx2.Target)
	}

	nft := accountDB.GetNFTById("1c9b03bf-5975-417f-a6fa-dc098ba8ff2e", "123450")
	if nil == nft || 0 != strings.Compare("5.99", nft.GetData("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443")) {
		t.Fatalf("fail to get nft")
	}

	nft2 := accountDB.GetNFTById("1c9b03bf-5975-417f-a6fa-dc098ba8ff2e", "123451")
	if nil == nft2 || 0 != strings.Compare("6.99", nft2.GetData("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443")) {
		t.Fatalf("fail to get nft2")
	}

	balance := accountDB.GetBalance(common.HexToAddress(tx1.Source))
	if nil == balance || 0 != balance.Cmp(big.NewInt(999999700000)) {
		t.Fatalf("fee error")
	}
}

// 正常mintnft maxsupply=0
// publishnftSet 3次，成功2次
// mintNFT 4次，成功3次
func testVMExecutorMintNFTWithoutLimitGoodAndEvil(t *testing.T) {
	block := generateBlock()

	// publishnftSet
	txString := `{"data":"{\"symbol\":\"testNFTSetSymbol\",\"createTime\":\"1585791972730\",\"name\":\"testNFTSetName\",\"setId\":\"1c9b03bf-5975-417f-a6fa-dc098ba8ff2e\",\"maxSupply\":0}","extraData":"","hash":"0xdf3d6b5ade4bd0de884bbf85000d9fa6f7ea91eeb99592f9cf9e825f428c2305","nonce":0,"sign":"0x275eb9cfdc5a850625cad2710fe16cb9d2383c2b9494069721db5660bae033a61ecbde189f03605ebc03916937a54e9942a01c0953cb3a833df86b2e8bdbeb8100","socketRequestId":"6044708977077341138-111","source":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","target":"","time":"1585791972709","type":111}`
	var tx1 types.Transaction
	json.Unmarshal([]byte(txString), &tx1)
	block.Transactions = append(block.Transactions, &tx1)

	// mintNFT
	tx2String := `{"data":"{\"data\":\"5.99\",\"createTime\":\"1556076659050692000\",\"setId\":\"1c9b03bf-5975-417f-a6fa-dc098ba8ff2e\",\"id\":\"123450\",\"target\":\"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443\"}","extraData":"","hash":"0xe923663d6bb41b47755febb3984e9154050b788f0bd9f6ca41b77c3a3d7ed863","nonce":0,"sign":"0x2c4f232f809273f84d686dc0b08690070cdea95982ccd7e8740240acb81b74625c48b9e4e7ed6574adc31d0d700fff79f692d16017f851d996d81a800b0bdc1901","socketRequestId":"-7139970467356776184-8107300104116841842-100","source":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","target":"","time":"1585796407753","type":117}`
	var tx2 types.Transaction
	json.Unmarshal([]byte(tx2String), &tx2)
	block.Transactions = append(block.Transactions, &tx2)

	// mintNFT
	tx3String := `{"data":"{\"data\":\"6.99\",\"createTime\":\"1556076659050692000\",\"setId\":\"1c9b03bf-5975-417f-a6fa-dc098ba8ff2e\",\"id\":\"123451\",\"target\":\"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443\"}","extraData":"","hash":"0xe923663d6bb41b47755febb3984e9154050b788f0bd9f6ca41b77c3a3d7ed863","nonce":0,"sign":"0x2c4f232f809273f84d686dc0b08690070cdea95982ccd7e8740240acb81b74625c48b9e4e7ed6574adc31d0d700fff79f692d16017f851d996d81a800b0bdc1901","socketRequestId":"-7139970467356776184-8107300104116841842-100","source":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","target":"","time":"1585796407753","type":117}`
	var tx3 types.Transaction
	json.Unmarshal([]byte(tx3String), &tx3)
	block.Transactions = append(block.Transactions, &tx3)

	// 重复发nftSet, 失败
	tx4String := `{"data":"{\"symbol\":\"testNFTSetSymbol\",\"createTime\":\"1585791972730\",\"name\":\"testNFTSetName\",\"setId\":\"1c9b03bf-5975-417f-a6fa-dc098ba8ff2e\",\"maxSupply\":0}","extraData":"","hash":"0xdf3d6b5ade4bd0de884bbf85000d9fa6f7ea91eeb99592f9cf9e825f428c2305","nonce":0,"sign":"0x275eb9cfdc5a850625cad2710fe16cb9d2383c2b9494069721db5660bae033a61ecbde189f03605ebc03916937a54e9942a01c0953cb3a833df86b2e8bdbeb8100","socketRequestId":"6044708977077341138-111","source":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","target":"","time":"1585791972709","type":111}`
	var tx4 types.Transaction
	json.Unmarshal([]byte(tx4String), &tx4)
	block.Transactions = append(block.Transactions, &tx4)

	// 新nftSet，只能有一个nft
	tx5String := `{"data":"{\"symbol\":\"testNFTSetSymbol\",\"createTime\":\"1585791972730\",\"name\":\"testNFTSetName\",\"setId\":\"onlyOne\",\"maxSupply\":1}","extraData":"","hash":"0xdf3d6b5ade4bd0de884bbf85000d9fa6f7ea91eeb99592f9cf9e825f428c2305","nonce":0,"sign":"0x275eb9cfdc5a850625cad2710fe16cb9d2383c2b9494069721db5660bae033a61ecbde189f03605ebc03916937a54e9942a01c0953cb3a833df86b2e8bdbeb8100","socketRequestId":"6044708977077341138-111","source":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","target":"","time":"1585791972709","type":111}`
	var tx5 types.Transaction
	json.Unmarshal([]byte(tx5String), &tx5)
	block.Transactions = append(block.Transactions, &tx5)

	// mintNFT
	tx6String := `{"data":"{\"data\":\"a.99\",\"createTime\":\"1556076659050692000\",\"setId\":\"onlyOne\",\"id\":\"a00\",\"target\":\"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf444\"}","extraData":"","hash":"0xe923663d6bb41b47755febb3984e9154050b788f0bd9f6ca41b77c3a3d7ed863","nonce":0,"sign":"0x2c4f232f809273f84d686dc0b08690070cdea95982ccd7e8740240acb81b74625c48b9e4e7ed6574adc31d0d700fff79f692d16017f851d996d81a800b0bdc1901","socketRequestId":"-7139970467356776184-8107300104116841842-100","source":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","target":"","time":"1585796407753","type":117}`
	var tx6 types.Transaction
	json.Unmarshal([]byte(tx6String), &tx6)
	block.Transactions = append(block.Transactions, &tx6)

	// 失败的mintnft，超出上限
	tx7String := `{"data":"{\"data\":\"a.99\",\"createTime\":\"1556076659050692000\",\"setId\":\"onlyOne\",\"id\":\"a01\",\"target\":\"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf444\"}","extraData":"","hash":"0xe923663d6bb41b47755febb3984e9154050b788f0bd9f6ca41b77c3a3d7ed863","nonce":0,"sign":"0x2c4f232f809273f84d686dc0b08690070cdea95982ccd7e8740240acb81b74625c48b9e4e7ed6574adc31d0d700fff79f692d16017f851d996d81a800b0bdc1901","socketRequestId":"-7139970467356776184-8107300104116841842-100","source":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","target":"","time":"1585796407753","type":117}`
	var tx7 types.Transaction
	json.Unmarshal([]byte(tx7String), &tx7)
	block.Transactions = append(block.Transactions, &tx7)

	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress(tx1.Source), big.NewInt(1000000000000))

	executor := newVMExecutor(accountDB, block, "testing")
	stateRoot, evictedTxs, transactions, receipts := executor.Execute()

	if 0 != strings.Compare("a2f4a59767b5fefca5715a4b12437243d4c354cd6beca90ceb33c1ad2f1231cd", common.Bytes2Hex(stateRoot[:])) {
		t.Fatalf("fail to get stateRoot. %s", common.Bytes2Hex(stateRoot[:]))
	}
	if 2 != len(evictedTxs) {
		t.Fatalf("fail to get evictedTxs")
	}
	if 7 != len(transactions) {
		t.Fatalf("fail to get transactions")
	}
	if 7 != len(receipts) ||
		0 != strings.Compare(receipts[0].Msg, "nft publish successful, setId: 1c9b03bf-5975-417f-a6fa-dc098ba8ff2e") ||
		0 != strings.Compare(receipts[1].Msg, "nft mint successful. setId: 1c9b03bf-5975-417f-a6fa-dc098ba8ff2e,id: 123450") ||
		0 != strings.Compare(receipts[2].Msg, "nft mint successful. setId: 1c9b03bf-5975-417f-a6fa-dc098ba8ff2e,id: 123451") ||
		0 != strings.Compare(receipts[4].Msg, "nft publish successful, setId: onlyOne") {
		t.Fatalf("fail to get receipts. %s", receipts[0].Msg)
	}

	root, err := accountDB.Commit(true)
	if nil != err {
		t.Fatalf("fail to commit accountDB")
	}
	err = accountDB.Database().TrieDB().Commit(root, false)
	if nil != err {
		t.Fatalf("fail to commit TrieDB, %s", err.Error())
	}

	accountDB, _ = account.NewAccountDB(root, accountDB.Database())

	nftSet := service.NFTManagerInstance.GetNFTSet("1c9b03bf-5975-417f-a6fa-dc098ba8ff2e", accountDB)
	if nil == nftSet || 0 != strings.Compare(nftSet.Owner, tx1.Source) || 0 != nftSet.MaxSupply ||
		2 != nftSet.TotalSupply || 2 != len(nftSet.OccupiedID) ||
		0 != strings.Compare(nftSet.OccupiedID["123450"].GetHexString(), "0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443") ||
		0 != strings.Compare(nftSet.OccupiedID["123451"].GetHexString(), "0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443") {
		t.Fatalf("fail to get nftSet, %s, %s", nftSet.OccupiedID["123450"].GetHexString(), tx2.Target)
	}

	nftSet2 := service.NFTManagerInstance.GetNFTSet("onlyOne", accountDB)
	if nil == nftSet2 || 0 != strings.Compare(nftSet2.Owner, tx1.Source) || 1 != nftSet2.MaxSupply ||
		1 != nftSet2.TotalSupply || 1 != len(nftSet2.OccupiedID) ||
		0 != strings.Compare(nftSet2.OccupiedID["a00"].GetHexString(), "0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf444") {
		t.Fatalf("fail to get nftSet2, %s", nftSet2.OccupiedID["a00"].GetHexString())
	}

	nft := accountDB.GetNFTById("1c9b03bf-5975-417f-a6fa-dc098ba8ff2e", "123450")
	if nil == nft || 0 != strings.Compare("5.99", nft.GetData("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443")) {
		t.Fatalf("fail to get nft")
	}

	nft2 := accountDB.GetNFTById("1c9b03bf-5975-417f-a6fa-dc098ba8ff2e", "123451")
	if nil == nft2 || 0 != strings.Compare("6.99", nft2.GetData("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443")) {
		t.Fatalf("fail to get nft2")
	}

	nft3 := accountDB.GetNFTById("onlyOne", "a00")
	if nil == nft3 || 0 != strings.Compare("a.99", nft3.GetData("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443")) {
		t.Fatalf("fail to get nft3")
	}

	balance := accountDB.GetBalance(common.HexToAddress(tx1.Source))
	if nil == balance || 0 != balance.Cmp(big.NewInt(999999300000)) {
		t.Fatalf("fee error")
	}
}
