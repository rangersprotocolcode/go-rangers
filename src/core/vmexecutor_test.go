package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"
)

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
		testVMExecutorMintNFTWithoutLimit,testVMExecutorMintNFTWithoutLimitGoodAndEvil,
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
