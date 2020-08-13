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
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"strings"
	"time"
)

var executors map[int32]executor

type executor interface {
	BeforeExecute(tx *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string)
	Execute(tx *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string)
}

type baseFeeExecutor struct {
}

func (this *baseFeeExecutor) BeforeExecute(tx *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	err := service.GetTransactionPool().ProcessFee(*tx, accountdb)
	if err == nil {
		return true, ""
	}
	return false, err.Error()
}

func initExecutors() {
	executors = make(map[int32]executor, 21)

	executors[types.TransactionTypeOperatorEvent] = &operatorExecutor{}
	executors[types.TransactionTypeWithdraw] = &withdrawExecutor{}
	executors[types.TransactionTypeCoinDepositAck] = &coinDepositExecutor{}
	executors[types.TransactionTypeFTDepositAck] = &ftDepositExecutor{}
	executors[types.TransactionTypeNFTDepositAck] = &nftDepositExecutor{}
	executors[types.TransactionTypeMinerApply] = &minerApplyExecutor{}
	executors[types.TransactionTypeMinerAdd] = &minerAddExecutor{}
	executors[types.TransactionTypeMinerRefund] = &minerRefundExecutor{}

	executors[types.TransactionTypePublishFT] = &ftExecutor{}
	executors[types.TransactionTypePublishNFTSet] = &ftExecutor{}
	executors[types.TransactionTypeMintFT] = &ftExecutor{}
	executors[types.TransactionTypeMintNFT] = &ftExecutor{}
	executors[types.TransactionTypeShuttleNFT] = &ftExecutor{}
	executors[types.TransactionTypeUpdateNFT] = &ftExecutor{}
	executors[types.TransactionTypeApproveNFT] = &ftExecutor{}
	executors[types.TransactionTypeRevokeNFT] = &ftExecutor{}

	executors[types.TransactionTypeAddStateMachine] = &stmExecutor{}
	executors[types.TransactionTypeUpdateStorage] = &stmExecutor{}
	executors[types.TransactionTypeStartSTM] = &stmExecutor{}
	executors[types.TransactionTypeStopSTM] = &stmExecutor{}
	executors[types.TransactionTypeUpgradeSTM] = &stmExecutor{}
	executors[types.TransactionTypeQuitSTM] = &stmExecutor{}
	executors[types.TransactionTypeImportNFT] = &stmExecutor{}

	executors[types.TransactionTypeSetExchangeRate] = &exchangeRateExecutor{}
}

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

	return vm
}

func (this *VMExecutor) Execute() (common.Hash, []common.Hash, []*types.Transaction, []*types.Receipt) {
	beginTime := utility.GetTime()

	receipts := make([]*types.Receipt, 0)
	transactions := make([]*types.Transaction, 0)
	evictedTxs := make([]common.Hash, 0)

	this.prepare()

	for _, transaction := range this.block.Transactions {
		executeTime := utility.GetTime()
		if this.situation == "casting" && executeTime.Sub(beginTime) > MaxCastBlockTime {
			logger.Infof("Cast block execute tx time out! Tx hash:%s ", transaction.Hash.String())
			break
		}
		logger.Debugf("Execute %s, type:%d", transaction.Hash.String(), transaction.Type)

		executor := executors[transaction.Type]
		success := false
		msg := ""

		if executor != nil {
			success, msg = executor.BeforeExecute(transaction, this.block.Header, this.accountdb, this.context)
			if success {
				snapshot := this.accountdb.Snapshot()
				success, msg = executor.Execute(transaction, this.block.Header, this.accountdb, this.context)

				if !success {
					logger.Debugf("Execute failed tx: %s, type: %d, msg: %s", transaction.Hash.String(), transaction.Type, msg)
					evictedTxs = append(evictedTxs, transaction.Hash)
					this.accountdb.RevertToSnapshot(snapshot)
				} else {
					if transaction.Source != "" {
						this.accountdb.SetNonce(common.HexToAddress(transaction.Source), transaction.Nonce)
					}

					logger.Debugf("Execute success, txhash: %s, type: %d", transaction.Hash.String(), transaction.Type)
				}
			}
		}

		transactions = append(transactions, transaction)
		receipt := types.NewReceipt(nil, !success, 0, this.block.Header.Height, msg, transaction.Source)
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
	executor.context["refund"] = make(map[uint64]RefundInfoList)
}

func (executor *VMExecutor) after() {
	if 0 == strings.Compare("testing", executor.situation) {
		return
	}

	height := executor.block.Header.Height

	// 计算定时任务（冻结、退款等等）
	RefundManagerImpl.Add(getRefundInfo(executor.context), executor.accountdb)
	RefundManagerImpl.CheckAndMove(height, executor.accountdb)

	// 计算出块奖励
	RewardCalculatorImpl.CalculateReward(height, executor.accountdb)
}
func getRefundInfo(context map[string]interface{}) map[uint64]RefundInfoList {
	raw := context["refund"]
	return raw.(map[uint64]RefundInfoList)
}
