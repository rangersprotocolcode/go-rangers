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
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
)

type contractExecutor struct {
	baseFeeExecutor
	logger log.Logger
}

const (
	defaultGasLimit     uint64 = 6000000
	p017defaultGasLimit uint64 = 30000000
)

var (
	defaultGasPrice      = big.NewInt(1000000000)
	ErrInsufficientFunds = errors.New("insufficient funds for gas * price + value")
	ErrIntrinsicGas      = errors.New("intrinsic gas too low")
)

type executeResultData struct {
	ContractAddress string `json:"contractAddress,omitempty"`

	ExecuteResult string `json:"result,omitempty"`

	Logs []*types.Log `json:"logs,omitempty"`
}

type ContractRawData struct {
	GasLimit      uint64
	TransferValue *big.Int
	AbiData       []byte
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

func (this *contractExecutor) BeforeExecute(tx *types.Transaction, header *types.BlockHeader, accountDB *account.AccountDB, context map[string]interface{}) (bool, string) {
	err := service.GetTransactionPool().ProcessFee(*tx, accountDB)
	if err != nil {
		return false, err.Error()
	}

	raw, errMessage := this.decodeContractData(tx.Data)
	if errMessage != "" {
		return false, errMessage
	}
	context["contractData"] = raw
	//check if balance > (gasLimit * gasPrice) + transfer value
	if common.IsProposal015() {
		balance := accountDB.GetBalance(common.HexToAddress(tx.Source))
		gasFee := new(big.Int).Mul(new(big.Int).SetUint64(raw.GasLimit), defaultGasPrice)
		if balance.Cmp(new(big.Int).Add(gasFee, raw.TransferValue)) < 0 {
			this.logger.Errorf("[ContractExecutor]insufficient funds:%s,balance:%s,gasFess:%s,transferValue:%s,", tx.Hash.String(), balance.String(), gasFee.String(), raw.TransferValue.String())
			return false, ErrInsufficientFunds.Error()
		}
		this.logger.Tracef("[ContractExecutor]pre check passed:%s,balance:%s,gasFess:%s,transferValue:%s,", tx.Hash.String(), balance.String(), gasFee.String(), raw.TransferValue.String())
	}
	return true, ""
}

func (this *contractExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	// check chain status if it is subChain
	if common.IsSub() && transaction.Target == common.WhitelistForCreate {
		status := service.GetSubChainStatus(accountdb)
		if 2 != status {
			return false, fmt.Sprintf("cannot call contract , status: %d", status)
		}
	}

	var contractCreation = false
	if transaction.Target == "" {
		contractCreation = true
	}
	contractRawData := context["contractData"].(*ContractRawData)
	gasLimit := contractRawData.GasLimit
	transferValue := contractRawData.TransferValue
	input := contractRawData.AbiData
	var err error
	var intrinsicGas uint64
	if common.IsProposal015() {
		intrinsicGas, err = IntrinsicGas(input, contractCreation)
		if err != nil {
			this.logger.Errorf("[ContractExecutor]IntrinsicGas error:%s", err.Error())
			return false, err.Error()
		}
		if contractRawData.GasLimit < intrinsicGas {
			this.logger.Errorf("[ContractExecutor]gas limit too low,gas limit:%d,intrinsic gas:%d", contractRawData.GasLimit, intrinsicGas)
			return false, ErrIntrinsicGas.Error()
		}
	}

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
	vmCtx.GasPrice = defaultGasPrice
	vmCtx.GasLimit = defaultGasLimit
	if common.IsProposal015() {
		if common.IsProposal017() && gasLimit > p017defaultGasLimit {
			gasLimit = p017defaultGasLimit
		}
		vmCtx.GasLimit = gasLimit - intrinsicGas
	}

	vmInstance := vm.NewEVMWithNFT(vmCtx, accountdb, accountdb)
	caller := vm.AccountRef(vmCtx.Origin)
	var (
		result          []byte
		leftOverGas     uint64
		logs            []*types.Log
		contractAddress = common.HexToAddress(transaction.Target)
	)

	this.logger.Debugf("before vm instance,intrinsicGas:%d,gasLimit:%d", intrinsicGas, vmCtx.GasLimit)
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
	}

	context["logs"] = logs
	if common.IsProposal015() {
		gasUsed := gasLimit - leftOverGas
		gasFeeUsed := new(big.Int).Mul(new(big.Int).SetUint64(gasUsed), defaultGasPrice)
		accountdb.SubBalance(common.HexToAddress(transaction.Source), gasFeeUsed)
		accountdb.AddBalance(common.FeeAccount, gasFeeUsed)
		context["gasUsed"] = gasUsed
	}
	if err != nil {
		return false, err.Error()
	}

	returnData := executeResultData{contractAddress.GetHexString(), toHex(result), logs}
	json, _ := json.Marshal(returnData)
	return true, string(json)
}

// IntrinsicGas computes the 'intrinsic gas' for a message with the given data.
func IntrinsicGas(data []byte, contractCreation bool) (uint64, error) {
	// Set the starting gas for the raw transaction
	var gas uint64
	if contractCreation {
		gas = vm.TxGasContractCreation
	} else {
		gas = vm.TxGas
	}
	// Bump the required gas by the amount of transactional data
	if len(data) > 0 {
		// Zero and non-zero bytes are priced differently
		var nz uint64
		for _, byt := range data {
			if byt != 0 {
				nz++
			}
		}
		// Make sure we don't exceed uint64 for all data combinations
		nonZeroGas := vm.TxDataNonZeroGasEIP2028

		if (math.MaxUint64-gas)/nonZeroGas < nz {
			return 0, vm.ErrGasUintOverflow
		}
		gas += nz * nonZeroGas

		z := uint64(len(data)) - nz
		if (math.MaxUint64-gas)/vm.TxDataZeroGas < z {
			return 0, vm.ErrGasUintOverflow
		}
		gas += z * vm.TxDataZeroGas
	}
	return gas, nil
}

func (this *contractExecutor) decodeContractData(txData string) (*ContractRawData, string) {
	var data types.ContractData
	err := json.Unmarshal([]byte(txData), &data)
	if err != nil {
		this.logger.Errorf("Contract data unmarshal error:%s", err.Error())
		return nil, fmt.Sprintf("Contract data error, data: %s", txData)
	}

	var rawGasLimit uint64
	if data.GasLimit == "" || data.GasLimit == "0" {
		rawGasLimit = defaultGasLimit
		if common.IsProposal017() {
			rawGasLimit = p017defaultGasLimit
		}
	} else {
		rawGasLimit, err = strconv.ParseUint(data.GasLimit, 10, 64)
		if err != nil {
			this.logger.Errorf("Contract gasLimit convert error:%s", err.Error())
			return nil, fmt.Sprintf("Contract data gasLimit eror, data: %s", data.GasLimit)
		}
	}

	transferValue, err := utility.StrToBigInt(data.TransferValue)
	if err != nil {
		this.logger.Errorf("Contract TransferValue convert error:%s", err.Error())
		return nil, fmt.Sprintf("Contract data TransferValue eror, data: %s", data.TransferValue)
	}

	var input []byte
	if common.IsProposal005() && (data.AbiData == "" || data.AbiData == "0x0") {
		input = []byte{}
	} else {
		input = common.FromHex(data.AbiData)
	}
	raw := &ContractRawData{rawGasLimit, transferValue, input}
	return raw, ""
}
