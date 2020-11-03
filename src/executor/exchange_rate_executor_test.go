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
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

func getAllRates(accountDB *account.AccountDB) string {
	iterator := accountDB.DataIterator(common.ExchangeRateAddress, []byte(""))
	if nil == iterator {
		return ""
	}

	rate := make(map[string]string)
	for iterator.Next() {
		rate[string(iterator.Key)] = string(iterator.Value)
	}

	data, _ := json.Marshal(rate)
	return string(data)
}

// 正常流程
// set 1 rate
func testExchangeRateExecutor(t *testing.T) {
	// prepare data
	accountDB := getTestAccountDB()
	context := make(map[string]interface{})
	context["refund"] = make(map[uint64]types.RefundInfoList)

	transaction := &types.Transaction{
		Source: "0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443",
		Data:   "{\"ETH.ETH\":\"200\"}",
	}

	// run
	processor := exchangeRateExecutor{}
	succ, err := processor.Execute(transaction, getTestBlockHeader(), accountDB, context)
	if !succ {
		t.Fatalf(err)
	}
	accountDB.Commit(true)

	rate := accountDB.GetData(common.ExchangeRateAddress, []byte("ETH.ETH"))
	if strings.Compare("200", string(rate)) != 0 {
		t.Fatalf("fail to set rate")
	}

	if strings.Compare(getAllRates(accountDB), `{"ETH.ETH":"200"}`) != 0 {
		t.Fatalf("fail to get all rate")
	}
}

// 正常流程
// set 1 rates
func testExchangeRateExecutor1(t *testing.T) {
	// prepare data
	accountDB := getTestAccountDB()
	context := make(map[string]interface{})
	context["refund"] = make(map[uint64]types.RefundInfoList)

	transaction := &types.Transaction{
		Source: "0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443",
		Data:   "{\"ETH.ETH\":\"200\",\"ETH.ETH\":\"200\"}",
	}

	// run
	processor := exchangeRateExecutor{}
	succ, err := processor.Execute(transaction, getTestBlockHeader(), accountDB, context)
	if !succ {
		t.Fatalf(err)
	}
	accountDB.Commit(true)

	rate := accountDB.GetData(common.ExchangeRateAddress, []byte("ETH.ETH"))
	if strings.Compare("200", string(rate)) != 0 {
		t.Fatalf("fail to set rate")
	}

	if strings.Compare(getAllRates(accountDB), `{"ETH.ETH":"200"}`) != 0 {
		t.Fatalf("fail to get all rate")
	}
}

// 正常流程
// set 2 rates
func testExchangeRateExecutor2(t *testing.T) {
	// prepare data
	accountDB := getTestAccountDB()
	context := make(map[string]interface{})
	context["refund"] = make(map[uint64]types.RefundInfoList)

	transaction := &types.Transaction{
		Source: "0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443",
		Data:   "{\"ETH.ETH\":\"200\",\"MIX\":\"1\"}",
	}

	// run
	processor := exchangeRateExecutor{}
	succ, err := processor.Execute(transaction, getTestBlockHeader(), accountDB, context)
	if !succ {
		t.Fatalf(err)
	}
	accountDB.Commit(true)

	rate := accountDB.GetData(common.ExchangeRateAddress, []byte("ETH.ETH"))
	if strings.Compare("200", string(rate)) != 0 {
		t.Fatalf("fail to set rate ETH.ETH")
	}
	rate = accountDB.GetData(common.ExchangeRateAddress, []byte("MIX"))
	if strings.Compare("1", string(rate)) != 0 {
		t.Fatalf("fail to set rate MIX")
	}

	all := getAllRates(accountDB)
	if strings.Compare(all, `{"ETH.ETH":"200","MIX":"1"}`) != 0 {
		t.Fatalf("fail to get all rate. %s", all)
	}
}

// 正常流程
// set 2 rates
// then update 1 rate
func testExchangeRateExecutor3(t *testing.T) {
	// prepare data
	accountDB := getTestAccountDB()
	context := make(map[string]interface{})
	context["refund"] = make(map[uint64]types.RefundInfoList)

	transaction := &types.Transaction{
		Source: "0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443",
		Data:   "{\"ETH.ETH\":\"200\",\"MIX\":\"1\"}",
	}

	// run
	processor := exchangeRateExecutor{}
	succ, err := processor.Execute(transaction, getTestBlockHeader(), accountDB, context)
	if !succ {
		t.Fatalf(err)
	}
	root, _ := accountDB.Commit(true)
	triedb.TrieDB().Commit(root, false)

	rate := accountDB.GetData(common.ExchangeRateAddress, []byte("ETH.ETH"))
	if strings.Compare("200", string(rate)) != 0 {
		t.Fatalf("fail to set rate ETH.ETH")
	}
	rate = accountDB.GetData(common.ExchangeRateAddress, []byte("MIX"))
	if strings.Compare("1", string(rate)) != 0 {
		t.Fatalf("fail to set rate MIX")
	}

	all := getAllRates(accountDB)
	if strings.Compare(all, `{"ETH.ETH":"200","MIX":"1"}`) != 0 {
		t.Fatalf("fail to get all rate. %s", all)
	}

	accountDB, _ = account.NewAccountDB(root, triedb)
	transaction1 := &types.Transaction{
		Source: "0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443",
		Data:   "{\"ETH.ETH\":\"10\"}",
	}
	succ, err = processor.Execute(transaction1, getTestBlockHeader(), accountDB, context)
	if !succ {
		t.Fatalf(err)
	}
	root, _ = accountDB.Commit(true)
	triedb.TrieDB().Commit(root, false)

	accountDB, _ = account.NewAccountDB(root, triedb)
	rate = accountDB.GetData(common.ExchangeRateAddress, []byte("ETH.ETH"))
	if strings.Compare("10", string(rate)) != 0 {
		t.Fatalf("fail to update ETH.ETH")
	}
	rate = accountDB.GetData(common.ExchangeRateAddress, []byte("MIX"))
	if strings.Compare("1", string(rate)) != 0 {
		t.Fatalf("fail to set rate MIX")
	}

	accountDB, _ = account.NewAccountDB(root, triedb)
	all = getAllRates(accountDB)
	if strings.Compare(all, `{"ETH.ETH":"10","MIX":"1"}`) != 0 {
		t.Fatalf("fail to get all rate. %s", all)
	}
}

// 正常流程
// set 2 rates
// then remove 1 rate
func testExchangeRateExecutor4(t *testing.T) {
	// prepare data
	accountDB := getTestAccountDB()
	context := make(map[string]interface{})
	context["refund"] = make(map[uint64]types.RefundInfoList)

	transaction := &types.Transaction{
		Source: "0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443",
		Data:   "{\"ETH.ETH\":\"200\",\"MIX\":\"1\"}",
	}

	// run
	processor := exchangeRateExecutor{}
	succ, err := processor.Execute(transaction, getTestBlockHeader(), accountDB, context)
	if !succ {
		t.Fatalf(err)
	}
	root, _ := accountDB.Commit(true)
	triedb.TrieDB().Commit(root, false)

	rate := accountDB.GetData(common.ExchangeRateAddress, []byte("ETH.ETH"))
	if strings.Compare("200", string(rate)) != 0 {
		t.Fatalf("fail to set rate ETH.ETH")
	}
	rate = accountDB.GetData(common.ExchangeRateAddress, []byte("MIX"))
	if strings.Compare("1", string(rate)) != 0 {
		t.Fatalf("fail to set rate MIX")
	}

	all := getAllRates(accountDB)
	if strings.Compare(all, `{"ETH.ETH":"200","MIX":"1"}`) != 0 {
		t.Fatalf("fail to get all rate. %s", all)
	}

	accountDB, _ = account.NewAccountDB(root, triedb)
	transaction1 := &types.Transaction{
		Source: "0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443",
		Data:   "{\"ETH.ETH\":\"\"}",
	}
	succ, err = processor.Execute(transaction1, getTestBlockHeader(), accountDB, context)
	if !succ {
		t.Fatalf(err)
	}
	root, _ = accountDB.Commit(true)
	triedb.TrieDB().Commit(root, false)

	accountDB, _ = account.NewAccountDB(root, triedb)
	rate = accountDB.GetData(common.ExchangeRateAddress, []byte("ETH.ETH"))
	if service.IsRateExisted("ETH.ETH", accountDB) {
		t.Fatalf("fail to delete ETH.ETH: %s", rate)
	}
	rate = accountDB.GetData(common.ExchangeRateAddress, []byte("MIX"))
	if strings.Compare("1", string(rate)) != 0 {
		t.Fatalf("fail to set rate MIX")
	}

	accountDB, _ = account.NewAccountDB(root, triedb)
	all = getAllRates(accountDB)
	if strings.Compare(all, `{"MIX":"1"}`) != 0 {
		t.Fatalf("fail to get all rate. %s", all)
	}
}

func TestExchangeRateExecutorAll(t *testing.T) {
	fs := []func(*testing.T){
		testExchangeRateExecutor,
		testExchangeRateExecutor1,
		testExchangeRateExecutor2,
		testExchangeRateExecutor3,
		testExchangeRateExecutor4,
	}

	for i, f := range fs {
		name := strconv.Itoa(i)
		setup(name)
		t.Run(name, f)
		teardown(name)
	}
}
