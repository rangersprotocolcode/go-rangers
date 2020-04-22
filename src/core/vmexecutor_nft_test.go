package core

import (
	"encoding/json"
	"math/big"
	"strings"
	"testing"
	"x/src/common"
	"x/src/middleware/types"
	"x/src/service"
	"x/src/storage/account"
)

// 正常发nftSet
func testVMExecutorPublishNFTSet(t *testing.T) {
	block := generateBlock()

	txString := `{"data":"{\"symbol\":\"testNFTSetSymbol\",\"createTime\":\"1585791972730\",\"name\":\"testNFTSetName\",\"setId\":\"1c9b03bf-5975-417f-a6fa-dc098ba8ff2e\",\"maxSupply\":100}","extraData":"","hash":"0xdf3d6b5ade4bd0de884bbf85000d9fa6f7ea91eeb99592f9cf9e825f428c2305","nonce":0,"sign":"0x275eb9cfdc5a850625cad2710fe16cb9d2383c2b9494069721db5660bae033a61ecbde189f03605ebc03916937a54e9942a01c0953cb3a833df86b2e8bdbeb8100","socketRequestId":"6044708977077341138-111","source":"0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443","target":"","time":"1585791972709","type":111}`
	var tx1 types.Transaction
	json.Unmarshal([]byte(txString), &tx1)

	block.Transactions = append(block.Transactions, &tx1)
	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress(tx1.Source), big.NewInt(1000000000000))

	executor := newVMExecutor(accountDB, block, "testing")
	stateRoot, evictedTxs, transactions, receipts := executor.Execute()

	if 0 != strings.Compare("ea5f03b21b10a1298e255fac4b6adf7367d67c075ffecdc872c843bb34b28831", common.Bytes2Hex(stateRoot[:])) {
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
		t.Fatalf("fail to get ftSet")
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

	if 0 != strings.Compare("e93ae5047d6c1a47fad6619d3f3c0ac44184ceb9141f0094a9ef229e1ed98b9b", common.Bytes2Hex(stateRoot[:])) {
		t.Fatalf("fail to get stateRoot. %s", common.Bytes2Hex(stateRoot[:]))
	}
	if 1 != len(evictedTxs) {
		t.Fatalf("fail to get evictedTxs")
	}
	if 1 != len(transactions) {
		t.Fatalf("fail to get transactions")
	}
	if 1 != len(receipts) || 0 != strings.Compare(receipts[0].Msg, "setId or maxSupply wrong, setId: 1c9b03bf-5975-417f-a6fa-dc098ba8ff2e, maxSupply: -100") {
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

	if 0 != strings.Compare("0e2948fe9faf293df1ca17163d99aa1af1354de69c5620969c61d49e601c44db", common.Bytes2Hex(stateRoot[:])) {
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

	nft := accountDB.GetNFTById(common.HexToAddress("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443"), "1c9b03bf-5975-417f-a6fa-dc098ba8ff2e", "123450")
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

	if 0 != strings.Compare("89f96f400c8e228da1528ebc4302b8d58b99193ef464052dbb7292602957426f", common.Bytes2Hex(stateRoot[:])) {
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

	nft := accountDB.GetNFTById(common.HexToAddress("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443"), "1c9b03bf-5975-417f-a6fa-dc098ba8ff2e", "123450")
	if nil == nft || 0 != strings.Compare("5.99", nft.GetData("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443")) {
		t.Fatalf("fail to get nft")
	}

	nft2 := accountDB.GetNFTById(common.HexToAddress("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443"), "1c9b03bf-5975-417f-a6fa-dc098ba8ff2e", "123451")
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

	if 0 != strings.Compare("07658b59eac68eefb103b1e90e86357e50541749cfa23ea9f368297c059082ea", common.Bytes2Hex(stateRoot[:])) {
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
		0 != strings.Compare(receipts[4].Msg, "nft publish successful, setId: onlyOne"){
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
		0 != strings.Compare(nftSet2.OccupiedID["a00"].GetHexString(), "0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf444"){
		t.Fatalf("fail to get nftSet2, %s", nftSet2.OccupiedID["a00"].GetHexString())
	}

	nft := accountDB.GetNFTById(common.HexToAddress("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443"), "1c9b03bf-5975-417f-a6fa-dc098ba8ff2e", "123450")
	if nil == nft || 0 != strings.Compare("5.99", nft.GetData("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443")) {
		t.Fatalf("fail to get nft")
	}

	nft2 := accountDB.GetNFTById(common.HexToAddress("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443"), "1c9b03bf-5975-417f-a6fa-dc098ba8ff2e", "123451")
	if nil == nft2 || 0 != strings.Compare("6.99", nft2.GetData("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443")) {
		t.Fatalf("fail to get nft2")
	}

	nft3 := accountDB.GetNFTById(common.HexToAddress("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf444"), "onlyOne", "a00")
	if nil == nft3 || 0 != strings.Compare("a.99", nft3.GetData("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443")) {
		t.Fatalf("fail to get nft3")
	}

	balance := accountDB.GetBalance(common.HexToAddress(tx1.Source))
	if nil == balance || 0 != balance.Cmp(big.NewInt(999999300000)) {
		t.Fatalf("fee error")
	}
}