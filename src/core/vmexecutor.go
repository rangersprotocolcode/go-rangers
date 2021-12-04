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
	"com.tuntun.rocket/node/src/executor"
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"sort"
	"strings"
	"time"
)

const MaxCastBlockTime = time.Second * 3

type VMExecutor struct {
	accountdb *account.AccountDB
	block     *types.Block
	situation string
	context   map[string]interface{}
	mode      bool
}

func newVMExecutor(accountdb *account.AccountDB, block *types.Block, situation string) *VMExecutor {
	vm := &VMExecutor{
		accountdb: accountdb,
		block:     block,
		situation: situation,
		context:   make(map[string]interface{}),
	}
	vm.context["chain"] = blockChainImpl
	vm.context["situation"] = situation

	return vm
}

func (this *VMExecutor) Execute() (common.Hash, []common.Hash, []*types.Transaction, []*types.Receipt) {
	beginTime := utility.GetTime()

	receipts := make([]*types.Receipt, 0)
	transactions := make([]*types.Transaction, 0)
	evictedTxs := make([]common.Hash, 0)

	this.prepare()

	txs := types.Transactions(this.block.Transactions)
	if 0 != len(txs) && 0 != strings.Compare(this.situation, "casting") {
		sort.Sort(txs)
	}

	for _, transaction := range txs {
		executeTime := utility.GetTime()
		if this.situation == "casting" && executeTime.Sub(beginTime) > MaxCastBlockTime {
			logger.Infof("Cast block execute tx time out! Tx hash:%s ", transaction.Hash.String())
			break
		}
		logger.Debugf("Execute %s, type:%d", transaction.Hash.String(), transaction.Type)

		txExecutor := executor.GetTxExecutor(transaction.Type)
		success := false
		msg := ""

		if txExecutor != nil {
			success, msg = txExecutor.BeforeExecute(transaction, this.block.Header, this.accountdb, this.context)
			if success {
				snapshot := this.accountdb.Snapshot()
				success, msg = txExecutor.Execute(transaction, this.block.Header, this.accountdb, this.context)

				if !success {
					logger.Debugf("Execute failed tx: %s, type: %d, msg: %s", transaction.Hash.String(), transaction.Type, msg)
					evictedTxs = append(evictedTxs, transaction.Hash)
					this.accountdb.RevertToSnapshot(snapshot)
				} else {
					if transaction.Source != "" {
						this.accountdb.IncreaseNonce(common.HexToAddress(transaction.Source))
					}

					logger.Debugf("Execute success, txhash: %s, type: %d", transaction.Hash.String(), transaction.Type)
				}
			}
		}

		transactions = append(transactions, transaction)

		receipt := types.NewReceipt(nil, !success, 0, this.block.Header.Height, msg, transaction.Source, "")
		logs := this.context["logs"]
		if logs != nil {
			receipt.Logs = logs.([]*types.Log)
		}
		contractAddress := this.context["contractAddress"]
		if contractAddress != nil {
			receipt.ContractAddress = contractAddress.(common.Address)
		}
		receipt.TxHash = transaction.Hash
		receipts = append(receipts, receipt)
	}

	this.after()

	state := this.accountdb.IntermediateRoot(true)

	middleware.PerfLogger.Debugf("VMExecutor End. %s height: %d, cost: %v, txs: %d", this.situation, this.block.Header.Height, time.Since(beginTime), len(this.block.Transactions))
	return state, evictedTxs, transactions, receipts
}

func (executor *VMExecutor) validateNonce(accountdb *account.AccountDB, transaction *types.Transaction) bool {
	return true
}

func (executor *VMExecutor) prepare() {
	executor.context["refund"] = make(map[uint64]types.RefundInfoList)
}

func (executor *VMExecutor) after() {
	if 0 == strings.Compare("testing", executor.situation) {
		return
	}

	height := executor.block.Header.Height

	// 计算定时任务（冻结、退款等等）
	service.RefundManagerImpl.Add(types.GetRefundInfo(executor.context), executor.accountdb)

	// 计算出块奖励
	data := service.RewardCalculatorImpl.CalculateReward(height, executor.accountdb, executor.block.Header, executor.situation)
	service.RefundManagerImpl.Add(data, executor.accountdb)

	service.RefundManagerImpl.CheckAndMove(height, executor.accountdb)
}
