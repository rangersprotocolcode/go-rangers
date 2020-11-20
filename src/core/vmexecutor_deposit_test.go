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
	"com.tuntun.rocket/node/src/storage/account"
	"strings"
	"testing"
)

// 主链币充值
func testVMExecutorCoinDeposit(t *testing.T) {
	block := generateBlock()

	tx1 := types.Transaction{Source: "0x001", Type: types.TransactionTypeCoinDepositAck}
	tx1.Data = `{"chainType":"ETH.ETH","Amount":"12.56","addr":"0x12345abcde","txId":"0xaaaa"}`

	block.Transactions = append(block.Transactions, &tx1)
	accountDB := getTestAccountDB()
	executor := newVMExecutor(accountDB, block, "testing")
	stateRoot, evictedTxs, transactions, receipts := executor.Execute()

	if 0 != strings.Compare("66845646df71bfc03d43f69253ac393927c935bf42d662bd54c97f4dfb30b4ec", common.Bytes2Hex(stateRoot[:])) {
		t.Fatalf("fail to get stateRoot. %s", common.Bytes2Hex(stateRoot[:]))
	}
	if 0 != len(evictedTxs) {
		t.Fatalf("fail to get evictedTxs")
	}
	if 1 != len(transactions) {
		t.Fatalf("fail to get transactions")
	}
	if 1 != len(receipts) || 0 != strings.Compare(receipts[0].Msg, "coin: ETH.ETH, deposit 12560000000") {
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
	ft := accountDB.GetBNT(common.HexToAddress(tx1.Source), "ETH.ETH")
	if nil == ft || 0 != strings.Compare(ft.String(), "12560000000") {
		t.Fatalf("fail to get ft")
	}

	ftMap := accountDB.GetAllBNT(common.HexToAddress(tx1.Source))
	if nil == ftMap || 1 != len(ftMap) || 0 != strings.Compare(ftMap["ETH.ETH"].String(), "12560000000") {
		t.Fatalf("fail to get all ft")
	}
}

func testVMExecutorFtDepositExecutor(t *testing.T) {
	block := generateBlock()

	tx1 := types.Transaction{Source: "0x001", Type: types.TransactionTypeFTDepositAck}
	tx1.Data = `{"FtId":"dfaefeafe","Amount":"12.56","Addr":"0x12345abcde","ContractAddr":"0xdeadbeef","TxId":"0xaaaa"}`

	block.Transactions = append(block.Transactions, &tx1)
	accountDB := getTestAccountDB()
	executor := newVMExecutor(accountDB, block, "testing")
	stateRoot, evictedTxs, transactions, receipts := executor.Execute()

	if 0 != strings.Compare("2aa51016b4b5fa786ae5f2e69c9a6b59c5ad22df44bd00445d2ed334aa01be84", common.Bytes2Hex(stateRoot[:])) {
		t.Fatalf("fail to get stateRoot. %s", common.Bytes2Hex(stateRoot[:]))
	}
	if 0 != len(evictedTxs) {
		t.Fatalf("fail to get evictedTxs")
	}
	if 1 != len(transactions) {
		t.Fatalf("fail to get transactions")
	}
	if 1 != len(receipts) || 0 != strings.Compare(receipts[0].Msg, "coin: dfaefeafe, deposit 12560000000") {
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
	ft := accountDB.GetFT(common.HexToAddress(tx1.Source), "dfaefeafe")
	if nil == ft || 0 != strings.Compare(ft.String(), "12560000000") {
		t.Fatalf("fail to get ft")
	}

	ftMap := accountDB.GetAllFT(common.HexToAddress(tx1.Source))
	if nil == ftMap || 1 != len(ftMap) || 0 != strings.Compare(ftMap["dfaefeafe"].String(), "12560000000") {
		t.Fatalf("fail to get all ft")
	}
}

// NFTDeposit without appid
func testVMExecutorNFTDepositExecutor(t *testing.T) {
	block := generateBlock()

	tx1 := types.Transaction{Source: "0x001", Type: types.TransactionTypeNFTDepositAck}
	tx1.Data = `{"SetId":"dfaefeafe","Name":"abc","Symbol":"hhh","Amount":"12.56","ID":"dafrefae","Creator":"mmm","CreateTime":"15348638486","Owner":"deadfa","Value":"dfaefqaefewfa","Addr":"0x12345abcde","ContractAddr":"0xdeadbeef","TxId":"0xaaaa"}`

	block.Transactions = append(block.Transactions, &tx1)
	accountDB := getTestAccountDB()
	executor := newVMExecutor(accountDB, block, "testing")
	stateRoot, evictedTxs, transactions, receipts := executor.Execute()

	if 0 != strings.Compare("77a815747270f77bd0a5bc7daa4b7d3a9f2e6285d86f0e630c4c1bf930c0d535", common.Bytes2Hex(stateRoot[:])) {
		t.Fatalf("fail to get stateRoot. %s", common.Bytes2Hex(stateRoot[:]))
	}
	if 0 != len(evictedTxs) {
		t.Fatalf("fail to get evictedTxs")
	}
	if 1 != len(transactions) {
		t.Fatalf("fail to get transactions")
	}
	if 1 != len(receipts) || 0 != strings.Compare(receipts[0].Msg, "depositNFT result: nft mint successful. setId: dfaefeafe,id: dafrefae, true") {
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
	nft := accountDB.GetNFTById("dfaefeafe", "dafrefae")
	if nil == nft {
		t.Fatalf("fail to get nft")
	}

	nftList := accountDB.GetAllNFT(common.HexToAddress(tx1.Source))
	if nil == nftList || 1 != len(nftList) || 0 != strings.Compare(nftList[0].SetID, "dfaefeafe") ||
		0 != strings.Compare(nftList[0].ID, "dafrefae") {
		t.Fatalf("fail to get all nft")
	}

	nftList = accountDB.GetAllNFTByGameId(common.HexToAddress(tx1.Source), "")
	if nil == nftList || 1 != len(nftList) || 0 != strings.Compare(nftList[0].SetID, "dfaefeafe") ||
		0 != strings.Compare(nftList[0].ID, "dafrefae") {
		t.Fatalf("fail to get all nft by null gameId")
	}

	nftList = accountDB.GetAllNFTByGameId(common.HexToAddress(tx1.Source), "1")
	if nil != nftList && 0 != len(nftList) {
		t.Fatalf("fail to get all nft by null gameId")
	}
}

// NFTDeposit with appid
func testVMExecutorNFTDepositExecutorWithAppId(t *testing.T) {
	block := generateBlock()

	tx1 := types.Transaction{Source: "0x001", Type: types.TransactionTypeNFTDepositAck, Target: "abcdefg"}
	tx1.Data = `{"SetId":"dfaefeafe","Name":"abc","Symbol":"hhh","Amount":"12.56","ID":"dafrefae","Creator":"mmm","CreateTime":"15348638486","Owner":"deadfa","Value":"dfaefqaefewfa","Addr":"0x12345abcde","ContractAddr":"0xdeadbeef","TxId":"0xaaaa"}`

	block.Transactions = append(block.Transactions, &tx1)
	accountDB := getTestAccountDB()
	executor := newVMExecutor(accountDB, block, "testing")
	stateRoot, evictedTxs, transactions, receipts := executor.Execute()

	if 0 != strings.Compare("31e7cd91406e10a2e6bb0fd0bcb3dd3e88e8dbf43a7e29cdf1eaad86c3b4303d", common.Bytes2Hex(stateRoot[:])) {
		t.Fatalf("fail to get stateRoot. %s", common.Bytes2Hex(stateRoot[:]))
	}
	if 0 != len(evictedTxs) {
		t.Fatalf("fail to get evictedTxs")
	}
	if 1 != len(transactions) {
		t.Fatalf("fail to get transactions")
	}
	if 1 != len(receipts) || 0 != strings.Compare(receipts[0].Msg, "depositNFT result: nft mint successful. setId: dfaefeafe,id: dafrefae, true") {
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
	nft := accountDB.GetNFTById("dfaefeafe", "dafrefae")
	if nil == nft {
		t.Fatalf("fail to get nft")
	}

	nftList := accountDB.GetAllNFT(common.HexToAddress(tx1.Source))
	if nil == nftList || 1 != len(nftList) || 0 != strings.Compare(nftList[0].SetID, "dfaefeafe") ||
		0 != strings.Compare(nftList[0].ID, "dafrefae") {
		t.Fatalf("fail to get all nft")
	}

	nftList = accountDB.GetAllNFTByGameId(common.HexToAddress(tx1.Source), tx1.Target)
	if nil == nftList || 1 != len(nftList) || 0 != strings.Compare(nftList[0].SetID, "dfaefeafe") ||
		0 != strings.Compare(nftList[0].ID, "dafrefae") {
		t.Fatalf("fail to get all nft by null gameId")
	}

	nftList = accountDB.GetAllNFTByGameId(common.HexToAddress(tx1.Source), "1")
	if nil != nftList && 0 != len(nftList) {
		t.Fatalf("fail to get all nft by null gameId")
	}
}
