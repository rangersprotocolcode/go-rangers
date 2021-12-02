package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
	"encoding/json"
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

//-----------------------------------------------------------------------------
func getAccountDBByHashOrHeight(height string, hash string) *account.AccountDB {
	var accountDB *account.AccountDB
	if height == "" && hash == "" {
		accountDB = service.AccountDBManagerInstance.GetAccountDB("", true)
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
	if height == 0 {
		accountDB = service.AccountDBManagerInstance.GetLatestStateDB()
	} else {
		b := GetBlockChain().QueryBlock(height)
		if nil == b {
			return nil
		}
		accountDB, _ = service.AccountDBManagerInstance.GetAccountDBByHash(b.Header.StateTree)
	}
	return
}

func getAccountDBByHash(hash common.Hash) (accountDB *account.AccountDB) {
	b := GetBlockChain().QueryBlockByHash(hash)
	if nil == b {
		return nil
	}
	accountDB, _ = service.AccountDBManagerInstance.GetAccountDBByHash(b.Header.StateTree)
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
