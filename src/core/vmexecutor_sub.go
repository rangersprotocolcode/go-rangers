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

package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/vm"
	"math/big"
)

var (
	callerAddress    = common.HexToAddress("0x1111111111111111111111111111111111111111")
	callerRPGAddress = common.HexToAddress("0x0")
)

const padding = "0000000000000000000000000000000000000000000000000000000000000060"

func (executor *VMExecutor) calcSubReward() {
	header := executor.block.Header
	proposals, validators := service.MinerManagerImpl.GetAllMinerIdAndAccount(header.Height, executor.accountdb)

	// get validator group
	groupId := header.GroupId
	var group *types.Group
	if executor.situation != "fork" {
		group = groupChainImpl.GetGroupById(groupId)
	} else {
		group = SyncProcessor.GetGroupById(groupId)
	}
	if group == nil {
		rewardLog.Errorf("fail to get group. id: %v", groupId)
		return
	}

	executor.calcSubCoinReward(proposals, validators, group.Members)
	executor.calcSubRPGReward(proposals, validators, group.Members)
}

func (executor *VMExecutor) calcSubCoinReward(proposals, validators map[string]common.Address, member [][]byte) {
	header := executor.block.Header

	vmCtx := vm.Context{}
	vmCtx.CanTransfer = vm.CanTransfer
	vmCtx.Transfer = transfer
	vmCtx.GetHash = func(uint64) common.Hash { return emptyHash }
	vmCtx.Origin = callerAddress
	vmCtx.Coinbase = common.BytesToAddress(header.Castor)
	vmCtx.BlockNumber = new(big.Int).SetUint64(header.Height)
	vmCtx.Time = new(big.Int).SetUint64(uint64(header.CurTime.Unix()))
	vmCtx.GasPrice = big.NewInt(1)
	vmCtx.GasLimit = 30000000

	vmInstance := vm.NewEVMWithNFT(vmCtx, executor.accountdb, executor.accountdb)
	caller := vm.AccountRef(vmCtx.Origin)

	code, done := executor.generateCode(proposals, validators, member, header)
	if done {
		return
	}

	codeBytes := common.FromHex(code)
	_, _, _, err := vmInstance.Call(caller, common.EconomyContract, codeBytes, vmCtx.GasLimit, big.NewInt(0))
	if err != nil {
		rewardLog.Errorf("calcSubCoinReward error: %s. code: %s", err.Error(), code)
	}
}

func (executor *VMExecutor) generateCode(proposals, validators map[string]common.Address, members [][]byte, header *types.BlockHeader) (string, bool) {
	code := "0x7822b9ac" + common.GenerateCallDataAddress(proposals[common.ToHex(header.Castor)]) + padding + common.GenerateCallDataUint(uint64(4+len(proposals))*32)
	code += common.GenerateCallDataUint(uint64(len(proposals)))
	for _, addr := range proposals {
		code += common.GenerateCallDataAddress(addr)
	}

	code += common.GenerateCallDataUint(uint64(len(members)))
	for _, member := range members {
		code += common.GenerateCallDataAddress(validators[common.ToHex(member)])
	}

	return code, false
}

func (executor *VMExecutor) calcSubRPGReward(proposals, validators map[string]common.Address, members [][]byte) {
	header := executor.block.Header

	vmCtx := vm.Context{}
	vmCtx.CanTransfer = vm.CanTransfer
	vmCtx.Transfer = transfer
	vmCtx.GetHash = func(uint64) common.Hash { return emptyHash }
	//vmCtx.Origin = callerRPGAddress
	vmCtx.Coinbase = common.BytesToAddress(header.Castor)
	vmCtx.BlockNumber = new(big.Int).SetUint64(header.Height)
	vmCtx.Time = new(big.Int).SetUint64(uint64(header.CurTime.Unix()))
	vmCtx.GasPrice = big.NewInt(1)
	vmCtx.GasLimit = 30000000

	vmInstance := vm.NewEVMWithNFT(vmCtx, executor.accountdb, executor.accountdb)
	caller := vm.AccountRef(vmCtx.Origin)

	code, done := executor.generateRPGCode(proposals, validators, members, header)
	if done {
		return
	}

	codeBytes := common.FromHex(code)
	_, _, _, err := vmInstance.Call(caller, common.EconomyContract, codeBytes, vmCtx.GasLimit, big.NewInt(0))
	if err != nil {
		rewardLog.Errorf("calcSubCoinReward error: %s. code: %s", err.Error(), code)
	}
}

func (executor *VMExecutor) generateRPGCode(proposals, validators map[string]common.Address, members [][]byte, header *types.BlockHeader) (string, bool) {

	code := "0x87f614e10000000000000000000000000000000000000000000000000000000000000020"
	code += common.GenerateCallDataUint(uint64(len(members) + 1))
	code += common.GenerateCallDataAddress(proposals[common.ToHex(header.Castor)])

	for _, member := range members {
		code += common.GenerateCallDataAddress(validators[common.ToHex(member)])
	}

	return code, false
}

// Transfer subtracts amount from sender and adds amount to recipient using the given Db
func transfer(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {
	if nil == amount || 0 == amount.Sign() {
		return
	}

	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}
