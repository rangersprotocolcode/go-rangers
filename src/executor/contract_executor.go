package executor

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"com.tuntun.rocket/node/src/vm"
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

var (
	gasPrice                    = big.NewInt(1)
	gasLimit             uint64 = 6000000
	p011GasPrice                = big.NewInt(1000000000)
	p011GasLimit         uint64 = 7000000
	ErrInsufficientFunds        = errors.New("insufficient funds for gas * price + value")
	ErrInsufficientGas          = errors.New("insufficient gas for data")
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

	var exeGasLimit uint64
	if common.IsProposal011() {
		if data.GasLimit == "" {
			exeGasLimit = p011GasLimit
		} else {
			exeGasLimit, err = strconv.ParseUint(data.GasLimit, 10, 64)
			if err != nil {
				this.logger.Errorf("Contract gasLimit convert error:%s", err.Error())
				return false, fmt.Sprintf("Contract data gasLimit eror, data: %s", data.GasLimit)
			}
		}
		coinLimit := new(big.Int).Mul(new(big.Int).SetUint64(exeGasLimit), p011GasPrice)
		expectBalance := new(big.Int).Add(coinLimit, transferValue)
		if !vm.CanTransfer(accountdb, vmCtx.Origin, expectBalance) {
			this.logger.Errorf("Contract insufficient funds:%s,need:%s", err.Error(), expectBalance)
			return false, ErrInsufficientFunds.Error()
		}

		dataGas, err := CalDataGas(input)
		if err != nil {
			this.logger.Errorf("Contract data gas cal error:%s", err.Error())
			return false, err.Error()
		}
		if exeGasLimit < dataGas {
			this.logger.Errorf("Contract  insufficient gas,given:%d,dataGas:%d", exeGasLimit, dataGas)
			return false, ErrInsufficientGas.Error()
		}
		vmCtx.GasLimit = exeGasLimit - dataGas
	}

	vmInstance := vm.NewEVMWithNFT(vmCtx, accountdb, accountdb)
	caller := vm.AccountRef(vmCtx.Origin)
	var (
		result          []byte
		leftOverGas     uint64
		logs            []*types.Log
		contractAddress common.Address = common.HexToAddress(transaction.Target)
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
		this.logger.Tracef("After execute contract call! result:%v,leftOverGas: %d,error:%v", result, leftOverGas, err)
	}
	context["logs"] = logs
	if common.IsProposal011() {
		gasUsed := exeGasLimit - leftOverGas
		coinUsed := new(big.Int).Mul(new(big.Int).SetUint64(gasUsed), p011GasPrice)
		accountdb.SubBalance(common.HexToAddress(transaction.Source), coinUsed)
		accountdb.AddBalance(common.FeeAccount, coinUsed)
	}
	if err != nil {
		return false, err.Error()
	}
	returnData := executeResultData{contractAddress.GetHexString(), toHex(result), logs}
	json, _ := json.Marshal(returnData)
	return true, string(json)
}

func CalDataGas(data []byte) (uint64, error) {
	var dataGas uint64 = 0
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
		if (math.MaxUint64)/nonZeroGas < nz {
			return 0, vm.ErrGasUintOverflow
		}
		dataGas += nz * nonZeroGas

		z := uint64(len(data)) - nz
		if (math.MaxUint64)/vm.TxDataZeroGas < z {
			return 0, vm.ErrGasUintOverflow
		}
		dataGas += z * vm.TxDataZeroGas
	}
	return dataGas, nil
}
