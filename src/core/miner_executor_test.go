package core

import (
	"bytes"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"testing"
	"x/src/common"
	"x/src/middleware/db"
	"x/src/middleware/log"
	"x/src/middleware/types"
	"x/src/storage/account"
)

var groupChainImplLocal *groupChain

// 正常流程
// 没加入组，退款不影响建组
func testMinerRefundExecutor(t *testing.T) {
	// prepare data
	accountDB := getTestAccountDB()
	context := make(map[string]interface{})
	context["refund"] = make(map[uint64]RefundInfoList)

	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeValidator,
		Stake: common.ValidatorStake * 3,
	}
	MinerManagerImpl.addMiner(miner, accountDB)

	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   "100",
	}

	// run
	processor := minerRefundExecutor{}
	if succ, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, context); !succ {
		t.Fatalf("fail to refund")
	}

	// check result
	refundInfos := getRefundInfo(context)
	if nil == refundInfos || 1 != len(refundInfos) {
		t.Fatalf("fail to getRefundInfos")
	}

	refundInfoList, ok := refundInfos[10140]
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
	context["refund"] = make(map[uint64]RefundInfoList)

	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeValidator,
		Stake: common.ValidatorStake * 3,
	}
	MinerManagerImpl.addMiner(miner, accountDB)

	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   "10000000000",
	}

	// run
	processor := minerRefundExecutor{}
	if s, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, context); s {
		t.Fatalf("fail to refund")
	}

	// check result
	refundInfos := getRefundInfo(context)
	if 0 != len(refundInfos) {
		t.Fatalf("fail to getRefundInfos")
	}

	miner2 := MinerManagerImpl.GetMiner(miner.Id, accountDB)
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
	context["refund"] = make(map[uint64]RefundInfoList)

	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeValidator,
		Stake: common.ValidatorStake * 3,
	}
	MinerManagerImpl.addMiner(miner, accountDB)

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
	processor := minerRefundExecutor{}
	if s, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, context); !s {
		t.Fatalf("fail to refund")
	}

	// check result
	refundInfos := getRefundInfo(context)
	if nil == refundInfos || 1 != len(refundInfos) {
		t.Fatalf("fail to getRefundInfos")
	}

	refundInfoList, ok := refundInfos[10140]
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
	miner2 := MinerManagerImpl.GetMiner(miner.Id, accountDB)
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
	context["refund"] = make(map[uint64]RefundInfoList)

	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeValidator,
		Stake: common.ValidatorStake * 3,
	}
	MinerManagerImpl.addMiner(miner, accountDB)

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
	processor := minerRefundExecutor{}
	if s, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, context); !s {
		t.Fatalf("fail to refund")
	}

	// check result
	refundInfos := getRefundInfo(context)
	if nil == refundInfos || 1 != len(refundInfos) {
		t.Fatalf("fail to getRefundInfos")
	}

	refundInfoList, ok := refundInfos[10140]
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
	miner2 := MinerManagerImpl.GetMiner(miner.Id, accountDB)
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
	context["refund"] = make(map[uint64]RefundInfoList)

	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeValidator,
		Stake: common.ValidatorStake * 3,
	}
	MinerManagerImpl.addMiner(miner, accountDB)

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
	processor := minerRefundExecutor{}
	if s, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, context); !s {
		t.Fatalf("fail to refund")
	}

	// check result
	refundInfos := getRefundInfo(context)
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
	miner2 := MinerManagerImpl.GetMiner(miner.Id, accountDB)
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
	context["refund"] = make(map[uint64]RefundInfoList)

	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeValidator,
		Stake: common.ValidatorStake * 3,
	}
	MinerManagerImpl.addMiner(miner, accountDB)

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
	processor := minerRefundExecutor{}
	if s, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, context); !s {
		t.Fatalf("fail to refund")
	}

	// check result
	refundInfos := getRefundInfo(context)
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
	miner2 := MinerManagerImpl.GetMiner(miner.Id, accountDB)
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
	context["refund"] = make(map[uint64]RefundInfoList)

	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeValidator,
		Stake: common.ValidatorStake * 3,
	}
	MinerManagerImpl.addMiner(miner, accountDB)

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
	processor := minerRefundExecutor{}
	if s, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, context); !s {
		t.Fatalf("fail to refund")
	}

	// check result
	refundInfos := getRefundInfo(context)
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
	miner2 := MinerManagerImpl.GetMiner(miner.Id, accountDB)
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
	context["refund"] = make(map[uint64]RefundInfoList)

	miner := &types.Miner{
		Id:    common.FromHex("0xdd03"),
		Type:  common.MinerTypeProposer,
		Stake: common.ProposerStake * 3,
	}
	MinerManagerImpl.addMiner(miner, accountDB)

	transaction := &types.Transaction{
		Source: "0xdd03",
		Data:   "100",
	}

	// run
	processor := minerRefundExecutor{}
	if s, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, context); !s {
		t.Fatalf("fail to refund")
	}

	// check result
	refundInfos := getRefundInfo(context)
	if nil == refundInfos || 1 != len(refundInfos) {
		t.Fatalf("fail to getRefundInfos")
	}

	refundInfoList, ok := refundInfos[10140]
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

	miner2 := MinerManagerImpl.GetMiner(miner.Id, accountDB)
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
	context["refund"] = make(map[uint64]RefundInfoList)

	miner := &types.Miner{
		Id:    common.FromHex("0xdd03"),
		Type:  common.MinerTypeProposer,
		Stake: common.ProposerStake * 3,
	}
	MinerManagerImpl.addMiner(miner, accountDB)

	transaction := &types.Transaction{
		Source: "0xdd03",
		Data:   "10000900",
	}

	// run
	processor := minerRefundExecutor{}
	if s, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, context); !s {
		t.Fatalf("fail to refund")
	}

	// check result
	refundInfos := getRefundInfo(context)
	if nil == refundInfos || 1 != len(refundInfos) {
		t.Fatalf("fail to getRefundInfos")
	}

	refundInfoList, ok := refundInfos[10140]
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

	miner2 := MinerManagerImpl.GetMiner(miner.Id, accountDB)
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
	context["refund"] = make(map[uint64]RefundInfoList)

	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeValidator,
		Stake: common.ValidatorStake * 3,
	}
	MinerManagerImpl.addMiner(miner, accountDB)

	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   "100000000000000000",
	}

	// run
	processor := minerRefundExecutor{}
	if s, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, context); s {
		t.Fatalf("fail to refund")
	}

	// check result
	refundInfos := getRefundInfo(context)
	if 0 != len(refundInfos) {
		t.Fatalf("fail to getRefundInfos")
	}

	miner2 := MinerManagerImpl.GetMiner(miner.Id, accountDB)
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

func getTestAccountDB() *account.AccountDB {
	db, _ := db.NewLDBDatabase("test", 0, 0)
	triedb := account.NewDatabase(db)
	accountdb, _ := account.NewAccountDB(common.Hash{}, triedb)

	return accountdb
}

func getTestAccountDBWithRoot(root common.Hash) *account.AccountDB {
	db, err := db.NewLDBDatabase("test", 0, 0)
	if err != nil {
		panic(err)
	}

	triedb := account.NewDatabase(db)
	accountdb, _ := account.NewAccountDB(root, triedb)

	return accountdb
}

func clean() {
	os.RemoveAll("logs")
	os.RemoveAll("test")
	os.RemoveAll("database")
	os.RemoveAll("1.ini")
}
func setup(id string) {
	fmt.Printf("Before %s tests\n", id)
	clean()
	logger = log.GetLoggerByIndex(log.CoreLogConfig, "0")

	chain := &groupChain{}
	var err error
	chain.groups, err = db.NewDatabase(groupChainPrefix)
	if err != nil {
		panic("Init group chain error:" + err.Error())
	}
	groupChainImplLocal = chain
	groupChainImpl = chain

	RefundManagerImpl = &RefundManager{}
	RefundManagerImpl.logger = log.GetLoggerByIndex(log.RefundLogConfig, "0")

	initMinerManager()
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
		testMinerRefundExecutor9,}

	for i, f := range fs {
		name := strconv.Itoa(i)
		setup(name)
		t.Run(name, f)
		teardown(name)
	}
}
