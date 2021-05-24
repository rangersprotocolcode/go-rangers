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
	"com.tuntun.rocket/node/src/executor"
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/db"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestVMExecutor_Execute(t *testing.T) {
	context := make(map[string]interface{})
	context["refund"] = make(map[uint64]types.RefundInfoList)
	data, _ := json.Marshal(context)
	fmt.Printf("before test, context: %s\n", string(data))

	testMap(context)

	data, _ = json.Marshal(context)
	fmt.Printf("after test, context: %s\n", context)

	refundInfos := types.GetRefundInfo(context)
	refundHeight := uint64(999)
	minerId := common.FromHex("0x0001")
	money := big.NewInt(100)
	refundInfo, ok := refundInfos[refundHeight]
	if ok {
		fmt.Println(string(refundInfo.TOJSON()))
		refundInfo.AddRefundInfo(minerId, money)
	} else {
		refundInfo = types.RefundInfoList{}
		refundInfo.AddRefundInfo(minerId, money)
		refundInfos[refundHeight] = refundInfo
	}

	a := types.GetRefundInfo(context)
	refundInfo, ok = a[refundHeight]
	if ok {
		fmt.Println(string(refundInfo.TOJSON()))
	}

}

func testMap(context map[string]interface{}) {
	refundInfos := types.GetRefundInfo(context)
	refundHeight := uint64(999)
	minerId := common.FromHex("0x0001")
	money := big.NewInt(100)
	refundInfo, ok := refundInfos[refundHeight]
	if ok {
		refundInfo.AddRefundInfo(minerId, money)
	} else {
		refundInfo = types.RefundInfoList{}
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

func getTestBlockHeader() *types.BlockHeader {
	header := &types.BlockHeader{
		Height: 10086,
	}

	return header
}

func TestVMExecutorAll(t *testing.T) {
	fs := []func(*testing.T){
		testVMExecutorFeeFail, testVMExecutorCoinDeposit, testVMExecutorFtDepositExecutor,
		testVMExecutorNFTDepositExecutor, testVMExecutorNFTDepositExecutorWithAppId, testVMExecutorPublishFTSet,
		testVMExecutorPublishFTSetError, testVMExecutorPublishNFTSet, testVMExecutorPublishNFTSetError,
		testVMExecutorMintFT, testVMExecutorMintFTError, testVMExecutorMintFTGoodAndEvil, testVMExecutorMintNFT,
		testVMExecutorMintNFTWithoutLimit, testVMExecutorMintNFTWithoutLimitGoodAndEvil,
		testVMExecutorJackPot,
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
	executor.InitExecutors()
	service.InitRewardCalculator(blockChainImpl, groupChainImpl, SyncProcessor)
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

var (
	leveldb *db.LDBDatabase
	triedb  account.AccountDatabase
)

func getTestAccountDB() *account.AccountDB {
	if nil == leveldb {
		leveldb, _ = db.NewLDBDatabase("test", 0, 0)
		triedb = account.NewDatabase(leveldb)
	}

	accountdb, _ := account.NewAccountDB(common.Hash{}, triedb)
	return accountdb
}

func clean() {
	os.RemoveAll("storage0")
	os.RemoveAll("1.ini")
	leveldb = nil
}

func setup(id string) {
	fmt.Printf("Before %s tests\n", id)
	clean()
	common.InitConf("1.ini")
	logger = log.GetLoggerByIndex(log.TxLogConfig, common.GlobalConf.GetString("instance", "index", ""))

	service.InitMinerManager()
	service.InitRefundManager(groupChainImpl)
}

func teardown(id string) {
	clean()
	fmt.Printf("After %s test\n", id)
}
