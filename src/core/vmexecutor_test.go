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
	"com.tuntun.rocket/node/src/middleware/notify"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
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
		testVMExecutorFeeFail,
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
	middleware.InitMiddleware("")
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
	kinds := []int32{types.TransactionTypeOperatorEvent, types.TransactionTypeMinerApply,
		types.TransactionTypeMinerAdd, types.TransactionTypeMinerRefund,
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
	service.InitRefundManager(groupChainImpl, SyncProcessor)
}

func teardown(id string) {
	clean()
	fmt.Printf("After %s test\n", id)
}

func TestNilPoint(t *testing.T) {
	id := "0xe059d17139e2915d270ef8f3eee2f3e1438546ba2f06eb674dda0967846b6951"

	common.InitConf("log")
	notify.BUS = notify.NewBus()
	privateKey := common.HexStringToSecKey("0x8c986d44fac408757d77a9ab065ac0ba50f8163c4e4c035dc3a3da79e07bc364")
	syncLogger = log.GetLoggerByIndex(log.SyncLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	InitSyncProcessor(*privateKey, id)

	go func() {
		for {
			mockBlockMsg()
			time.Sleep(time.Millisecond * 100)
		}
	}()

	//go func() {
	//	for i:=0;i<100;i++{
	//		go mockRcv()
	//	}
	//}()

	//go mockSetNil()

	go func() {
		time.Sleep(time.Second * 3)
		SyncProcessor = nil
		notify.BUS.Subscribe(notify.BlockResponse, SyncProcessor.blockResponseMsgHandler)

		//notify.BUS.Subscribe(notify.BlockResponse, nil)
		fmt.Printf("after set\n")
	}()

	for {

	}
}

func mockRcv() {
	for {
		mockBlockMsg()
		time.Sleep(time.Millisecond * 100)
	}
}

func mockSetNil() {
	for {
		time.Sleep(time.Second * 3)
		SyncProcessor = nil
		//notify.BUS.Subscribe(notify.BlockResponse, SyncProcessor.blockResponseMsgHandler)

		fmt.Printf("after set\n")
	}
}

func mockBlockMsg() {
	privateKey := common.HexStringToSecKey("0x8c986d44fac408757d77a9ab065ac0ba50f8163c4e4c035dc3a3da79e07bc364")
	id := "0xe059d17139e2915d270ef8f3eee2f3e1438546ba2f06eb674dda0967846b6951"

	var block *types.Block
	isLastBlock := false
	response := blockMsgResponse{Block: block, IsLastBlock: isLastBlock}
	response.SignInfo = common.NewSignData(*privateKey, id, &response)
	body, e := marshalBlockMsgResponse(response)
	if e != nil {
		fmt.Printf("Marshal block msg response error:%s\n", e.Error())
		return
	}
	msg := notify.BlockResponseMessage{BlockResponseByte: body, Peer: id}
	notify.BUS.Publish(notify.BlockResponse, &msg)
}

var sampleId = "0x7f0746723b141b79b802eb48eb556178fd622201"

func TestNil(t *testing.T) {
	common.InitConf("1.ini")
	syncLogger = log.GetLoggerByIndex(log.SyncLogConfig, common.GlobalConf.GetString("instance", "index", "1"))
	sk := common.HexStringToSecKey("0x8c986d44fac408757d77a9ab065ac0ba50f8163c4e4c035dc3a3da79e07bc364")
	notify.BUS = notify.NewBus()
	initPeerManager(syncLogger)
	InitSyncProcessor(*sk, sampleId)

	go func() {
		for i := 0; i < 100; i++ {
			go set()
		}
	}()

	go func() {
		for i := 0; i < 10; i++ {
			go publishBus()
			//syncLogger.Debugf("[BlockResponseMessage]Unexpected candidate! Expect from:%s, actual:%s,!", SyncProcessor.candidateInfo.Id, "111")
			//syncLogger.Debugf("111")
			//			time.Sleep(time.Second * 1)
		}
	}()
	for {
	}
}

func publishBus() {
	for {
		msg := notify.BlockResponseMessage{BlockResponseByte: nil, Peer: "1111"}
		notify.BUS.Publish(notify.BlockResponse, &msg)
		randInterval := rand.Intn(1000)
		time.Sleep(time.Millisecond * time.Duration(randInterval))
	}
}

func set() {
	for {
		id := randStringRunes(32)
		SyncProcessor.candidateInfo = CandidateInfo{Id: id}

		randInterval := rand.Intn(1000)
		time.Sleep(time.Millisecond * time.Duration(randInterval))
	}
}

//func read() {
//	for {
//		if sampleId != SyncProcessor.candidateInfo.Id {
//			a := SyncProcessor.candidateInfo.Id
//			fmt.Printf("%s\n", a)
//			//fmt.Printf("%s\n", SyncProcessor.candidateInfo.Id)
//		}
//	}
//}

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
