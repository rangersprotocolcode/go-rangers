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
	"com.tuntun.rangers/node/src/vm"
	"fmt"
	"math/big"
)

type minerNodeExecutor struct {
	baseFeeExecutor
	logger log.Logger
}

var ten, _ = utility.StrToBigInt("10")

func (this *minerNodeExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	source := common.FromHex(transaction.Source)

	// check balance for 10rpg
	owner := common.BytesToAddress(source)
	balance := accountdb.GetBalance(owner)
	if ten.Cmp(balance) > 0 {
		msg := fmt.Sprintf("not enough rpg, account: %s, balance: %s", transaction.Source, balance.String())
		this.logger.Errorf(msg)
		return false, msg
	}
	accountdb.SubBalance(owner, ten)

	minerId := service.MinerManagerImpl.GetMinerIdByAccount(source, accountdb)
	if nil == minerId {
		msg := fmt.Sprintf("fail to getMiner by account: %s", transaction.Source)
		this.logger.Errorf(msg)
		return false, msg
	}

	current := service.MinerManagerImpl.GetMiner(minerId, accountdb)
	if nil == current {
		msg := fmt.Sprintf("fail to getMiner, %s", common.ToHex(minerId))
		this.logger.Errorf(msg)
		return false, msg
	}

	// create2
	vmCtx := vm.Context{}
	vmCtx.CanTransfer = vm.CanTransfer
	vmCtx.Transfer = vm.Transfer
	//chainContext := context["chain"].(ChainContext)
	//vmCtx.GetHash = getBlockHashFn(chainContext)
	vmCtx.Origin = common.HexToAddress(transaction.Source)
	vmCtx.Coinbase = common.BytesToAddress(header.Castor)
	vmCtx.BlockNumber = new(big.Int).SetUint64(header.Height)
	vmCtx.Time = new(big.Int).SetUint64(uint64(header.CurTime.Unix()))
	vmCtx.Difficulty = new(big.Int).SetUint64(123)
	vmCtx.GasPrice = defaultGasPrice
	vmCtx.GasLimit = defaultGasLimit

	contractAddress, logs := this.generateContractAddress(vmCtx, accountdb)
	if nil == contractAddress {
		msg := fmt.Sprintf("fail to call create2")
		return false, msg
	}
	context["logs"] = logs
	current.Account = contractAddress
	service.MinerManagerImpl.UpdateMiner(current, accountdb, false)

	msg := fmt.Sprintf("successfully change contract account, from %s to %s", transaction.Source, common.ToHex(contractAddress))
	this.logger.Warnf(msg)
	return true, msg
}

func (this *minerNodeExecutor) generateContractAddress(vmCtx vm.Context, accountdb *account.AccountDB) (types.HexBytes, []*types.Log) {
	vmInstance := vm.NewEVMWithNFT(vmCtx, accountdb, accountdb)
	_, _, logs, err := vmInstance.Call(vm.AccountRef(vmCtx.Origin), common.MainNodeContract(), common.FromHex("0x412a5a6d"), vmCtx.GasLimit, big.NewInt(0))
	if err != nil {
		this.logger.Errorf("fail to call create2, err: %s, length: %d", err, len(logs))
		return nil, nil
	}
	if 4 != len(logs) {
		this.logger.Errorf("fail to call create2, length: %d", len(logs))
		return nil, nil
	}

	log := logs[3]
	if len(log.Data) < 32 {
		this.logger.Errorf("fail to call create2, log.data error: %s", toHex(log.Data))
		return nil, nil
	}
	realAddress := log.Data[12:32]

	return realAddress, logs
}
