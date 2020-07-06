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

// 正常发ft
func testVMExecutorPublishFTSet(t *testing.T) {
	block := generateBlock()

	txString := `{"data":"{\"symbol\":\"testFTSetSymbol3\",\"name\":\"testFTSetName\",\"maxSupply\":\"100\"}","extraData":"","hash":"0x2343783c29a0451facff1d406e1abc9c61112f99a023bc94a689ce82b9617fef","nonce":0,"sign":"0x2324fd2181f0008ad6337d80dc1fcc1a2218d88bc691a013975e0b013620b64126da72acea29d8fcfad9545588f58dd514b39287346ee8f1ccdc9e4f8809ca4301","socketRequestId":"-8164044966151317681-100","source":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","target":"","time":"1585738759571","type":110}`
	var tx1 types.Transaction
	json.Unmarshal([]byte(txString), &tx1)

	block.Transactions = append(block.Transactions, &tx1)
	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress(tx1.Source), big.NewInt(1000000000000))

	executor := newVMExecutor(accountDB, block, "testing")
	stateRoot, evictedTxs, transactions, receipts := executor.Execute()

	if 0 != strings.Compare("7f6187c5b79e8b3dcc9722b005bb93ece090d440d00fa20a0569d8ffbd63e7dd", common.Bytes2Hex(stateRoot[:])) {
		t.Fatalf("fail to get stateRoot. %s", common.Bytes2Hex(stateRoot[:]))
	}
	if 0 != len(evictedTxs) {
		t.Fatalf("fail to get evictedTxs")
	}
	if 1 != len(transactions) {
		t.Fatalf("fail to get transactions")
	}
	if 1 != len(receipts) || 0 != strings.Compare(receipts[0].Msg, "0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443-testFTSetSymbol3") {
		t.Fatalf("fail to get receipts")
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
	ftSet := service.FTManagerInstance.GetFTSet("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443-testFTSetSymbol3", accountDB)
	if nil == ftSet || 0 != strings.Compare(ftSet.Owner, tx1.Source) || 0 != big.NewInt(100000000000).Cmp(ftSet.MaxSupply) {
		t.Fatalf("fail to get ftSet")
	}

	balance := accountDB.GetBalance(common.HexToAddress(tx1.Source))
	if nil == balance || 0 != balance.Cmp(big.NewInt(999999900000)) {
		t.Fatalf("fee error")
	}
}

// 不正常发ft
// 手续费不退
func testVMExecutorPublishFTSetError(t *testing.T) {
	block := generateBlock()

	txString := `{"data":"{\"symbol\":\"testFTSet-Symbol3\",\"name\":\"testFTSetName\",\"maxSupply\":\"100\"}","extraData":"","hash":"0x2343783c29a0451facff1d406e1abc9c61112f99a023bc94a689ce82b9617fef","nonce":0,"sign":"0x2324fd2181f0008ad6337d80dc1fcc1a2218d88bc691a013975e0b013620b64126da72acea29d8fcfad9545588f58dd514b39287346ee8f1ccdc9e4f8809ca4301","socketRequestId":"-8164044966151317681-100","source":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","target":"","time":"1585738759571","type":110}`
	var tx1 types.Transaction
	json.Unmarshal([]byte(txString), &tx1)

	block.Transactions = append(block.Transactions, &tx1)
	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress(tx1.Source), big.NewInt(1000000000000))

	executor := newVMExecutor(accountDB, block, "testing")
	stateRoot, evictedTxs, transactions, receipts := executor.Execute()

	if 0 != strings.Compare("e93ae5047d6c1a47fad6619d3f3c0ac44184ceb9141f0094a9ef229e1ed98b9b", common.Bytes2Hex(stateRoot[:])) {
		t.Fatalf("fail to get stateRoot. %s", common.Bytes2Hex(stateRoot[:]))
	}
	if 1 != len(evictedTxs) {
		t.Fatalf("fail to get evictedTxs")
	}
	if 1 != len(transactions) {
		t.Fatalf("fail to get transactions")
	}
	if 1 != len(receipts) || 0 != strings.Compare(receipts[0].Msg, "appId or symbol wrong") {
		t.Fatalf("fail to get receipts")
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
	ftSet := service.FTManagerInstance.GetFTSet("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443-testFTSet-Symbol3", accountDB)
	if nil != ftSet {
		t.Fatalf("fail to get ftSet")
	}

	balance := accountDB.GetBalance(common.HexToAddress(tx1.Source))
	if nil == balance || 0 != balance.Cmp(big.NewInt(999999900000)) {
		t.Fatalf("fee error, %s", balance)
	}
}

// 正常mintFT
func testVMExecutorMintFT(t *testing.T) {
	block := generateBlock()

	tx1String := `{"data":"{\"symbol\":\"testFTSetSymbol1\",\"name\":\"testFTSetName\",\"maxSupply\":\"100\"}","extraData":"","hash":"0x2343783c29a0451facff1d406e1abc9c61112f99a023bc94a689ce82b9617fef","nonce":0,"sign":"0x2324fd2181f0008ad6337d80dc1fcc1a2218d88bc691a013975e0b013620b64126da72acea29d8fcfad9545588f58dd514b39287346ee8f1ccdc9e4f8809ca4301","socketRequestId":"-8164044966151317681-100","source":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","target":"","time":"1585738759571","type":110}`
	tx2String := `{"data":"{\"ftId\":\"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443-testFTSetSymbol1\",\"supply\":\"3.15\"}","extraData":"","hash":"0x45d0cd7345d4f524331efb1d6850cff7a805f32999cf4eefad67070c4223fa5b","nonce":0,"sign":"0xa5443215bf12ecb5f178f6ef09059e5c194bd5e688efe66e08771b2eba8dc8fe548d0aa6118e900ac6d316004f705eee162894913ff9b4e24fb80d8c154d186101","socketRequestId":"-7284513562776747698-111","source":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","target":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf444","time":"1585793497359","type":116}`
	var tx1 types.Transaction
	json.Unmarshal([]byte(tx1String), &tx1)
	block.Transactions = append(block.Transactions, &tx1)

	var tx2 types.Transaction
	json.Unmarshal([]byte(tx2String), &tx2)
	block.Transactions = append(block.Transactions, &tx2)

	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress(tx1.Source), big.NewInt(1000000000000))

	executor := newVMExecutor(accountDB, block, "testing")
	stateRoot, evictedTxs, transactions, receipts := executor.Execute()

	if 0 != strings.Compare("c5abe5b5b8205197ad7d660d649e190efef82dda63f6a43c084b357a9151ff9a", common.Bytes2Hex(stateRoot[:])) {
		t.Fatalf("fail to get stateRoot. %s", common.Bytes2Hex(stateRoot[:]))
	}
	if 0 != len(evictedTxs) {
		t.Fatalf("fail to get evictedTxs")
	}
	if 2 != len(transactions) {
		t.Fatalf("fail to get transactions")
	}
	if 2 != len(receipts) ||
		0 != strings.Compare(receipts[0].Msg, "0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443-testFTSetSymbol1") ||
		0 != strings.Compare(receipts[1].Msg, "mintFT successful. ftId: 0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443-testFTSetSymbol1, supply: 3.15, target: 0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf444") {
		t.Fatalf("fail to get receipts")
	}

	root, err := accountDB.Commit(true)
	if nil != err {
		t.Fatalf("fail to commit accountDB")
	}
	err = accountDB.Database().TrieDB().Commit(root, false)
	if nil != err {
		t.Fatalf("fail to commit TrieDB, %s", err.Error())
	}

	// check
	accountDB, _ = account.NewAccountDB(root, accountDB.Database())
	ftSet := service.FTManagerInstance.GetFTSet("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443-testFTSetSymbol1", accountDB)
	if nil == ftSet || 0 != strings.Compare(ftSet.Owner, tx1.Source) ||
		0 != big.NewInt(100000000000).Cmp(ftSet.MaxSupply) || 0 != big.NewInt(3150000000).Cmp(ftSet.TotalSupply) {
		t.Fatalf("fail to get ftSet")
	}

	ft := accountDB.GetFT(common.HexToAddress("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf444"), "0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443-testFTSetSymbol1")
	if nil == ft || 0 != big.NewInt(3150000000).Cmp(ft) {
		t.Fatalf("fail to get ft. addr: 0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf444 ")
	}

	balance := accountDB.GetBalance(common.HexToAddress(tx1.Source))
	if nil == balance || 0 != balance.Cmp(big.NewInt(999999800000)) {
		t.Fatalf("fee error")
	}
}

// 不正常mintFT
// 手续费不退
func testVMExecutorMintFTError(t *testing.T) {
	block := generateBlock()

	txString := `{"data":"{\"symbol\":\"testFTSetSymbol1\",\"name\":\"testFTSetName\",\"maxSupply\":\"100\"}","extraData":"","hash":"0x2343783c29a0451facff1d406e1abc9c61112f99a023bc94a689ce82b9617fef","nonce":0,"sign":"0x2324fd2181f0008ad6337d80dc1fcc1a2218d88bc691a013975e0b013620b64126da72acea29d8fcfad9545588f58dd514b39287346ee8f1ccdc9e4f8809ca4301","socketRequestId":"-8164044966151317681-100","source":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","target":"","time":"1585738759571","type":110}`
	var tx1 types.Transaction
	json.Unmarshal([]byte(txString), &tx1)
	block.Transactions = append(block.Transactions, &tx1)

	// ftId 不对
	tx2String := `{"data":"{\"ftId\":\"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443-testFTSetSymbol3\",\"supply\":\"3.15\"}","extraData":"","hash":"0x45d0cd7345d4f524331efb1d6850cff7a805f32999cf4eefad67070c4223fa5b","nonce":0,"sign":"0xa5443215bf12ecb5f178f6ef09059e5c194bd5e688efe66e08771b2eba8dc8fe548d0aa6118e900ac6d316004f705eee162894913ff9b4e24fb80d8c154d186101","socketRequestId":"-7284513562776747698-111","source":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","target":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf444","time":"1585793497359","type":116}`
	var tx2 types.Transaction
	json.Unmarshal([]byte(tx2String), &tx2)
	block.Transactions = append(block.Transactions, &tx2)

	// supply 不对
	tx3String := `{"data":"{\"ftId\":\"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443-testFTSetSymbol1\",\"supply\":\"315\"}","extraData":"","hash":"0x45d0cd7345d4f524331efb1d6850cff7a805f32999cf4eefad67070c4223fa5b","nonce":0,"sign":"0xa5443215bf12ecb5f178f6ef09059e5c194bd5e688efe66e08771b2eba8dc8fe548d0aa6118e900ac6d316004f705eee162894913ff9b4e24fb80d8c154d186101","socketRequestId":"-7284513562776747698-111","source":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","target":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf444","time":"1585793497359","type":116}`
	var tx3 types.Transaction
	json.Unmarshal([]byte(tx3String), &tx3)
	block.Transactions = append(block.Transactions, &tx3)

	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress(tx1.Source), big.NewInt(1000000000000))

	executor := newVMExecutor(accountDB, block, "testing")
	stateRoot, evictedTxs, transactions, receipts := executor.Execute()

	if 0 != strings.Compare("36ac7f86e700c2747f4d954869bae610793af2874d82185ed623163c6ddffce1", common.Bytes2Hex(stateRoot[:])) {
		t.Fatalf("fail to get stateRoot. %s", common.Bytes2Hex(stateRoot[:]))
	}
	if 2 != len(evictedTxs) {
		t.Fatalf("fail to get evictedTxs")
	}
	if 3 != len(transactions) {
		t.Fatalf("fail to get transactions")
	}
	if 3 != len(receipts) ||
		0 != strings.Compare(receipts[0].Msg, "0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443-testFTSetSymbol1") ||
		0 != strings.Compare(receipts[1].Msg, "not enough FT") ||
		0 != strings.Compare(receipts[2].Msg, "not enough FT") {
		t.Fatalf("fail to get receipts")
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
	ftSet := service.FTManagerInstance.GetFTSet("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443-testFTSetSymbol1", accountDB)
	if nil == ftSet {
		t.Fatalf("fail to get ftSet")
	}

	ft := accountDB.GetFT(common.HexToAddress(tx2.Target), "0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443-testFTSetSymbol1")
	if nil != ft && 0 != big.NewInt(0).Cmp(ft) {
		t.Fatalf("fail to get ft. addr: 0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf444")
	}

	balance := accountDB.GetBalance(common.HexToAddress(tx1.Source))
	if nil == balance || 0 != balance.Cmp(big.NewInt(999999700000)) {
		t.Fatalf("fee error, %s", balance)
	}
}

// 2不正常mintFT+1正常mintFT
// 手续费不退
func testVMExecutorMintFTGoodAndEvil(t *testing.T) {
	block := generateBlock()

	txString := `{"data":"{\"symbol\":\"testFTSetSymbol1\",\"name\":\"testFTSetName\",\"maxSupply\":\"100\"}","extraData":"","hash":"0x2343783c29a0451facff1d406e1abc9c61112f99a023bc94a689ce82b9617fef","nonce":0,"sign":"0x2324fd2181f0008ad6337d80dc1fcc1a2218d88bc691a013975e0b013620b64126da72acea29d8fcfad9545588f58dd514b39287346ee8f1ccdc9e4f8809ca4301","socketRequestId":"-8164044966151317681-100","source":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","target":"","time":"1585738759571","type":110}`
	var tx1 types.Transaction
	json.Unmarshal([]byte(txString), &tx1)
	block.Transactions = append(block.Transactions, &tx1)

	// ftId 不对
	tx2String := `{"data":"{\"ftId\":\"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443-testFTSetSymbol3\",\"supply\":\"3.15\"}","extraData":"","hash":"0x45d0cd7345d4f524331efb1d6850cff7a805f32999cf4eefad67070c4223fa5b","nonce":0,"sign":"0xa5443215bf12ecb5f178f6ef09059e5c194bd5e688efe66e08771b2eba8dc8fe548d0aa6118e900ac6d316004f705eee162894913ff9b4e24fb80d8c154d186101","socketRequestId":"-7284513562776747698-111","source":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","target":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf444","time":"1585793497359","type":116}`
	var tx2 types.Transaction
	json.Unmarshal([]byte(tx2String), &tx2)
	block.Transactions = append(block.Transactions, &tx2)

	// supply 不对
	tx3String := `{"data":"{\"ftId\":\"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443-testFTSetSymbol1\",\"supply\":\"315\"}","extraData":"","hash":"0x45d0cd7345d4f524331efb1d6850cff7a805f32999cf4eefad67070c4223fa5b","nonce":0,"sign":"0xa5443215bf12ecb5f178f6ef09059e5c194bd5e688efe66e08771b2eba8dc8fe548d0aa6118e900ac6d316004f705eee162894913ff9b4e24fb80d8c154d186101","socketRequestId":"-7284513562776747698-111","source":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","target":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf444","time":"1585793497359","type":116}`
	var tx3 types.Transaction
	json.Unmarshal([]byte(tx3String), &tx3)
	block.Transactions = append(block.Transactions, &tx3)

	tx4String := `{"data":"{\"ftId\":\"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443-testFTSetSymbol1\",\"supply\":\"3.15\"}","extraData":"","hash":"0x45d0cd7345d4f524331efb1d6850cff7a805f32999cf4eefad67070c4223fa5b","nonce":0,"sign":"0xa5443215bf12ecb5f178f6ef09059e5c194bd5e688efe66e08771b2eba8dc8fe548d0aa6118e900ac6d316004f705eee162894913ff9b4e24fb80d8c154d186101","socketRequestId":"-7284513562776747698-111","source":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","target":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf444","time":"1585793497359","type":116}`
	var tx4 types.Transaction
	json.Unmarshal([]byte(tx4String), &tx4)
	block.Transactions = append(block.Transactions, &tx4)

	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress(tx1.Source), big.NewInt(1000000000000))

	executor := newVMExecutor(accountDB, block, "testing")
	stateRoot, evictedTxs, transactions, receipts := executor.Execute()

	if 0 != strings.Compare("4d52d3c7efa09a0b37f827e5de1ef6b000d6df4b7a71416d797d4ea923c6292f", common.Bytes2Hex(stateRoot[:])) {
		t.Fatalf("fail to get stateRoot. %s", common.Bytes2Hex(stateRoot[:]))
	}
	if 2 != len(evictedTxs) {
		t.Fatalf("fail to get evictedTxs")
	}
	if 4 != len(transactions) {
		t.Fatalf("fail to get transactions")
	}
	if 4 != len(receipts) ||
		0 != strings.Compare(receipts[0].Msg, "0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443-testFTSetSymbol1") ||
		0 != strings.Compare(receipts[1].Msg, "not enough FT") ||
		0 != strings.Compare(receipts[2].Msg, "not enough FT") {
		t.Fatalf("fail to get receipts")
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
	ftSet := service.FTManagerInstance.GetFTSet("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443-testFTSetSymbol1", accountDB)
	if nil == ftSet {
		t.Fatalf("fail to get ftSet")
	}

	ft := accountDB.GetFT(common.HexToAddress(tx2.Target), "0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443-testFTSetSymbol1")
	if nil == ft || 0 != big.NewInt(3150000000).Cmp(ft) {
		t.Fatalf("fail to get ft. addr: 0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf444 ")
	}

	balance := accountDB.GetBalance(common.HexToAddress(tx1.Source))
	if nil == balance || 0 != balance.Cmp(big.NewInt(999999600000)) {
		t.Fatalf("fee error, %s", balance)
	}
}

