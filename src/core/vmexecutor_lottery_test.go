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

var (
	sourceAddr  = common.HexStringToAddress("0xe7260a418579c2e6ca36db4fe0bf70f84d687bdf7ec6c0c181b43ee096a84aea")
	sourceAddr2 = common.HexStringToAddress("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443")
	targetAddr  = common.GenerateNFTSetAddress("test1")
	targetAddr2 = common.GenerateNFTSetAddress("0x7dba6865f337148e5887d6bea97e6a98701a2fa774bd00474ea68bcc645142f2")
)

const (
	NFTSETID1 = "nftSetId1"
	NFTID1    = "id1"
	APPID1    = "testApp1"

	NFTSETID2 = "nftSetId2"
	NFTID2    = "id2"
	APPID2    = "testApp2"

	NFTSETID3 = "nftSetId3"
	NFTID3    = "id3"
	APPID3    = "testApp3"

	NFTSETID4 = "nftSetId4"
	NFTID4    = "id4"
	APPID4    = "testApp4"

	NFTSETID5 = "nftSetId5"
	NFTID5    = "id5"
	APPID5    = "testApp5"

	NFTSETID6 = "nftSetId6"
	NFTID6    = "id6"
	APPID6    = "testApp6"
)

func testVMExecutorJackPot(t *testing.T) {
	// 定义奖池
	txString := `{"data":"{\"combo\":{\"p\":\"1\",\"content\":{\"3\":\"0.4\",\"2\":\"0.4\"}},\"prizes\":{\"nft\":{\"p\":\"0.4\",\"content\":{\"nftSetId1\":\"0.2\",\"nftSetId2\":\"0.8\"}},\"ft\":{\"p\":\"0.6\",\"content\":{\"ftSetId1\":{\"p\":\"0.2\",\"range\":\"0-1\"},\"ftSetId2\":{\"p\":\"0.8\",\"range\":\"1-100\"}}}}}","extraData":"","hash":"0xdf3d6b5ade4bd0de884bbf85000d9fa6f7ea91eeb99592f9cf9e825f428c2305","nonce":0,"sign":"0x275eb9cfdc5a850625cad2710fe16cb9d2383c2b9494069721db5660bae033a61ecbde189f03605ebc03916937a54e9942a01c0953cb3a833df86b2e8bdbeb8100","socketRequestId":"6044708977077341138-111","source":"0xe7260a418579c2e6ca36db4fe0bf70f84d687bdf7ec6c0c181b43ee096a84aea","target":"","time":"1585791972709","type":501}`
	var txJson types.TxJson
	err := json.Unmarshal([]byte(txString), &txJson)
	if nil != err {
		t.Fatalf(err.Error())
	}
	tx1 := txJson.ToTransaction()
	block := generateBlock()
	block.Transactions = append(block.Transactions, &tx1)

	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress(tx1.Source), big.NewInt(1000000000000))
	service.NFTManagerInstance.PublishNFTSet(&types.NFTSet{
		SetID:     NFTSETID1,
		MaxSupply: 0,
		Owner:     tx1.Source,
	}, accountDB)
	service.NFTManagerInstance.PublishNFTSet(&types.NFTSet{
		SetID:     NFTSETID2,
		MaxSupply: 0,
		Owner:     tx1.Source,
	}, accountDB)
	service.FTManagerInstance.PublishFTSet(&types.FTSet{
		AppId:     "test",
		Symbol:    "f1",
		ID:        "ftSetId1",
		MaxSupply: big.NewInt(0),
		Owner:     tx1.Source,
	}, accountDB)
	service.FTManagerInstance.PublishFTSet(&types.FTSet{
		AppId:     "test",
		Symbol:    "f2",
		ID:        "ftSetId2",
		MaxSupply: big.NewInt(0),
		Owner:     tx1.Source,
	}, accountDB)

	executor := newVMExecutor(accountDB, block, "testing")
	stateRoot, evictedTxs, transactions, receipts := executor.Execute()

	if 0 != strings.Compare("0d9cb8aa577928698fa109bd8e894a2cd8d1a435cd8d4e2b57e591014c606179", common.Bytes2Hex(stateRoot[:])) {
		t.Fatalf("fail to get stateRoot. %s", common.Bytes2Hex(stateRoot[:]))
	}
	if 0 != len(evictedTxs) {
		t.Fatalf("evictedTxs error")
	}
	if 1 != len(transactions) {
		t.Fatalf("txs error")
	}
	if 1 != len(receipts) {
		t.Fatalf("receipts error")
	}
	root, err := accountDB.Commit(true)
	if nil != err {
		t.Fatalf("fail to commit accountDB")
	}
	err = accountDB.Database().TrieDB().Commit(root, false)
	if nil != err {
		t.Fatalf("fail to commit TrieDB, %s", err.Error())
	}

	// testJackpot
	accountDB, _ = account.NewAccountDB(root, accountDB.Database())
	txString = `{"extraData":"","hash":"0xdf3d6b5ade4bd0de884bbf85000d9fa6f7ea91eeb99592f9cf9e825f428c2305","nonce":0,"sign":"0x275eb9cfdc5a850625cad2710fe16cb9d2383c2b9494069721db5660bae033a61ecbde189f03605ebc03916937a54e9942a01c0953cb3a833df86b2e8bdbeb8100","socketRequestId":"6044708977077341138-111","source":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","target":"f50ea6071d4c7e189f38b95a8277a9c20e2c821f5800e0eea58c518a70f97214","time":"1585791972709","type":502}`
	err = json.Unmarshal([]byte(txString), &txJson)
	if nil != err {
		t.Fatalf(err.Error())
	}
	tx2 := txJson.ToTransaction()
	tx2.RequestId = 10086
	accountDB.SetBalance(common.HexToAddress(tx2.Source), big.NewInt(1000000000000))

	block = generateBlock()
	block.Transactions = append(block.Transactions, &tx2)
	executor = newVMExecutor(accountDB, block, "testing")
	stateRoot, evictedTxs, transactions, receipts = executor.Execute()
	if 0 != strings.Compare("d01dab20330cd5e133785526582e2888063ddd2d5c78e8d8a7bce9c080845089", common.Bytes2Hex(stateRoot[:])) {
		t.Fatalf("fail to get stateRoot. %s", common.Bytes2Hex(stateRoot[:]))
	}
	if 0 != len(evictedTxs) {
		t.Fatalf("evictedTxs error")
	}
	if 1 != len(transactions) {
		t.Fatalf("txs error")
	}
	if 1 != len(receipts) {
		t.Fatalf("receipts error")
	}
	root, err = accountDB.Commit(true)
	if nil != err {
		t.Fatalf("fail to commit accountDB")
	}
	err = accountDB.Database().TrieDB().Commit(root, false)
	if nil != err {
		t.Fatalf("fail to commit TrieDB, %s", err.Error())
	}
}
