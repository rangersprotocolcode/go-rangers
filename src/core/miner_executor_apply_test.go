package core

import (
	"encoding/json"
	"math/big"
	"strconv"
	"testing"
	"x/src/common"
	"x/src/middleware/types"
)

// 异常流程
// 账户钱不够
func testMinerExecutorApply(t *testing.T) {
	accountDB := getTestAccountDB()

	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeValidator,
		Stake: common.ValidatorStake * 3,
	}
	data, _ := json.Marshal(miner)

	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   string(data),
	}

	processor := &minerApplyExecutor{}
	succ, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil)
	if succ {
		t.Fatalf("error apply miner")
	}
	miner2 := MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
	if miner2 != nil {
		t.Fatalf("error apply miner")
	}
}

// 正常流程
func testMinerExecutorApply1(t *testing.T) {
	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress("0x0003"), big.NewInt(1000000000000000))
	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeValidator,
		Stake: common.ValidatorStake * 3,
	}
	data, _ := json.Marshal(miner)

	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   string(data),
	}

	processor := &minerApplyExecutor{}
	succ, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil)
	if !succ {
		t.Fatalf("error apply miner")
	}

	miner2 := MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
	if miner2 == nil || miner2.Stake != miner.Stake || miner2.ApplyHeight != 10086+common.HeightAfterStake {
		t.Fatalf("error apply miner")
	}

	left := accountDB.GetBalance(common.HexToAddress("0x0003"))
	if left == nil || 0 != left.Cmp(big.NewInt(700000000000000)) {
		t.Fatalf("error money")
	}
}

// 异常流程
// 质押钱不够
func testMinerExecutorApply2(t *testing.T) {
	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress("0x0003"), big.NewInt(1000000000000000))
	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeValidator,
		Stake: common.ValidatorStake - 1,
	}
	data, _ := json.Marshal(miner)

	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   string(data),
	}

	processor := &minerApplyExecutor{}
	if succ, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil); succ {
		t.Fatalf("error apply miner")
	}
	miner2 := MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
	if miner2 != nil {
		t.Fatalf("error apply miner")
	}
}

// 异常流程
// 重复申请
func testMinerExecutorApply3(t *testing.T) {
	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress("0x0003"), big.NewInt(1000000000000000))
	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeValidator,
		Stake: common.ValidatorStake * 3,
	}
	data, _ := json.Marshal(miner)

	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   string(data),
	}

	processor := &minerApplyExecutor{}
	if succ, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil); !succ {
		t.Fatalf("error apply miner")
	}

	miner2 := MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
	if miner2 == nil || miner2.Stake != miner.Stake || miner2.ApplyHeight != 10086+common.HeightAfterStake {
		t.Fatalf("error apply miner")
	}

	left := accountDB.GetBalance(common.HexToAddress("0x0003"))
	if left == nil || 0 != left.Cmp(big.NewInt(700000000000000)) {
		t.Fatalf("error money")
	}

	if succ, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil); succ {
		t.Fatalf("error apply miner twice")
	}
	miner3 := MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
	if miner3 == nil || miner3.Stake != miner.Stake || miner3.ApplyHeight != 10086+common.HeightAfterStake {
		t.Fatalf("error apply miner")
	}

	left2 := accountDB.GetBalance(common.HexToAddress("0x0003"))
	if left2 == nil || 0 != left2.Cmp(big.NewInt(700000000000000)) {
		t.Fatalf("error money")
	}
}

// 以下提案节点

// 异常流程
// 账户钱不够
func testMinerExecutorApply4(t *testing.T) {
	accountDB := getTestAccountDB()

	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeProposer,
		Stake: common.ProposerStake * 3,
	}
	data, _ := json.Marshal(miner)

	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   string(data),
	}

	processor := &minerApplyExecutor{}
	if succ, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil); succ {
		t.Fatalf("error apply miner")
	}
	miner2 := MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
	if miner2 != nil {
		t.Fatalf("error apply miner")
	}
}

// 正常流程
func testMinerExecutorApply5(t *testing.T) {
	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress("0x0003"), big.NewInt(10000000000000000))
	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeProposer,
		Stake: common.ProposerStake * 3,
	}
	data, _ := json.Marshal(miner)

	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   string(data),
	}

	processor := &minerApplyExecutor{}
	if succ, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil); !succ {
		t.Fatalf("error apply miner")
	}

	miner2 := MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
	if miner2 == nil || miner2.Stake != miner.Stake || miner2.ApplyHeight != 10086+common.HeightAfterStake {
		t.Fatalf("error apply miner")
	}

	left := accountDB.GetBalance(common.HexToAddress("0x0003"))
	if left == nil || 0 != left.Cmp(big.NewInt(7000000000000000)) {
		t.Fatalf("error money")
	}
}

// 异常流程
// 质押钱不够
func testMinerExecutorApply6(t *testing.T) {
	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress("0x0003"), big.NewInt(1000000000000000))
	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeProposer,
		Stake: common.ProposerStake - 100,
	}
	data, _ := json.Marshal(miner)

	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   string(data),
	}

	processor := &minerApplyExecutor{}
	if succ, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil); succ {
		t.Fatalf("error apply miner")
	}
	miner2 := MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
	if miner2 != nil {
		t.Fatalf("error apply miner")
	}
}

// 异常流程
// 重复申请
func testMinerExecutorApply7(t *testing.T) {
	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress("0x0003"), big.NewInt(10000000000000000))
	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeProposer,
		Stake: common.ProposerStake * 3,
	}
	data, _ := json.Marshal(miner)

	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   string(data),
	}

	processor := &minerApplyExecutor{}
	if succ, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil); !succ {
		t.Fatalf("error apply miner")
	}

	miner2 := MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
	if miner2 == nil || miner2.Stake != miner.Stake || miner2.ApplyHeight != 10086+common.HeightAfterStake {
		t.Fatalf("error apply miner")
	}

	left := accountDB.GetBalance(common.HexToAddress("0x0003"))
	if left == nil || 0 != left.Cmp(big.NewInt(7000000000000000)) {
		t.Fatalf("error money")
	}

	if succ, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil); succ {
		t.Fatalf("error apply miner twice")
	}
	miner3 := MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
	if miner3 == nil || miner3.Stake != miner.Stake || miner3.ApplyHeight != 10086+common.HeightAfterStake {
		t.Fatalf("error apply miner")
	}

	left2 := accountDB.GetBalance(common.HexToAddress("0x0003"))
	if left2 == nil || 0 != left2.Cmp(big.NewInt(7000000000000000)) {
		t.Fatalf("error money")
	}
}

// 异常流程
// 重复申请
func testMinerExecutorApply8(t *testing.T) {
	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress("0x0003"), big.NewInt(10000000000000000))
	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeProposer,
		Stake: common.ProposerStake * 3,
	}
	data, _ := json.Marshal(miner)

	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   string(data),
	}

	processor := &minerApplyExecutor{}
	if succ, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil); !succ {
		t.Fatalf("error apply miner")
	}

	miner2 := MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
	if miner2 == nil || miner2.Stake != miner.Stake || miner2.ApplyHeight != 10086+common.HeightAfterStake {
		t.Fatalf("error apply miner")
	}

	left := accountDB.GetBalance(common.HexToAddress("0x0003"))
	if left == nil || 0 != left.Cmp(big.NewInt(7000000000000000)) {
		t.Fatalf("error money")
	}

	miner = &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeValidator,
		Stake: common.ValidatorStake * 3,
	}
	data, _ = json.Marshal(miner)

	transaction = &types.Transaction{
		Source: "0x0003",
		Data:   string(data),
	}

	if succ, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil); succ {
		t.Fatalf("error apply miner twice")
	}
	miner3 := MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
	if miner3 == nil || miner3.Stake != miner2.Stake || miner3.ApplyHeight != 10086+common.HeightAfterStake {
		t.Fatalf("error apply miner")
	}

	left2 := accountDB.GetBalance(common.HexToAddress("0x0003"))
	if left2 == nil || 0 != left2.Cmp(big.NewInt(7000000000000000)) {
		t.Fatalf("error money")
	}
}

// 正常流程
// 默认账户
func testMinerExecutorApply9(t *testing.T) {
	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress("0x0003"), big.NewInt(1000000000000000))
	miner := &types.Miner{
		Type:  common.MinerTypeValidator,
		Stake: common.ValidatorStake * 3,
	}
	data, _ := json.Marshal(miner)

	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   string(data),
	}

	processor := &minerApplyExecutor{}
	if succ, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil); !succ {
		t.Fatalf("error apply miner")
	}

	miner2 := MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
	if miner2 == nil || miner2.Stake != miner.Stake || miner2.ApplyHeight != 10086+common.HeightAfterStake {
		t.Fatalf("error apply miner")
	}

	left := accountDB.GetBalance(common.HexToAddress("0x0003"))
	if left == nil || 0 != left.Cmp(big.NewInt(700000000000000)) {
		t.Fatalf("error money")
	}
}

func TestMinerExecutorApplyAll(t *testing.T) {
	fs := []func(*testing.T){testMinerExecutorApply,
		testMinerExecutorApply1,
		testMinerExecutorApply2,
		testMinerExecutorApply3,
		testMinerExecutorApply4,
		testMinerExecutorApply5,
		testMinerExecutorApply6,
		testMinerExecutorApply7,
		testMinerExecutorApply8,
		testMinerExecutorApply9,}

	for i, f := range fs {
		name := strconv.Itoa(i)
		setup(name)
		t.Run(name, f)
		teardown(name)
	}
}
