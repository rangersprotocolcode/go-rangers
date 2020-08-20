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
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"encoding/json"
	"math/big"
	"strconv"
	"testing"
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
	success, msg := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil)
	if success {
		t.Fatalf(msg)
	}

	left := accountDB.GetBalance(common.HexToAddress("0x0003"))
	if nil == left || 0 != left.Cmp(big.NewInt(10000000000000000)) {
		t.Fatalf("error add value")
	}
}

// 正常流程
func testMinerExecutorAdd1(t *testing.T) {
	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress("0x0003"), big.NewInt(100000000000000000))

	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeProposer,
		Stake: common.ProposerStake * 3,
	}
	service.MinerManagerImpl.InsertMiner(miner, accountDB)

	data, _ := json.Marshal(miner)
	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   string(data),
	}

	processor := &minerAddExecutor{}
	success, msg := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil)
	if !success {
		t.Fatalf(msg)
	}

	miner2 := service.MinerManagerImpl.GetMiner(miner.Id, accountDB)
	if nil == miner2 || miner2.Stake != common.ProposerStake*6 {
		t.Fatalf("error add miner")
	}

	left := accountDB.GetBalance(common.HexToAddress("0x0003"))
	if nil == left || 0 != left.Cmp(big.NewInt(85000000000000000)) {
		t.Fatalf("error add value, %d", left)
	}
}

// 正常流程
// 代质押
func testMinerExecutorAdd2(t *testing.T) {
	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress("0x00a3"), big.NewInt(100000000000000000))

	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeProposer,
		Stake: common.ProposerStake * 3,
	}
	service.MinerManagerImpl.InsertMiner(miner, accountDB)

	data, _ := json.Marshal(miner)
	transaction := &types.Transaction{
		Source: "0x00a3",
		Data:   string(data),
	}

	processor := &minerAddExecutor{}
	success, msg := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil)
	if !success {
		t.Fatalf(msg)
	}

	miner2 := service.MinerManagerImpl.GetMiner(miner.Id, accountDB)
	if nil == miner2 || miner2.Stake != common.ProposerStake*6 {
		t.Fatalf("error add miner")
	}

	left := accountDB.GetBalance(common.HexToAddress("0x00a3"))
	if nil == left || 0 != left.Cmp(big.NewInt(85000000000000000)) {
		t.Fatalf("error add value, %d", left)
	}
}

// 正常流程
// 缺少minerId
func testMinerExecutorAdd3(t *testing.T) {
	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress("0x0003"), big.NewInt(100000000000000000))

	miner := &types.Miner{
		Id:    common.FromHex("0x0003"),
		Type:  common.MinerTypeProposer,
		Stake: common.ProposerStake * 3,
	}
	service.MinerManagerImpl.InsertMiner(miner, accountDB)

	miner.Id = []byte{}
	data, _ := json.Marshal(miner)
	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   string(data),
	}

	processor := &minerAddExecutor{}
	success, msg := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil)
	if !success {
		t.Fatalf(msg)
	}

	miner2 := service.MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
	if nil == miner2 || miner2.Stake != common.ProposerStake*6 {
		t.Fatalf("error add miner")
	}

	left := accountDB.GetBalance(common.HexToAddress("0x0003"))
	if nil == left || 0 != left.Cmp(big.NewInt(85000000000000000)) {
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
	service.MinerManagerImpl.InsertMiner(miner, accountDB)

	data, _ := json.Marshal(miner)
	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   string(data),
	}

	processor := &minerAddExecutor{}
	success, msg := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil)
	if success {
		t.Fatalf(msg)
	}

	miner2 := service.MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
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
