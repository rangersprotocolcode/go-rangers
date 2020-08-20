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

package executor

import (
	"bytes"
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/db"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"testing"
)

var (
	groupChainImplLocal dummyGroupChain
	leveldb             *db.LDBDatabase
	triedb              account.AccountDatabase
	logger              log.Logger
)

func getTestAccountDB() *account.AccountDB {
	if nil == leveldb {
		leveldb, _ = db.NewLDBDatabase("test", 0, 0)
		triedb = account.NewDatabase(leveldb)
	}

	accountdb, _ := account.NewAccountDB(common.Hash{}, triedb)
	return accountdb
}

// 正常流程
// 没加入组，退款不影响建组
func testMinerRefundExecutor(t *testing.T) {
	// prepare data
	accountDB := getTestAccountDB()
	context := make(map[string]interface{})
	context["refund"] = make(map[uint64]types.RefundInfoList)

	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeValidator,
		Stake: common.ValidatorStake * 3,
	}
	service.MinerManagerImpl.InsertMiner(miner, accountDB)

	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   "100",
	}

	// run
	processor := minerRefundExecutor{logger: logger}
	if succ, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, context); !succ {
		t.Fatalf("fail to refund")
	}

	// check result
	refundInfos := types.GetRefundInfo(context)
	if nil == refundInfos || 1 != len(refundInfos) {
		t.Fatalf("fail to getRefundInfos")
	}

	refundInfoList, ok := refundInfos[10136]
	if !ok || 1 != len(refundInfoList.List) {
		t.Fatalf("fail to getRefundInfo")
	}

	refundInfo := refundInfoList.List[0]
	if 0 != bytes.Compare(refundInfo.Id, common.FromHex(transaction.Source)) {
		t.Fatalf("fail to refundInfo.Id")
	}
	if 0 != refundInfo.Value.Cmp(big.NewInt(100000000000)) {
		t.Fatalf("fail to refundInfo.Value, %d", refundInfo.Value)
	}
}

// 异常流程
// 金额不够
func testMinerRefundExecutor1(t *testing.T) {
	// prepare data
	accountDB := getTestAccountDB()
	context := make(map[string]interface{})
	context["refund"] = make(map[uint64]types.RefundInfoList)

	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeValidator,
		Stake: common.ValidatorStake * 3,
	}
	service.MinerManagerImpl.InsertMiner(miner, accountDB)

	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   "10000000000",
	}

	// run
	processor := minerRefundExecutor{logger: logger}
	if s, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, context); s {
		t.Fatalf("fail to refund")
	}

	// check result
	refundInfos := types.GetRefundInfo(context)
	if 0 != len(refundInfos) {
		t.Fatalf("fail to getRefundInfos")
	}

	miner2 := service.MinerManagerImpl.GetMiner(miner.Id, accountDB)
	if miner2.Stake != miner.Stake {
		t.Fatalf("fail to miner stake")
	}
}

// 正常流程
// 已经加入了两个组
// 退款不影响两个组
func testMinerRefundExecutor2(t *testing.T) {
	// prepare data
	accountDB := getTestAccountDB()
	context := make(map[string]interface{})
	context["refund"] = make(map[uint64]types.RefundInfoList)

	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeValidator,
		Stake: common.ValidatorStake * 3,
	}
	service.MinerManagerImpl.InsertMiner(miner, accountDB)

	group := &types.Group{}
	group.Id = common.FromHex("01")
	group.Header = &types.GroupHeader{
		WorkHeight:    0,
		DismissHeight: 1000000000,
	}
	group.Members = [][]byte{common.FromHex("0x0003")}
	groupChainImplLocal.save(group)

	group = &types.Group{}
	group.Id = common.FromHex("02")
	group.Header = &types.GroupHeader{
		WorkHeight:    1000,
		DismissHeight: 10000,
		PreGroup:      common.FromHex("01"),
	}
	group.Members = [][]byte{common.FromHex("0x0003")}
	groupChainImplLocal.save(group)

	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   "100",
	}

	// run
	processor := minerRefundExecutor{logger: logger}
	if s, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, context); !s {
		t.Fatalf("fail to refund")
	}

	// check result
	refundInfos := types.GetRefundInfo(context)
	if nil == refundInfos || 1 != len(refundInfos) {
		t.Fatalf("fail to getRefundInfos")
	}

	refundInfoList, ok := refundInfos[10136]
	if !ok || 1 != len(refundInfoList.List) {
		t.Fatalf("fail to getRefundInfo")
	}

	refundInfo := refundInfoList.List[0]
	if 0 != bytes.Compare(refundInfo.Id, common.FromHex(transaction.Source)) {
		t.Fatalf("fail to refundInfo.Id")
	}
	if 0 != refundInfo.Value.Cmp(big.NewInt(100000000000)) {
		t.Fatalf("fail to refundInfo.Value, %d", refundInfo.Value)
	}
	miner2 := service.MinerManagerImpl.GetMiner(miner.Id, accountDB)
	if miner2.Stake != miner.Stake-100 {
		t.Fatalf("fail to miner stake")
	}
}

// 正常流程
// 已经加入了两个组
// 退款后只够一个组
func testMinerRefundExecutor3(t *testing.T) {
	// prepare data
	accountDB := getTestAccountDB()
	context := make(map[string]interface{})
	context["refund"] = make(map[uint64]types.RefundInfoList)

	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeValidator,
		Stake: common.ValidatorStake * 3,
	}
	service.MinerManagerImpl.InsertMiner(miner, accountDB)

	group := &types.Group{}
	group.Id = common.FromHex("01")
	group.Header = &types.GroupHeader{
		WorkHeight:    0,
		DismissHeight: 1000000000,
	}
	group.Members = [][]byte{common.FromHex("0x0003")}
	groupChainImplLocal.save(group)

	group = &types.Group{}
	group.Id = common.FromHex("02")
	group.Header = &types.GroupHeader{
		WorkHeight:    1000,
		DismissHeight: 10000,
		PreGroup:      common.FromHex("01"),
	}
	group.Members = [][]byte{common.FromHex("0x0003")}
	groupChainImplLocal.save(group)

	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   "200000",
	}

	// run
	processor := minerRefundExecutor{logger: logger}
	if s, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, context); !s {
		t.Fatalf("fail to refund")
	}

	// check result
	refundInfos := types.GetRefundInfo(context)
	if nil == refundInfos || 1 != len(refundInfos) {
		t.Fatalf("fail to getRefundInfos")
	}

	refundInfoList, ok := refundInfos[10136]
	if !ok || 1 != len(refundInfoList.List) {
		t.Fatalf("fail to getRefundInfo")
	}

	refundInfo := refundInfoList.List[0]
	if 0 != bytes.Compare(refundInfo.Id, common.FromHex(transaction.Source)) {
		t.Fatalf("fail to refundInfo.Id")
	}
	if 0 != refundInfo.Value.Cmp(big.NewInt(200000000000000)) {
		t.Fatalf("fail to refundInfo.Value, %d", refundInfo.Value)
	}
	miner2 := service.MinerManagerImpl.GetMiner(miner.Id, accountDB)
	if miner2.Stake != miner.Stake-200000 {
		t.Fatalf("fail to miner stake")
	}
}

// 正常流程
// 已经加入了两个组
// 退款后不够一个组，全额退款，矿工删除
func testMinerRefundExecutor4(t *testing.T) {
	// prepare data
	accountDB := getTestAccountDB()
	context := make(map[string]interface{})
	context["refund"] = make(map[uint64]types.RefundInfoList)

	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeValidator,
		Stake: common.ValidatorStake * 3,
	}
	service.MinerManagerImpl.InsertMiner(miner, accountDB)

	group := &types.Group{}
	group.Id = common.FromHex("01")
	group.Header = &types.GroupHeader{
		WorkHeight:    0,
		DismissHeight: 1000000000,
	}
	group.Members = [][]byte{common.FromHex("0x0003")}
	groupChainImplLocal.save(group)

	group = &types.Group{}
	group.Id = common.FromHex("02")
	group.Header = &types.GroupHeader{
		WorkHeight:    1000,
		DismissHeight: 10000,
		PreGroup:      common.FromHex("01"),
	}
	group.Members = [][]byte{common.FromHex("0x0003")}
	groupChainImplLocal.save(group)

	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   "200010",
	}

	// run
	processor := minerRefundExecutor{logger: logger}
	if s, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, context); !s {
		t.Fatalf("fail to refund")
	}

	// check result
	refundInfos := types.GetRefundInfo(context)
	if nil == refundInfos || 1 != len(refundInfos) {
		t.Fatalf("fail to getRefundInfos")
	}

	refundInfoList, ok := refundInfos[1000000050]
	if !ok || 1 != len(refundInfoList.List) {
		t.Fatalf("fail to getRefundInfo")
	}

	refundInfo := refundInfoList.List[0]
	if 0 != bytes.Compare(refundInfo.Id, common.FromHex(transaction.Source)) {
		t.Fatalf("fail to refundInfo.Id")
	}
	if 0 != refundInfo.Value.Cmp(big.NewInt(300000000000000)) {
		t.Fatalf("fail to refundInfo.Value, %d", refundInfo.Value)
	}
	miner2 := service.MinerManagerImpl.GetMiner(miner.Id, accountDB)
	if miner2 != nil {
		t.Fatalf("fail to miner stake")
	}
}

// 正常流程
// 已经加入了三个组
// 退款后不够一个组，全额退款，矿工删除
func testMinerRefundExecutor5(t *testing.T) {
	// prepare data
	accountDB := getTestAccountDB()
	context := make(map[string]interface{})
	context["refund"] = make(map[uint64]types.RefundInfoList)

	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeValidator,
		Stake: common.ValidatorStake * 3,
	}
	service.MinerManagerImpl.InsertMiner(miner, accountDB)

	group := &types.Group{}
	group.Id = common.FromHex("01")
	group.Header = &types.GroupHeader{
		WorkHeight:    0,
		DismissHeight: 1000000000,
	}
	group.Members = [][]byte{common.FromHex("0x0003")}
	groupChainImplLocal.save(group)

	group = &types.Group{}
	group.Id = common.FromHex("02")
	group.Header = &types.GroupHeader{
		WorkHeight:    1000,
		DismissHeight: 10000,
		PreGroup:      common.FromHex("01"),
	}
	group.Members = [][]byte{common.FromHex("0x0003")}
	groupChainImplLocal.save(group)

	group = &types.Group{}
	group.Id = common.FromHex("03")
	group.Header = &types.GroupHeader{
		WorkHeight:    10000,
		DismissHeight: 1000000,
		PreGroup:      common.FromHex("02"),
	}
	group.Members = [][]byte{common.FromHex("0x0003")}
	groupChainImplLocal.save(group)

	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   "200010",
	}

	// run
	processor := minerRefundExecutor{logger: logger}
	if s, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, context); !s {
		t.Fatalf("fail to refund")
	}

	// check result
	refundInfos := types.GetRefundInfo(context)
	if nil == refundInfos || 1 != len(refundInfos) {
		t.Fatalf("fail to getRefundInfos")
	}

	refundInfoList, ok := refundInfos[1000000050]
	if !ok || 1 != len(refundInfoList.List) {
		t.Fatalf("fail to getRefundInfo")
	}

	refundInfo := refundInfoList.List[0]
	if 0 != bytes.Compare(refundInfo.Id, common.FromHex(transaction.Source)) {
		t.Fatalf("fail to refundInfo.Id")
	}
	if 0 != refundInfo.Value.Cmp(big.NewInt(300000000000000)) {
		t.Fatalf("fail to refundInfo.Value, %d", refundInfo.Value)
	}
	miner2 := service.MinerManagerImpl.GetMiner(miner.Id, accountDB)
	if miner2 != nil {
		t.Fatalf("fail to miner stake")
	}
}

// 正常流程
// 已经加入了三个组
// 退款后只够一个组
func testMinerRefundExecutor6(t *testing.T) {
	// prepare data
	accountDB := getTestAccountDB()
	context := make(map[string]interface{})
	context["refund"] = make(map[uint64]types.RefundInfoList)

	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeValidator,
		Stake: common.ValidatorStake * 3,
	}
	service.MinerManagerImpl.InsertMiner(miner, accountDB)

	group := &types.Group{}
	group.Id = common.FromHex("01")
	group.Header = &types.GroupHeader{
		WorkHeight:    0,
		DismissHeight: 1000000000,
	}
	group.Members = [][]byte{common.FromHex("0x0003")}
	groupChainImplLocal.save(group)

	group = &types.Group{}
	group.Id = common.FromHex("02")
	group.Header = &types.GroupHeader{
		WorkHeight:    1000,
		DismissHeight: 10000,
		PreGroup:      common.FromHex("01"),
	}
	group.Members = [][]byte{common.FromHex("0x0003")}
	groupChainImplLocal.save(group)

	group = &types.Group{}
	group.Id = common.FromHex("03")
	group.Header = &types.GroupHeader{
		WorkHeight:    10000,
		DismissHeight: 1000000,
		PreGroup:      common.FromHex("02"),
	}
	group.Members = [][]byte{common.FromHex("0x0003")}
	groupChainImplLocal.save(group)

	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   "100110",
	}

	// run
	processor := minerRefundExecutor{logger: logger}
	if s, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, context); !s {
		t.Fatalf("fail to refund")
	}

	// check result
	refundInfos := types.GetRefundInfo(context)
	if nil == refundInfos || 1 != len(refundInfos) {
		t.Fatalf("fail to getRefundInfos")
	}

	refundInfoList, ok := refundInfos[1000050]
	if !ok || 1 != len(refundInfoList.List) {
		t.Fatalf("fail to getRefundInfo")
	}

	refundInfo := refundInfoList.List[0]
	if 0 != bytes.Compare(refundInfo.Id, common.FromHex(transaction.Source)) {
		t.Fatalf("fail to refundInfo.Id")
	}
	if 0 != refundInfo.Value.Cmp(big.NewInt(100110000000000)) {
		t.Fatalf("fail to refundInfo.Value, %d", refundInfo.Value)
	}
	miner2 := service.MinerManagerImpl.GetMiner(miner.Id, accountDB)
	if miner2 != nil && miner2.Stake != miner.Stake-100110 {
		t.Fatalf("fail to miner stake")
	}
}

// 提案节点
// 正常流程
// 没加入组，退款不影响建组
func testMinerRefundExecutor7(t *testing.T) {
	// prepare data
	accountDB := getTestAccountDB()
	context := make(map[string]interface{})
	context["refund"] = make(map[uint64]types.RefundInfoList)

	miner := &types.Miner{
		Id:    common.FromHex("0xdd03"),
		Type:  common.MinerTypeProposer,
		Stake: common.ProposerStake * 3,
	}
	service.MinerManagerImpl.InsertMiner(miner, accountDB)

	transaction := &types.Transaction{
		Source: "0xdd03",
		Data:   "100",
	}

	// run
	processor := minerRefundExecutor{logger: logger}
	if s, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, context); !s {
		t.Fatalf("fail to refund")
	}

	// check result
	refundInfos := types.GetRefundInfo(context)
	if nil == refundInfos || 1 != len(refundInfos) {
		t.Fatalf("fail to getRefundInfos")
	}

	refundInfoList, ok := refundInfos[10136]
	if !ok || 1 != len(refundInfoList.List) {
		t.Fatalf("fail to getRefundInfo")
	}

	refundInfo := refundInfoList.List[0]
	if 0 != bytes.Compare(refundInfo.Id, common.FromHex(transaction.Source)) {
		t.Fatalf("fail to refundInfo.Id")
	}
	if 0 != refundInfo.Value.Cmp(big.NewInt(100000000000)) {
		t.Fatalf("fail to refundInfo.Value, %d", refundInfo.Value)
	}

	miner2 := service.MinerManagerImpl.GetMiner(miner.Id, accountDB)
	if miner2 != nil && miner2.Stake != miner.Stake-100 {
		t.Fatalf("fail to miner stake")
	}
}

// 提案节点
// 正常流程
// 退款后不够
func testMinerRefundExecutor8(t *testing.T) {
	// prepare data
	accountDB := getTestAccountDB()
	context := make(map[string]interface{})
	context["refund"] = make(map[uint64]types.RefundInfoList)

	miner := &types.Miner{
		Id:    common.FromHex("0xdd03"),
		Type:  common.MinerTypeProposer,
		Stake: common.ProposerStake * 3,
	}
	service.MinerManagerImpl.InsertMiner(miner, accountDB)

	transaction := &types.Transaction{
		Source: "0xdd03",
		Data:   "10000900",
	}

	// run
	processor := minerRefundExecutor{logger: logger}
	if s, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, context); !s {
		t.Fatalf("fail to refund")
	}

	// check result
	refundInfos := types.GetRefundInfo(context)
	if nil == refundInfos || 1 != len(refundInfos) {
		t.Fatalf("fail to getRefundInfos")
	}

	refundInfoList, ok := refundInfos[10136]
	if !ok || 1 != len(refundInfoList.List) {
		t.Fatalf("fail to getRefundInfo")
	}

	refundInfo := refundInfoList.List[0]
	if 0 != bytes.Compare(refundInfo.Id, common.FromHex(transaction.Source)) {
		t.Fatalf("fail to refundInfo.Id")
	}
	if 0 != refundInfo.Value.Cmp(big.NewInt(15000000000000000)) {
		t.Fatalf("fail to refundInfo.Value, %d", refundInfo.Value)
	}

	miner2 := service.MinerManagerImpl.GetMiner(miner.Id, accountDB)
	if miner2 != nil {
		t.Fatalf("fail to miner stake")
	}
}

// 提案节点
// 异常流程
// 金额不够
func testMinerRefundExecutor9(t *testing.T) {
	// prepare data
	accountDB := getTestAccountDB()
	context := make(map[string]interface{})
	context["refund"] = make(map[uint64]types.RefundInfoList)

	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeValidator,
		Stake: common.ValidatorStake * 3,
	}
	service.MinerManagerImpl.InsertMiner(miner, accountDB)

	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   "100000000000000000",
	}

	// run
	processor := minerRefundExecutor{logger: logger}
	if s, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, context); s {
		t.Fatalf("fail to refund")
	}

	// check result
	refundInfos := types.GetRefundInfo(context)
	if 0 != len(refundInfos) {
		t.Fatalf("fail to getRefundInfos")
	}

	miner2 := service.MinerManagerImpl.GetMiner(miner.Id, accountDB)
	if miner2.Stake != miner.Stake {
		t.Fatalf("fail to miner stake")
	}
}

func getTestBlockHeader() *types.BlockHeader {
	header := &types.BlockHeader{
		Height: 10086,
	}

	return header
}

func clean() {
	os.RemoveAll("pkp0")
	os.RemoveAll("logs")
	os.RemoveAll("test")
	os.RemoveAll("database")
	os.RemoveAll("1.ini")
	leveldb = nil
}

func setup(id string) {
	fmt.Printf("Before %s tests\n", id)
	clean()
	common.InitConf("1.ini")
	logger = log.GetLoggerByIndex(log.TxLogConfig, common.GlobalConf.GetString("instance", "index", ""))

	groupChainImplLocal = dummyGroupChain{}
	service.InitMinerManager()
	service.InitRefundManager(&groupChainImplLocal)
}

func teardown(id string) {
	clean()
	fmt.Printf("After %s test\n", id)
}

//func TestMain(m *testing.M) {
//	setup()
//
//	for _, t := range m.{
//		setup()
//		t.Run()  // invokes the current test function
//		if err := cleanup(); err != nil {
//			t.Error("error cleaning up test:", err)
//		}
//	}
//	teardown()
//}

func TestMinerExecutorRefundAll(t *testing.T) {
	fs := []func(*testing.T){testMinerRefundExecutor,
		testMinerRefundExecutor1,
		testMinerRefundExecutor2,
		testMinerRefundExecutor3,
		testMinerRefundExecutor4,
		testMinerRefundExecutor5,
		testMinerRefundExecutor6,
		testMinerRefundExecutor7,
		testMinerRefundExecutor8,
		testMinerRefundExecutor9}

	for i, f := range fs {
		name := strconv.Itoa(i)
		setup(name)
		t.Run(name, f)
		teardown(name)
	}
}

type dummyGroupChain struct {
	groups map[string]*types.Group
}

func (this *dummyGroupChain) GetAvailableGroupsByMinerId(height uint64, minerId []byte) []*types.Group {
	result := make([]*types.Group, 0)
	for _, group := range this.groups {
		if group.Header.DismissHeight < height {
			continue
		}

		for _, mem := range group.Members {
			if bytes.Equal(mem, minerId) {
				result = append(result, group)
				break
			}
		}
	}
	return result
}

func (this *dummyGroupChain) GetGroupById(id []byte) *types.Group {
	return this.groups[common.Bytes2Hex(id)]
}

func (this *dummyGroupChain) save(group *types.Group) {
	if nil == this.groups {
		this.groups = make(map[string]*types.Group)
	}

	this.groups[common.Bytes2Hex(group.Id)] = group
}
