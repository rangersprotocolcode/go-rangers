package core

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"x/src/common"
	"x/src/middleware"
	"x/src/middleware/types"
	"x/src/service"
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

func vmExecutorSetup() {
	common.InitConf("1.ini")
	middleware.InitMiddleware()
	service.InitService()
	setup("")
	initExecutors()
	initRewardCalculator(MinerManagerImpl, blockChainImpl, groupChainImpl)
}

func TestVMExecutor_Execute2(t *testing.T) {
	vmExecutorSetup()
	defer teardown("1")

	block := &types.Block{}
	block.Header = getTestBlockHeader()
	block.Transactions = make([]*types.Transaction, 0)

	tx1 := types.Transaction{Source: "0x001", Type: types.TransactionTypeOperatorEvent}
	block.Transactions = append(block.Transactions, &tx1)

	executor := newVMExecutor(getTestAccountDB(), block, "testing")
	stateRoot, evictedTxs, transactions, receipts := executor.Execute()

	fmt.Println(stateRoot)
	fmt.Println(evictedTxs)
	fmt.Println(transactions)
	fmt.Println(receipts)
}
