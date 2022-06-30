package executor

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
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

// 正常流程
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
