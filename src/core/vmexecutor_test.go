package core

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"
	"x/src/common"
	"x/src/middleware"
	"x/src/middleware/log"
	"x/src/middleware/types"
	"x/src/service"
	"x/src/storage/account"
)

func TestError(t *testing.T) {
	var e error
	if e == nil {
		fmt.Println("nil")
	} else {
		fmt.Println("not nil")
	}
}

func TestVMExecutor_Execute(t *testing.T) {
	context := make(map[string]interface{})
	context["refund"] = make(map[uint64]RefundInfoList)
	data, _ := json.Marshal(context)
	fmt.Printf("before test, context: %s\n", string(data))

	testMap(context)

	data, _ = json.Marshal(context)
	fmt.Printf("after test, context: %s\n", context)

	refundInfos := getRefundInfo(context)
	refundHeight := uint64(999)
	minerId := common.FromHex("0x0001")
	money := big.NewInt(100)
	refundInfo, ok := refundInfos[refundHeight]
	if ok {
		fmt.Println(string(refundInfo.TOJSON()))
		refundInfo.AddRefundInfo(minerId, money)
	} else {
		refundInfo = RefundInfoList{}
		refundInfo.AddRefundInfo(minerId, money)
		refundInfos[refundHeight] = refundInfo
	}

	a := getRefundInfo(context)
	refundInfo, ok = a[refundHeight]
	if ok {
		fmt.Println(string(refundInfo.TOJSON()))
	}

}

func testMap(context map[string]interface{}) {
	refundInfos := getRefundInfo(context)
	refundHeight := uint64(999)
	minerId := common.FromHex("0x0001")
	money := big.NewInt(100)
	refundInfo, ok := refundInfos[refundHeight]
	if ok {
		refundInfo.AddRefundInfo(minerId, money)
	} else {
		refundInfo = RefundInfoList{}
		refundInfo.AddRefundInfo(minerId, money)
		refundInfos[refundHeight] = refundInfo
	}

	//fmt.Println(context)
	//fmt.Println(refundInfos)
}

func generateBlock() *types.Block {
	block := &types.Block{}
	block.Header = getTestBlockHeader()
	block.Transactions = make([]*types.Transaction, 0)

	return block
}

func TestVMExecutorAll(t *testing.T) {
	fs := []func(*testing.T){testVMExecutorFeeFail, testVMExecutorCoinDeposit, testVMExecutorFtDepositExecutor,
		testVMExecutorNFTDepositExecutor, testVMExecutorNFTDepositExecutorWithAppId, testVMExecutorPublishFTSet,
		testVMExecutorPublishFTSetError, testVMExecutorPublishNFTSet, testVMExecutorPublishNFTSetError,
		testVMExecutorMintFT, testVMExecutorMintFTError, testVMExecutorMintFTGoodAndEvil, testVMExecutorMintNFT,
	}

	for i, f := range fs {
		name := strconv.Itoa(i)
		vmExecutorSetup(name)
		t.Run(name, f)
		teardown(name)
	}
}

func vmExecutorSetup(name string) {
	common.InitConf("1.ini")
	txLogger = log.GetLoggerByIndex(log.TxLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	middleware.InitMiddleware()
	service.InitService()
	setup(name)
	initExecutors()
	initRewardCalculator(MinerManagerImpl, blockChainImpl, groupChainImpl)
}

func testFee(kind int32, t *testing.T) {
	block := generateBlock()

	tx1 := types.Transaction{Source: "0x001", Type: kind}
	block.Transactions = append(block.Transactions, &tx1)
	accountDB := getTestAccountDB()
	executor := newVMExecutor(accountDB, block, "testing")
	stateRoot, evictedTxs, transactions, receipts := executor.Execute()

	if 0 != strings.Compare("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421", common.Bytes2Hex(stateRoot[:])) {
		t.Fatalf("fail to get stateRoot")
	}
	if 0 != len(evictedTxs) {
		t.Fatalf("fail to get evictedTxs")
	}
	if 1 != len(transactions) {
		t.Fatalf("fail to get transactions")
	}
	if 1 != len(receipts) {
		t.Fatalf("fail to get receipts")
	}
	if 0 != strings.Compare("not enough max, addr: 0x001, balance: 0", receipts[0].Msg) {
		t.Fatalf("fail to get receipt[0] msg")
	}
}

// 手续费不够测试
func testVMExecutorFeeFail(t *testing.T) {
	kinds := []int32{types.TransactionTypeOperatorEvent, types.TransactionTypeWithdraw, types.TransactionTypeMinerApply,
		types.TransactionTypeMinerAdd, types.TransactionTypeMinerRefund, types.TransactionTypePublishFT, types.TransactionTypePublishNFTSet,
		types.TransactionTypeMintFT, types.TransactionTypeMintNFT, types.TransactionTypeShuttleNFT, types.TransactionTypeUpdateNFT,
		types.TransactionTypeApproveNFT, types.TransactionTypeRevokeNFT, types.TransactionTypeAddStateMachine, types.TransactionTypeUpdateStorage,
		types.TransactionTypeStartSTM, types.TransactionTypeStopSTM, types.TransactionTypeUpgradeSTM,
		types.TransactionTypeQuitSTM, types.TransactionTypeImportNFT,
	}

	for _, kind := range kinds {
		testFee(kind, t)
	}

}

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

// 正常发nft
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

// 不正常发nft
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
