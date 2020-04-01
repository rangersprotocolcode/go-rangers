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
		testVMExecutorPublishFTSetError,
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

// 手续费测试
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
