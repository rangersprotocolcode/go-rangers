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

	if 0 != strings.Compare("14497f95a55d510e738040dd23ed6de069ca795823bf6397e94b25cc596fc411", common.Bytes2Hex(stateRoot[:])) {
		t.Fatalf("fail to get stateRoot. %s", common.Bytes2Hex(stateRoot[:]))
	}
	if 0 != len(evictedTxs) {
		t.Fatalf("fail to get evictedTxs")
	}
	if 1 != len(transactions) {
		t.Fatalf("fail to get transactions")
	}
	if 1 != len(receipts) || 0 != strings.Compare(receipts[0].Msg, "coin: official-ETH.ETH, deposit 12560000000") {
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
	ft := accountDB.GetFT(common.HexToAddress(tx1.Source), "official-ETH.ETH")
	if nil == ft || 0 != strings.Compare(ft.String(), "12560000000") {
		t.Fatalf("fail to get ft")
	}

	ftMap := accountDB.GetAllFT(common.HexToAddress(tx1.Source))
	if nil == ftMap || 1 != len(ftMap) || 0 != strings.Compare(ftMap["official-ETH.ETH"].String(), "12560000000") {
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

	if 0 != strings.Compare("cb3684d14734e676961db43c2b7a44f5161594de5db4fad90cea2236d53c4398", common.Bytes2Hex(stateRoot[:])) {
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

	if 0 != strings.Compare("6cf5f1c81b60620708c01020c9a716bcbbc33512fd10c83dd13427fbd992bd36", common.Bytes2Hex(stateRoot[:])) {
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
	nft := accountDB.GetNFTById(common.HexToAddress(tx1.Source), "dfaefeafe", "dafrefae")
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
	if nil != nftList {
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

	if 0 != strings.Compare("fcec4c096c0825e4f3b9e8f43ebeafc52d29237ce02ecdc8fbff2e1c28759e0d", common.Bytes2Hex(stateRoot[:])) {
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
	nft := accountDB.GetNFTById(common.HexToAddress(tx1.Source), "dfaefeafe", "dafrefae")
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
	if nil != nftList {
		t.Fatalf("fail to get all nft by null gameId")
	}
}

