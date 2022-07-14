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
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"com.tuntun.rocket/node/src/vm"
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
	vmCtx.GasPrice = gasPrice
	vmCtx.GasLimit = gasLimit

	nonce := new(big.Int).SetBytes(transaction.Hash.Bytes()[:20]).String()
	contractAddress := this.generateContractAddress(vmCtx, nonce, owner, current.Stake, accountdb)
	if nil == contractAddress {
		msg := fmt.Sprintf("fail to call create2, nonce %s", nonce)
		return false, msg
	}

	current.Account = contractAddress
	service.MinerManagerImpl.UpdateMiner(current, accountdb, false)

	msg := fmt.Sprintf("successfully change contract account, from %s to %s", transaction.Source, common.ToHex(contractAddress))
	this.logger.Warnf(msg)
	return true, msg
}

//
func (this *minerNodeExecutor) generateContractAddress(vmCtx vm.Context, nonce string, owner common.Address, stake uint64, accountdb *account.AccountDB) types.HexBytes {
	vmInstance := vm.NewEVMWithNFT(vmCtx, accountdb, accountdb)
	_, _, logs, err := vmInstance.Call(vm.AccountRef(vmCtx.Origin), common.MainNodeContract(), common.FromHex("0x412a5a6d"), vmCtx.GasLimit, big.NewInt(0))
	if err != nil{
		this.logger.Errorf("fail to call create2, err: %s, length: %d", err, len(logs) )
		return nil
	}

	log := logs[2]
	realAddress := log.Data[:32]
	return realAddress
}
