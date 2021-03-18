package service

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"com.tuntun.rocket/node/src/vm"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
)

type contractData struct {
	GasPrice string `json:"gasPrice,omitempty"`
	GasLimit string `json:"gasLimit,omitempty"`

	TransferValue string `json:"transferValue,omitempty"`
	AbiData       string `json:"abiData,omitempty"`
}

type executeResultData struct {
	ContractAddress string `json:"contractAddress,omitempty"`

	ExecuteResult []byte `json:"result,omitempty"`

	Logs []*types.Log `json:"logs,omitempty"`
}

func ExecuteContract(accountdb *account.AccountDB, transaction *types.Transaction, header *types.BlockHeader, context map[string]interface{}) (bool, string) {
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

	var data contractData
	err := json.Unmarshal([]byte(transaction.Data), &data)
	if err != nil {
		txLogger.Errorf("Contract data unmarshal error:%s", err.Error())
		return false, fmt.Sprintf("Contract data error, data: %s", transaction.Data)
	}

	vmCtx.GasPrice, err = utility.StrToBigInt(data.GasPrice)
	if err != nil {
		txLogger.Errorf("Contract GasPrice convert error:%s", err.Error())
		return false, fmt.Sprintf("Contract data GasPrice error, data: %s", data.GasPrice)
	}
	gasLimit, err := strconv.Atoi(data.GasLimit)
	if err != nil {
		txLogger.Errorf("Contract gasLimit convert error:%s", err.Error())
		return false, fmt.Sprintf("Contract data gasLimit error, data: %s", data.GasLimit)
	}
	vmCtx.GasLimit = uint64(gasLimit)

	transferValue, err := transferValueToBigInt(data.TransferValue)
	if err != nil {
		txLogger.Errorf("Contract TransferValue convert error:%s", err.Error())
		return false, fmt.Sprintf("Contract data TransferValue eror, data: %s", data.TransferValue)
	}
	txLogger.Tracef("Execute contract! data: %v,target address:%s", data, transaction.Target)

	vmInstance := vm.NewEVM(vmCtx, accountdb)
	caller := vm.AccountRef(vmCtx.Origin)
	var (
		result          []byte
		leftOverGas     uint64
		logs            []*types.Log
		contractAddress common.Address = common.HexToAddress(transaction.Target)
	)
	if transaction.Target == "" {
		result, contractAddress, leftOverGas, logs, err = vmInstance.Create(caller, common.FromHex(data.AbiData), vmCtx.GasLimit, transferValue)
		txLogger.Tracef("After execute contract create!Contract address:%s, leftOverGas: %d,error:%v", contractAddress.GetHexString(), leftOverGas, err)
	} else {
		result, leftOverGas, logs, err = vmInstance.Call(caller, contractAddress, common.FromHex(data.AbiData), vmCtx.GasLimit, transferValue)
		txLogger.Tracef("After execute contract call! result:%v,leftOverGas: %d,error:%v", result, leftOverGas, err)
	}
	context["logs"] = logs
	if err != nil {
		return false, err.Error()
	}
	returnData := executeResultData{contractAddress.GetHexString(), result, logs}
	json, _ := json.Marshal(returnData)
	return true, string(json)
}

func getBlockHashFn(chain ChainContext) func(n uint64) common.Hash {
	return func(n uint64) common.Hash {
		return chain.GetBlockHash(n)
	}
}

type ChainContext interface {
	GetBlockHash(height uint64) common.Hash
}

func transferValueToBigInt(s string) (*big.Int, error) {
	// 空字符串，默认返回0
	if 0 == len(s) {
		return big.NewInt(0), nil
	}

	target, _, err := big.ParseFloat(s, 10, 18, big.ToNearestEven)
	if err != nil {
		return nil, err
	}

	result := new(big.Int)
	target.Int(result)
	return result, nil
}
