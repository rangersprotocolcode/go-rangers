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
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"com.tuntun.rocket/node/src/vm"
	"encoding/json"
	"fmt"
	"math/big"
)

type contractExecutor struct {
	baseFeeExecutor
	logger log.Logger
}

var (
	gasPrice        = big.NewInt(1)
	gasLimit uint64 = 6000000
)

type executeResultData struct {
	ContractAddress string `json:"contractAddress,omitempty"`

	ExecuteResult string `json:"result,omitempty"`

	Logs []*types.Log `json:"logs,omitempty"`
}

func getBlockHashFn(chain ChainContext) func(n uint64) common.Hash {
	return func(n uint64) common.Hash {
		return chain.GetBlockHash(n)
	}
}

type ChainContext interface {
	GetBlockHash(height uint64) common.Hash
}

func toHex(b []byte) string {
	hex := common.Bytes2Hex(b)
	return "0x" + hex
}

func (this *contractExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	vmCtx := vm.Context{}
	vmCtx.CanTransfer = vm.CanTransfer
	vmCtx.Transfer = vm.Transfer
	chainContext := context["chain"].(ChainContext)
	vmCtx.GetHash = getBlockHashFn(chainContext)

	vmCtx.Origin = common.HexToAddress(transaction.Source)
	vmCtx.Coinbase = common.BytesToAddress(header.Castor)
	vmCtx.BlockNumber = new(big.Int).SetUint64(header.Height)
	vmCtx.Time = new(big.Int).SetUint64(uint64(header.CurTime.Unix()))
	//set constant value
	vmCtx.Difficulty = new(big.Int).SetUint64(123)
	vmCtx.GasPrice = gasPrice
	vmCtx.GasLimit = gasLimit

	var data types.ContractData
	err := json.Unmarshal([]byte(transaction.Data), &data)
	if err != nil {
		this.logger.Errorf("Contract data unmarshal error:%s", err.Error())
		return false, fmt.Sprintf("Contract data error, data: %s", transaction.Data)
	}

	transferValue, err := utility.StrToBigInt(data.TransferValue)
	if err != nil {
		this.logger.Errorf("Contract TransferValue convert error:%s", err.Error())
		return false, fmt.Sprintf("Contract data TransferValue eror, data: %s", data.TransferValue)
	}
	this.logger.Tracef("Execute contract! data: %v,target address:%s", data, transaction.Target)

	var input []byte
	if common.IsProposal005() && (data.AbiData == "" || data.AbiData == "0x0") {
		input = []byte{}
	} else {
		input = common.FromHex(data.AbiData)
	}

	vmInstance := vm.NewEVMWithNFT(vmCtx, accountdb, accountdb)
	caller := vm.AccountRef(vmCtx.Origin)
	var (
		result          []byte
		leftOverGas     uint64
		logs            []*types.Log
		contractAddress = common.HexToAddress(transaction.Target)
	)

	if transaction.Target == "" {
		result, contractAddress, leftOverGas, logs, err = vmInstance.Create(caller, input, vmCtx.GasLimit, transferValue)
		context["contractAddress"] = contractAddress

		this.logger.Tracef("After execute contract create!Contract address:%s, leftOverGas: %d,error:%v", contractAddress.GetHexString(), leftOverGas, err)
	} else {
		if common.IsProposal007() {
			nonce := accountdb.GetNonce(caller.Address())
			accountdb.SetNonce(caller.Address(), nonce+1)
		}
		result, leftOverGas, logs, err = vmInstance.Call(caller, contractAddress, input, vmCtx.GasLimit, transferValue)

		this.logger.Tracef("After execute contract call[%s]! result:%v,leftOverGas: %d,error:%v", transaction.Hash.String(), result, leftOverGas, err)
		if transaction.Hash.String() == "0xe17882a7e7e7239573cd7b9da08541841418238933bb80d3e4fa52406353e4b3" && logs != nil {
			this.logger.Tracef("logs:")
			for _, log := range logs {
				this.logger.Tracef("tx hash:%s,block hash:%s,address:%s,block num:%d,log index:%d,tx index:%d,removed:%v,data:%s,topics:%v", log.TxHash.String(), log.BlockHash.String(), log.Address.String(), log.BlockNumber, log.Index, log.TxIndex, log.Removed, common.ToHex(log.Data), log.Topics)
			}
		}
	}

	context["logs"] = logs
	if err != nil {
		return false, err.Error()
	}

	returnData := executeResultData{contractAddress.GetHexString(), toHex(result), logs}
	json, _ := json.Marshal(returnData)
	return true, string(json)
}
