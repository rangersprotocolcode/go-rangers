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

package core

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/executor"
	"com.tuntun.rangers/node/src/middleware"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/service"
	"com.tuntun.rangers/node/src/storage/account"
	"com.tuntun.rangers/node/src/utility"
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
	if situation == "fork" {
		vm.context["chain"] = SyncProcessor
	}
	return vm
}

func (this *VMExecutor) Execute() (common.Hash, []common.Hash, []*types.Transaction, []*types.Receipt) {
	var beginTime time.Time
	if this.situation == "casting" {
		beginTime = utility.GetTime()
	}

	receipts := make([]*types.Receipt, 0)
	transactions := make([]*types.Transaction, 0)
	evictedTxs := make([]common.Hash, 0)

	this.prepare()
	txs := types.Transactions(this.block.Transactions)
	if 0 != len(txs) && this.situation != "casting" {
		sort.Sort(txs)
	}

	i := 0
	for _, transaction := range txs {
		if 0 == transaction.Type {
			continue
		}

		if common.IsProposal013() {
			this.accountdb.Prepare(transaction.Hash, common.Hash{}, i)
		}

		if this.situation == "casting" && utility.GetTime().Sub(beginTime) > MaxCastBlockTime {
			logger.Infof("Cast block execute tx time out! Tx hash:%s ", transaction.Hash.String())
			break
		}
		logger.Debugf("Execute %s, type:%d", transaction.Hash.String(), transaction.Type)

		if common.IsProposal006() && !common.IsProposal007() {
			this.accountdb.IncreaseNonce(common.HexToAddress(transaction.Source))
		}

		txExecutor := executor.GetTxExecutor(transaction.Type)
		success := false
		addAble := true
		msg := ""

		if txExecutor != nil {
			success, addAble, msg = txExecutor.BeforeExecute(transaction, this.block.Header, this.accountdb, this.context)
			if common.IsProposal018() && !addAble {
				evictedTxs = append(evictedTxs, transaction.Hash)
				logger.Infof("Tx not addAble,skip.Hash:%s,msg:%s", transaction.Hash.String(), msg)
				continue
			}

			if success {
				snapshot := this.accountdb.Snapshot()
				success, msg = txExecutor.Execute(transaction, this.block.Header, this.accountdb, this.context)

				if !success {
					logger.Debugf("Execute failed tx: %s, type: %d, msg: %s", transaction.Hash.String(), transaction.Type, msg)
					if !common.IsProposal018() {
						evictedTxs = append(evictedTxs, transaction.Hash)
					}
					this.accountdb.RevertToSnapshot(snapshot)
				} else {
					if transaction.Source != "" {
						if !common.IsProposal006() {
							this.accountdb.IncreaseNonce(common.HexToAddress(transaction.Source))
						}
					}

					logger.Debugf("Execute success, txhash: %s, type: %d", transaction.Hash.String(), transaction.Type)
				}
			}
			if common.IsProposal007() {
				if !(types.IsContractTx(transaction.Type) && success) {
					nonce := this.accountdb.GetNonce(common.HexToAddress(transaction.Source))
					this.accountdb.SetNonce(common.HexToAddress(transaction.Source), nonce+1)
				}
			}
		}

		transactions = append(transactions, transaction)

		receipt := types.NewReceipt(nil, !success, 0, this.block.Header.Height, msg, transaction.Source, "")
		if common.IsProposal013() {
			receipt.Logs = this.accountdb.GetLogs(transaction.Hash)
		} else {
			logs := this.context["logs"]
			if logs != nil {
				receipt.Logs = logs.([]*types.Log)
			}
		}
		if this.context["logs"] != nil {
			delete(this.context, "logs")
		}

		contractAddress := this.context["contractAddress"]
		if contractAddress != nil {
			delete(this.context, "contractAddress")
			receipt.ContractAddress = contractAddress.(common.Address)
		}

		gasUsed := this.context["gasUsed"]
		if gasUsed != nil && common.IsProposal015() {
			receipt.GasUsed = gasUsed.(uint64)
		}
		receipt.TxHash = transaction.Hash
		receipts = append(receipts, receipt)
		i++
	}

	//only for robin
	if this.block.Header.Height == common.LocalChainConfig.Proposal010Block {
		removeUnusedValidator(this.accountdb)
	}
	if this.block.Header.Height == common.LocalChainConfig.Proposal019Block {
		removeUnusedValidator1(this.accountdb)
	}
	this.after()

	state := this.accountdb.IntermediateRoot(true)

	middleware.PerfLogger.Debugf("VMExecutor End. %s height: %d, cost: %v, txs: %d", this.situation, this.block.Header.Height, utility.GetTime().Sub(beginTime), len(this.block.Transactions))
	return state, evictedTxs, transactions, receipts
}

func (executor *VMExecutor) prepare() {
	executor.context["refund"] = make(map[uint64]types.RefundInfoList)
}

func (executor *VMExecutor) after() {
	if 0 == strings.Compare("testing", executor.situation) {
		return
	}

	executor.calcDifficulty()

	service.RefundManagerImpl.Add(types.GetRefundInfo(executor.context), executor.accountdb)

	height := executor.block.Header.Height

	if common.IsSub() {
		executor.calcSubReward()
	} else {
		data := service.RewardCalculatorImpl.CalculateReward(height, executor.accountdb, executor.block.Header, executor.situation)
		service.RefundManagerImpl.Add(data, executor.accountdb)
	}

	service.RefundManagerImpl.CheckAndMove(height, executor.accountdb)
	if common.LocalChainConfig.Proposal004Block == height {
		service.RefundManagerImpl.CheckAndMove(0, executor.accountdb)
	}

}

func (executor *VMExecutor) calcDifficulty() {
	height := executor.block.Header.Height
	if height < common.LocalChainConfig.Proposal025Block {
		return
	}

	// calculate difficulty
	data := executor.accountdb.GetData(common.DifficultyAddress, executor.block.Header.Castor)
	value := uint64(1)
	if 0 != len(data) {
		value = utility.ByteToUInt64(data) + 1
	} else {
		// new proposal found
		totalMinersBytes := executor.accountdb.GetData(common.DifficultyAddress, common.TotalWorkingMiners)
		totalMiners := utility.ByteToUInt64(totalMinersBytes) + 1
		executor.accountdb.SetData(common.DifficultyAddress, common.TotalWorkingMiners, utility.UInt64ToByte(totalMiners))
		logger.Infof("height: %d, add difficulty, new proporal %s, %d", height, common.ToHex(executor.block.Header.Castor), totalMiners)
	}
	executor.accountdb.SetData(common.DifficultyAddress, executor.block.Header.Castor, utility.UInt64ToByte(value))
	logger.Infof("height: %d, add difficulty, %s, %d", height, common.ToHex(executor.block.Header.Castor), value)

	if height < common.LocalChainConfig.Proposal025Block+common.GetRewardBlocks() {
		return
	}

	targetHeight := height - common.GetRewardBlocks()
	header := blockChainImpl.QueryBlockHeaderByHeight(targetHeight, true)
	if nil == header {
		// never occur
		logger.Errorf("fail to get header, %d", targetHeight)
		return
	}

	data = executor.accountdb.GetData(common.DifficultyAddress, header.Castor)
	value = utility.ByteToUInt64(data) - 1
	executor.accountdb.SetData(common.DifficultyAddress, header.Castor, utility.UInt64ToByte(value))
	logger.Infof("height: %d, minus difficulty, %s, %d", height, common.ToHex(header.Castor), value)

	// lost proposal
	if value == 0 {
		totalMinersBytes := executor.accountdb.GetData(common.DifficultyAddress, common.TotalWorkingMiners)
		totalMiners := utility.ByteToUInt64(totalMinersBytes) - 1
		executor.accountdb.SetData(common.DifficultyAddress, common.TotalWorkingMiners, utility.UInt64ToByte(totalMiners))
		logger.Infof("height: %d, minus difficulty, lost proporal %s, %d", height, common.ToHex(header.Castor), totalMiners)
	}
}

func removeUnusedValidator(accountdb *account.AccountDB) {
	var unusedValidatorList = []string{
		"0x01820ed1304f0484e252ddac1ab5a1e6e16e5ebf89f022c092e8decd69e088e6",
		"0x18b97514b118dda8d8a30f16fc6de49ebeac849359e6ffd17b5299a82112eedd",
		"0x008825f3184b9f6f0935830c7738d1da3f9dc2a055f99c8c06176f36f5951686",
		"0xb0951738af6ad10c10a2406eb46c4c3bd1df795eac566fb6cf9cecc15dfb388a",
		"0xaebb5bc7af7f6522f164b74f21d901faef12a15f8accd5f7bb3417107bdaa295",
		"0xddb0792bdf0bbd75ba85ab4343a38392b539c7e74c1b17de43e8768328cc31f4",
		"0xfae2464767a076614cff1c854504587d8a186fd1332eb28ab4146a89edad0dca",
		"0xe14a2ee33f83aa8af7f2cafe31bca30ab3b61ef00bda6931f03310e11f5e6acd",
		"0x1d5a3badc41060d4928e5117518d8d9d5fdbbc535e4660ad4b37b3593e717634",
	}

	for _, minerIdStr := range unusedValidatorList {
		minerId := common.FromHex(minerIdStr)
		miner := service.MinerManagerImpl.GetMinerById(minerId, common.MinerTypeValidator, accountdb)
		if miner == nil {
			continue
		}
		service.MinerManagerImpl.RemoveMiner(minerId, miner.Account[:], miner.Type, accountdb, 0)
	}
}

func removeUnusedValidator1(accountdb *account.AccountDB) {
	var whileList = make(map[string]byte, 0)
	whileList["0x5437f9dd7171db9d04a8347dca5bf2b7789081631d79d2d7882c1774d2f4d123"] = 0
	whileList["0x2a17671c5a32175335fa098951ba50a9b4730aea7ecee86df6536297900f5b77"] = 0
	whileList["0xb1979dd362353f0b59dff76cb223d5660a024db628257693f5470dec18c93160"] = 0
	whileList["0x7f88b4f2d36a83640ce5d782a0a20cc2b233de3df2d8a358bf0e7b29e9586a12"] = 0
	whileList["0xb715ee4bab1d5b63dc94bdd4841fce7b37310b3514c46357ca83475a248ec9b5"] = 0
	whileList["0xca5503987bf640e724a3b5f1cb00fd76138aa872ba1a0b339d83c0aa078a92f3"] = 0
	whileList["0xd6b97afd9fee6e22d37b7c0946bb6ed509b048a98c26d0611e7d4fa014f96c01"] = 0

	service.MinerManagerImpl.RemoveUnusedValidator(accountdb, whileList)
}
