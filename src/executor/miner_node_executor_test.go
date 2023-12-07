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
	"com.tuntun.rangers/node/src/middleware/log"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/service"
	"com.tuntun.rangers/node/src/storage/account"
	"com.tuntun.rangers/node/src/utility"
	"fmt"
	"strconv"
	"testing"
)

func TestNodeExecutorAddAll(t *testing.T) {
	fs := []func(*testing.T){
		testMinerExecutorNode1,
	}

	for i, f := range fs {
		name := strconv.Itoa(i)
		setup(name)
		t.Run(name, f)
		teardown(name)
	}
}

func testMinerExecutorNode1(t *testing.T) {
	accountDB := getTestAccountDB()
	balance, _ := utility.StrToBigInt("10000")
	accountDB.SetBalance(common.HexToAddress("0x0003"), balance)

	miner := &types.Miner{
		Id:      common.FromHex("0xaaaa"),
		Type:    common.MinerTypeProposer,
		Stake:   common.ProposerStake * 3,
		Account: common.FromHex("0x0003"),
	}
	service.MinerManagerImpl.InsertMiner(miner, accountDB)
	root, _ := accountDB.Commit(true)
	triedb.TrieDB().Commit(root, false)

	accountDB, _ = account.NewAccountDB(root, triedb)

	transaction := &types.Transaction{
		Source: "0x0003",
		Hash:   common.HexToHash("1fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
	}
	logger := log.GetLoggerByIndex(log.TxLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	processor := &minerNodeExecutor{logger: logger}
	success, msg := processor.Execute(transaction, getTestBlockHeader(), accountDB, make(map[string]interface{}))
	if !success {
		t.Fatalf(msg)
	}

	miner2 := service.MinerManagerImpl.GetMiner(miner.Id, accountDB)
	if nil == miner2 || miner2.Stake != common.ProposerStake*3 {
		t.Fatalf("error add miner")
	}

	left := accountDB.GetBalance(common.HexToAddress("0x0003"))
	expect, _ := utility.StrToBigInt(fmt.Sprintf("%d", 10000-10))
	if nil == left || 0 != left.Cmp(expect) {
		t.Fatalf("error add value, %d", left)
	}
}
