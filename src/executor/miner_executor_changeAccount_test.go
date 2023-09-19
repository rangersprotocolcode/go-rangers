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
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package executor

import (
	"bytes"
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
)

// 正常validator流程
func testMinerExecutorChangeAccount1(t *testing.T) {
	accountDB := getTestAccountDB()
	minerId := common.FromHex("0x0003")
	account := common.FromHex("0x000005")
	targetAccount := common.FromHex("0x000006")

	addMiner(t, accountDB, minerId, account, common.MinerTypeValidator, common.ValidatorStake*3)
	miner := &types.Miner{
		Id:      minerId,
		Account: targetAccount,
	}
	data, err := json.Marshal(miner)
	if nil != err {
		t.Fatal(err)
	}
	transaction := &types.Transaction{
		Source: common.ToHex(account),
		Data:   string(data),
	}

	processor := &minerChangeAccountExecutor{logger: logger}
	succ, msg := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil)
	if !succ {
		t.Fatalf(msg)
	}
	miner2 := service.MinerManagerImpl.GetMiner(minerId, accountDB)
	if miner2 == nil || 0 != bytes.Compare(miner2.Account, targetAccount) || miner2.ApplyHeight != 10086+common.HeightAfterStake {
		t.Fatalf("errorChangeAccount")
	}
}

// 正常proposer流程
func testMinerExecutorChangeAccount2(t *testing.T) {
	accountDB := getTestAccountDB()
	minerId := common.FromHex("0x0003")
	account := common.FromHex("0x000005")
	targetAccount := common.FromHex("0x000006")

	addMiner(t, accountDB, minerId, account, common.MinerTypeProposer, common.ProposerStake*3)
	miner := &types.Miner{
		Id:      minerId,
		Account: targetAccount,
	}
	data, err := json.Marshal(miner)
	if nil != err {
		t.Fatal(err)
	}
	transaction := &types.Transaction{
		Source: common.ToHex(account),
		Data:   string(data),
	}

	processor := &minerChangeAccountExecutor{logger: logger}
	succ, msg := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil)
	if !succ {
		t.Fatalf(msg)
	}
	miner2 := service.MinerManagerImpl.GetMiner(minerId, accountDB)
	if miner2 == nil || 0 != bytes.Compare(miner2.Account, targetAccount) || miner2.ApplyHeight != 10086+common.HeightAfterStake {
		t.Fatalf("errorChangeAccount")
	}
}

func addMiner(t *testing.T, accountDB *account.AccountDB, id, account []byte, kind byte, stake uint64) {
	balance, _ := utility.StrToBigInt("10000")
	accountDB.SetBalance(common.BytesToAddress(id), balance)
	miner := &types.Miner{
		Id:           id,
		Type:         kind,
		Stake:        stake,
		PublicKey:    []byte{0, 1, 2, 3},
		VrfPublicKey: []byte{4, 5, 6, 7},
		ApplyHeight:  10000000,
		Account:      account,
	}
	data, err := json.Marshal(miner)
	if nil != err {
		t.Fatal(err)
	}

	transaction := &types.Transaction{
		Source: common.ToHex(id),
		Data:   string(data),
	}

	processor := &minerApplyExecutor{}
	succ, msg := processor.Execute(transaction, getTestBlockHeader(), accountDB, nil)
	if !succ {
		t.Fatalf(msg)
	}

	miner2 := service.MinerManagerImpl.GetMiner(id, accountDB)
	if miner2 == nil || miner2.Stake != miner.Stake || miner2.ApplyHeight != 10086+common.HeightAfterStake {
		t.Fatalf("error apply miner")
	}

	left := accountDB.GetBalance(common.BytesToAddress(id))
	expect, _ := utility.StrToBigInt(fmt.Sprintf("%d", 10000-stake))
	if left == nil || 0 != left.Cmp(expect) {
		t.Fatalf("error money, %s", left.String())
	}

}
func TestMinerExecutorChangeAccountAll(t *testing.T) {
	fs := []func(*testing.T){
		testMinerExecutorChangeAccount1,
		testMinerExecutorChangeAccount2,
	}

	for i, f := range fs {
		name := strconv.Itoa(i)
		setup(name)
		t.Run(name, f)
		teardown(name)
	}
}
