package core

import (
	"testing"
	"strconv"
	"math/big"
	"x/src/middleware/types"
	"x/src/common"
	"encoding/json"
)

// 异常流程
// 矿工不存在
func testMinerExecutorAdd(t *testing.T) {
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

	processor := &minerAddExecutor{}
	if processor.Execute(transaction, getTestBlockHeader(), accountDB, nil) {
		t.Fatalf("error add miner")
	}

	left := accountDB.GetBalance(common.HexToAddress("0x0003"))
	if nil == left || 0 != left.Cmp(big.NewInt(10000000000000000)) {
		t.Fatalf("error add value")
	}
}

// 正常流程
func testMinerExecutorAdd1(t *testing.T) {
	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress("0x0003"), big.NewInt(10000000000000000))

	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeProposer,
		Stake: common.ProposerStake * 3,
	}
	MinerManagerImpl.addMiner(miner, accountDB)

	data, _ := json.Marshal(miner)
	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   string(data),
	}

	processor := &minerAddExecutor{}
	if !processor.Execute(transaction, getTestBlockHeader(), accountDB, nil) {
		t.Fatalf("error add miner")
	}

	miner2 := MinerManagerImpl.GetMiner(miner.Id, accountDB)
	if nil == miner2 || miner2.Stake != common.ProposerStake*6 {
		t.Fatalf("error add miner")
	}

	left := accountDB.GetBalance(common.HexToAddress("0x0003"))
	if nil == left || 0 != left.Cmp(big.NewInt(7000000000000000)) {
		t.Fatalf("error add value, %d", left)
	}
}

// 正常流程
// 代质押
func testMinerExecutorAdd2(t *testing.T) {
	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress("0x00a3"), big.NewInt(10000000000000000))

	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeProposer,
		Stake: common.ProposerStake * 3,
	}
	MinerManagerImpl.addMiner(miner, accountDB)

	data, _ := json.Marshal(miner)
	transaction := &types.Transaction{
		Source: "0x00a3",
		Data:   string(data),
	}

	processor := &minerAddExecutor{}
	if !processor.Execute(transaction, getTestBlockHeader(), accountDB, nil) {
		t.Fatalf("error add miner")
	}

	miner2 := MinerManagerImpl.GetMiner(miner.Id, accountDB)
	if nil == miner2 || miner2.Stake != common.ProposerStake*6 {
		t.Fatalf("error add miner")
	}

	left := accountDB.GetBalance(common.HexToAddress("0x00a3"))
	if nil == left || 0 != left.Cmp(big.NewInt(7000000000000000)) {
		t.Fatalf("error add value, %d", left)
	}
}

// 正常流程
// 缺少minerId
func testMinerExecutorAdd3(t *testing.T) {
	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress("0x0003"), big.NewInt(10000000000000000))

	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeProposer,
		Stake: common.ProposerStake * 3,
	}
	MinerManagerImpl.addMiner(miner, accountDB)

	miner.Id = []byte{}
	data, _ := json.Marshal(miner)
	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   string(data),
	}

	processor := &minerAddExecutor{}
	if !processor.Execute(transaction, getTestBlockHeader(), accountDB, nil) {
		t.Fatalf("error add miner")
	}

	miner2 := MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
	if nil == miner2 || miner2.Stake != common.ProposerStake*6 {
		t.Fatalf("error add miner")
	}

	left := accountDB.GetBalance(common.HexToAddress("0x0003"))
	if nil == left || 0 != left.Cmp(big.NewInt(7000000000000000)) {
		t.Fatalf("error add value, %d", left)
	}
}

// 异常流程
// 账户钱不够
func testMinerExecutorAdd4(t *testing.T) {
	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress("0x0003"), big.NewInt(1))

	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeProposer,
		Stake: common.ProposerStake * 3,
	}
	MinerManagerImpl.addMiner(miner, accountDB)

	data, _ := json.Marshal(miner)
	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   string(data),
	}

	processor := &minerAddExecutor{}
	if processor.Execute(transaction, getTestBlockHeader(), accountDB, nil) {
		t.Fatalf("error add miner")
	}

	miner2 := MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
	if nil == miner2 || miner2.Stake != common.ProposerStake*3 {
		t.Fatalf("error add miner")
	}

	left := accountDB.GetBalance(common.HexToAddress("0x0003"))
	if nil == left || 0 != left.Cmp(big.NewInt(1)) {
		t.Fatalf("error add value, %d", left)
	}
}

func TestMinerExecutorAddAll(t *testing.T) {
	fs := []func(*testing.T){testMinerExecutorAdd,
		testMinerExecutorAdd1,
		testMinerExecutorAdd2,
		testMinerExecutorAdd3,
		testMinerExecutorAdd4,}

	for i, f := range fs {
		name := strconv.Itoa(i)
		setup(name)
		t.Run(name, f)
		teardown(name)
	}
}
