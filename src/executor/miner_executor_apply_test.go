// Copyright 2020 The RangersProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RangersProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RangersProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package executor

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/service"
	"com.tuntun.rangers/node/src/utility"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"testing"
)

func testMinerExecutorApply(t *testing.T) {
	accountDB := getTestAccountDB()

	miner := &types.Miner{
		Id:      common.FromHex("0x0003"),
		Type:    common.MinerTypeValidator,
		Stake:   common.ValidatorStake * 3,
		Account: common.FromHex("0x000005"),
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
	miner2 := service.MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
	if miner2 != nil {
		t.Fatalf("error apply miner")
	}
}

func testMinerExecutorApply1(t *testing.T) {
	accountDB := getTestAccountDB()
	balance, _ := utility.StrToBigInt("10000")
	accountDB.SetBalance(common.HexToAddress("0x0003"), balance)
	miner := &types.Miner{
		Id:           common.FromHex("0x0003"),
		Type:         common.MinerTypeValidator,
		Stake:        common.ValidatorStake * 3,
		PublicKey:    []byte{0, 1, 2, 3},
		VrfPublicKey: []byte{4, 5, 6, 7},
		ApplyHeight:  10000000,
		Account:      common.FromHex("0x000005"),
	}
	data, err := json.Marshal(miner)
	if nil != err {
		t.Fatal(err)
	}

	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   string(data),
	}

	processor := &minerApplyExecutor{}
	succ, msg := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil)
	if !succ {
		t.Fatalf(msg)
	}

	miner2 := service.MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
	if miner2 == nil || miner2.Stake != miner.Stake || miner2.ApplyHeight != 10086+common.HeightAfterStake {
		t.Fatalf("error apply miner")
	}

	left := accountDB.GetBalance(common.HexToAddress("0x0003"))
	expect, _ := utility.StrToBigInt(fmt.Sprintf("%d", 10000-common.ValidatorStake*3))
	if left == nil || 0 != left.Cmp(expect) {
		t.Fatalf("error money, %s", left.String())
	}
}

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
	miner2 := service.MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
	if miner2 != nil {
		t.Fatalf("error apply miner")
	}
}

func testMinerExecutorApply3(t *testing.T) {
	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress("0x0003"), big.NewInt(1000000000000000))
	miner := &types.Miner{
		Id:           common.FromHex("0x0003"),
		Type:         common.MinerTypeValidator,
		Stake:        common.ValidatorStake * 3,
		PublicKey:    []byte{0, 1, 2, 3},
		VrfPublicKey: []byte{4, 5, 6, 7},
	}
	data, _ := json.Marshal(miner)

	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   string(data),
	}

	processor := &minerApplyExecutor{}
	if succ, msg := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil); !succ {
		t.Fatalf(msg)
	}

	miner2 := service.MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
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
	miner3 := service.MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
	if miner3 == nil || miner3.Stake != miner.Stake || miner3.ApplyHeight != 10086+common.HeightAfterStake {
		t.Fatalf("error apply miner")
	}

	left2 := accountDB.GetBalance(common.HexToAddress("0x0003"))
	if left2 == nil || 0 != left2.Cmp(big.NewInt(700000000000000)) {
		t.Fatalf("error money")
	}
}

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
	miner2 := service.MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
	if miner2 != nil {
		t.Fatalf("error apply miner")
	}
}

func testMinerExecutorApply5(t *testing.T) {
	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress("0x0003"), big.NewInt(100000000000000000))
	miner := &types.Miner{
		Id:           common.FromHex("0x0003"),
		Type:         common.MinerTypeProposer,
		Stake:        common.ProposerStake * 3,
		PublicKey:    []byte{0, 1, 2, 3},
		VrfPublicKey: []byte{4, 5, 6, 7},
	}
	data, _ := json.Marshal(miner)

	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   string(data),
	}

	processor := &minerApplyExecutor{}
	if succ, msg := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil); !succ {
		t.Fatalf(msg)
	}

	miner2 := service.MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
	if miner2 == nil || miner2.Stake != miner.Stake || miner2.ApplyHeight != 10086+common.HeightAfterStake {
		t.Fatalf("error apply miner")
	}

	left := accountDB.GetBalance(common.HexToAddress("0x0003"))
	if left == nil || 0 != left.Cmp(big.NewInt(85000000000000000)) {
		t.Fatalf("error money")
	}
}

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
	miner2 := service.MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
	if miner2 != nil {
		t.Fatalf("error apply miner")
	}
}

func testMinerExecutorApply7(t *testing.T) {
	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress("0x0003"), big.NewInt(100000000000000000))
	miner := &types.Miner{
		Id:           common.FromHex("0x0003"),
		Type:         common.MinerTypeProposer,
		Stake:        common.ProposerStake * 3,
		PublicKey:    []byte{0, 1, 2, 3},
		VrfPublicKey: []byte{4, 5, 6, 7},
	}
	data, _ := json.Marshal(miner)

	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   string(data),
	}

	processor := &minerApplyExecutor{}
	if succ, msg := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil); !succ {
		t.Fatalf(msg)
	}

	miner2 := service.MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
	if miner2 == nil || miner2.Stake != miner.Stake || miner2.ApplyHeight != 10086+common.HeightAfterStake {
		t.Fatalf("error apply miner")
	}

	left := accountDB.GetBalance(common.HexToAddress("0x0003"))
	if left == nil || 0 != left.Cmp(big.NewInt(85000000000000000)) {
		t.Fatalf("error money")
	}

	if succ, _ := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil); succ {
		t.Fatalf("error apply miner twice")
	}
	miner3 := service.MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
	if miner3 == nil || miner3.Stake != miner.Stake || miner3.ApplyHeight != 10086+common.HeightAfterStake {
		t.Fatalf("error apply miner")
	}

	left2 := accountDB.GetBalance(common.HexToAddress("0x0003"))
	if left2 == nil || 0 != left2.Cmp(big.NewInt(85000000000000000)) {
		t.Fatalf("error money")
	}
}

func testMinerExecutorApply8(t *testing.T) {
	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress("0x0003"), big.NewInt(100000000000000000))
	miner := &types.Miner{
		Id:           common.FromHex("0x0003"),
		Type:         common.MinerTypeProposer,
		Stake:        common.ProposerStake * 3,
		PublicKey:    []byte{0, 1, 2, 3},
		VrfPublicKey: []byte{4, 5, 6, 7},
	}
	data, _ := json.Marshal(miner)

	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   string(data),
	}

	processor := &minerApplyExecutor{}
	if succ, msg := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil); !succ {
		t.Fatalf(msg)
	}

	miner2 := service.MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
	if miner2 == nil || miner2.Stake != miner.Stake || miner2.ApplyHeight != 10086+common.HeightAfterStake {
		t.Fatalf("error apply miner")
	}

	left := accountDB.GetBalance(common.HexToAddress("0x0003"))
	if left == nil || 0 != left.Cmp(big.NewInt(85000000000000000)) {
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
	miner3 := service.MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
	if miner3 == nil || miner3.Stake != miner2.Stake || miner3.ApplyHeight != 10086+common.HeightAfterStake {
		t.Fatalf("error apply miner")
	}

	left2 := accountDB.GetBalance(common.HexToAddress("0x0003"))
	if left2 == nil || 0 != left2.Cmp(big.NewInt(85000000000000000)) {
		t.Fatalf("error money")
	}
}

func testMinerExecutorApply9(t *testing.T) {
	accountDB := getTestAccountDB()
	accountDB.SetBalance(common.HexToAddress("0x0003"), big.NewInt(1000000000000000))
	miner := &types.Miner{
		Type:         common.MinerTypeValidator,
		Stake:        common.ValidatorStake * 3,
		PublicKey:    []byte{0, 1, 2, 3},
		VrfPublicKey: []byte{4, 5, 6, 7},
	}
	data, _ := json.Marshal(miner)

	transaction := &types.Transaction{
		Source: "0x0003",
		Data:   string(data),
	}

	processor := &minerApplyExecutor{}
	if succ, msg := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil); !succ {
		t.Fatalf(msg)
	}

	miner2 := service.MinerManagerImpl.GetMiner(common.FromHex("0x0003"), accountDB)
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
		testMinerExecutorApply9}

	for i, f := range fs {
		name := strconv.Itoa(i)
		setup(name)
		t.Run(name, f)
		teardown(name)
	}
}
