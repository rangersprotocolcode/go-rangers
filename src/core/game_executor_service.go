package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/mysql"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"com.tuntun.rocket/node/src/vm"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
)

type queryLogData struct {
	FromBlock uint64 `json:"fromBlock,omitempty"`
	ToBlock   uint64 `json:"toBlock,omitempty"`

	Address []string   `json:"address,omitempty"`
	Topics  [][]string `json:"topics,omitempty"`
}

type queryBlockData struct {
	Height string `json:"height,omitempty"`
	Hash   string `json:"hash,omitempty"`

	ReturnTransactionObjects bool `json:"returnTransactionObjects,omitempty"`
}

type callVMData struct {
	Height string `json:"height,omitempty"`
	Hash   string `json:"hash,omitempty"`

	From     string `json:"from"`
	To       string `json:"to"`
	Gas      uint64 `json:"gas"`
	GasPrice string `json:"gasPrice"`
	Value    string `json:"value"`
	Data     string `json:"data"`
}

func getBlock(height string, hash string, returnTxObjs bool) string {
	block := getBlockByHashOrHeight(height, hash)
	if block == nil {
		return ""
	}

	blockDetail := ConvertBlockWithTxDetail(block)
	var result []byte
	if returnTxObjs {
		result, _ = json.Marshal(blockDetail)
	} else {
		result, _ = json.Marshal(blockDetail.RPCBlock)
	}
	return string(result)
}

func getBlockNumber() string {
	block := GetBlockChain().TopBlock()
	if block == nil {
		return ""
	}
	return strconv.FormatUint(block.Height, 10)
}

func getTransactionCount(height string, hash string) string {
	block := getBlockByHashOrHeight(height, hash)
	if block == nil {
		return ""
	}
	return strconv.Itoa(len(block.Transactions))
}

func getTransactionFromBlock(height string, hash string, index string) string {
	block := getBlockByHashOrHeight(height, hash)
	if block == nil {
		return ""
	}
	indexInt, err := strconv.Atoi(index)
	if err != nil {
		return ""
	}
	if indexInt >= len(block.Transactions) {
		return ""
	}
	tx := block.Transactions[indexInt]
	txDetail := ConvertTransaction(tx)
	result, _ := json.Marshal(txDetail)
	return string(result)
}

func getTransaction(hash common.Hash) string {
	tx, _ := GetBlockChain().GetTransaction(hash)
	if tx == nil {
		return ""
	}
	txDetail := ConvertTransaction(tx)
	result, _ := json.Marshal(txDetail)
	return string(result)
}

func getPastLogs(crit types.FilterCriteria) string {
	logs := GetLogs(crit)
	result, _ := json.Marshal(logs)
	return string(result)
}

func GetLogs(crit types.FilterCriteria) []*types.Log {
	var logs []*types.Log
	if crit.BlockHash != nil {
		logs = mysql.SelectLogsByHash(*crit.BlockHash, crit.Addresses)
	} else {
		var begin, end uint64
		if crit.FromBlock == nil || crit.FromBlock.Cmp(common.Big0) < 0 || !crit.FromBlock.IsUint64() {
			begin = GetBlockChain().Height()
		} else {
			begin = crit.FromBlock.Uint64()
		}

		if crit.ToBlock == nil || crit.ToBlock.Cmp(common.Big0) < 0 || !crit.ToBlock.IsUint64() {
			end = GetBlockChain().Height()
		} else {
			end = crit.ToBlock.Uint64()
		}
		logs = mysql.SelectLogs(begin, end, crit.Addresses)
	}

	result := types.FilterLogsByTopics(logs, crit.Topics)
	if result == nil {
		return []*types.Log{}
	}
	return result
}

func (executor *GameExecutor) callVM(param callVMData) string {
	block := getBlockByHashOrHeight(param.Height, param.Hash)
	accountdb := getAccountDBByHashOrHeight(param.Height, param.Hash)
	if accountdb == nil || block == nil {
		return "illegal height or hash"
	}

	vmCtx := vm.Context{}
	vmCtx.CanTransfer = vm.CanTransfer
	vmCtx.Transfer = vm.Transfer
	vmCtx.GetHash = GetBlockChain().GetBlockHash
	if param.From != "" {
		vmCtx.Origin = common.HexToAddress(param.From)
	}
	vmCtx.Coinbase = common.BytesToAddress(block.Header.Castor)
	vmCtx.BlockNumber = new(big.Int).SetUint64(block.Header.Height)
	vmCtx.Time = new(big.Int).SetUint64(uint64(block.Header.CurTime.Unix()))
	//set constant value
	vmCtx.Difficulty = new(big.Int).SetUint64(123)
	vmCtx.GasLimit = param.Gas
	var convertError error
	vmCtx.GasPrice, convertError = utility.StrToBigInt(param.GasPrice)
	if convertError != nil {
		logger.Errorf("call vm gas price convert error:%s,%s", convertError.Error(), param.GasPrice)
		return fmt.Sprintf("gas price illegel, data: %s", param.GasPrice)
	}
	transferValue, convertError := utility.StrToBigInt(param.Value)
	if convertError != nil {
		logger.Errorf("call vm transfer value convert error:%s,%s", convertError.Error(), param.Value)
		return fmt.Sprintf("transfer value illegel, data: %s", param.Value)
	}

	var data []byte
	if param.Data == "" || param.Data == "0x0" {
		data = []byte{}
	} else {
		data = common.FromHex(param.Data)
	}

	vmInstance := vm.NewEVMWithNFT(vmCtx, accountdb, accountdb)
	caller := vm.AccountRef(vmCtx.Origin)
	var (
		result          []byte
		leftOverGas     uint64
		contractAddress common.Address
		err             error
	)
	if param.To == "" {
		result, contractAddress, leftOverGas, _, err = vmInstance.Create(caller, data, vmCtx.GasLimit, transferValue)
		logger.Debugf("[vm_call]After execute contract create!Contract address:%s, leftOverGas: %d,error:%v", contractAddress.GetHexString(), leftOverGas, err)
	} else {
		result, leftOverGas, _, err = vmInstance.Call(caller, common.HexToAddress(param.To), data, vmCtx.GasLimit, transferValue)
		logger.Debugf("[vm_call]After execute contract call! result:%v,leftOverGas: %d,error:%v", result, leftOverGas, err)
	}

	if err != nil {
		return err.Error()
	}
	return common.ToHex(result)
}

//-----------------------------------------------------------------------------
func getAccountDBByHashOrHeight(height string, hash string) *account.AccountDB {
	var accountDB *account.AccountDB
	if height == "" && hash == "" {
		accountDB = getAccountDBByHeight(0)
	} else if hash != "" {
		accountDB = getAccountDBByHash(common.HexToHash(hash))
	} else {
		heightInt, err := strconv.Atoi(height)
		if err == nil {
			accountDB = getAccountDBByHeight(uint64(heightInt))
		}
	}
	return accountDB
}
func getAccountDBByHeight(height uint64) (accountDB *account.AccountDB) {
	var bh *types.BlockHeader
	if height == 0 {
		bh = GetBlockChain().TopBlock()
	} else {
		bh = GetBlockChain().QueryBlockHeaderByHeight(height, true)
	}
	if nil == bh {
		return nil
	}
	accountDB, _ = middleware.AccountDBManagerInstance.GetAccountDBByHash(bh.StateTree)
	return
}

func getAccountDBByHash(hash common.Hash) (accountDB *account.AccountDB) {
	b := GetBlockChain().QueryBlockByHash(hash)
	if nil == b {
		return nil
	}
	accountDB, _ = middleware.AccountDBManagerInstance.GetAccountDBByHash(b.Header.StateTree)
	return
}

func getBlockByHashOrHeight(height string, hash string) *types.Block {
	var block *types.Block
	if height == "" && hash == "" {
		header := GetBlockChain().TopBlock()
		if header != nil {
			block = GetBlockChain().QueryBlock(header.Height)
		}
	} else if hash != "" {
		block = GetBlockChain().QueryBlockByHash(common.HexToHash(hash))
	} else {
		heightInt, err := strconv.Atoi(height)
		if err == nil {
			block = GetBlockChain().QueryBlock(uint64(heightInt))
		}
	}
	return block
}
