package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
	"encoding/json"
	"strconv"
)

func getBlock(height string, hash string) string {
	block := getBlockByHashOrHeight(height, hash)
	if block == nil {
		return ""
	}
	result, _ := json.Marshal(block)
	return string(result)
}

func getBlockNumber(height string, hash string) string {
	block := getBlockByHashOrHeight(height, hash)
	if block == nil {
		return ""
	}
	return strconv.FormatUint(block.Header.Height, 10)
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
	result, _ := json.Marshal(block.Transactions[indexInt])
	return string(result)
}

func getTransaction(hash common.Hash) string {
	tx, _ := GetBlockChain().GetTransaction(hash)
	result, _ := json.Marshal(tx)
	return string(result)
}

func getPastLogs(from uint64, to uint64, addresses []common.Address, topics [][]string) string {
	var logs []*types.Log
	if to == 0 {
		to = GetBlockChain().Height()
	}
	for i := from; i <= to; i++ {
		block := GetBlockChain().QueryBlock(i)
		if block == nil {
			continue
		}
		txHashList := block.Header.Transactions[0]
		for _, txHash := range txHashList {
			tx := service.GetTransactionPool().GetExecuted(txHash)
			if tx == nil {
				continue
			}
			for _, log := range tx.Receipt.Logs {
				logs = append(logs, log)
			}
		}
	}
	logs = filterLogs(logs, addresses, topics)
	result, _ := json.Marshal(logs)
	return string(result)
}

// filterLogs creates a slice of logs matching the given criteria.
func filterLogs(logs []*types.Log, addresses []common.Address, topics [][]string) []*types.Log {
	var ret []*types.Log
Logs:
	for _, log := range logs {

		if len(addresses) > 0 && !includes(addresses, log.Address) {
			continue
		}
		// If the to filtered topics is greater than the amount of topics in logs, skip.
		if len(topics) > len(log.Topics) {
			continue Logs
		}
		for i, sub := range topics {
			match := len(sub) == 0 // empty rule set == wildcard
			for _, topic := range sub {
				if log.Topics[i].String() == topic {
					match = true
					break
				}
			}
			if !match {
				continue Logs
			}
		}
		ret = append(ret, log)
	}
	return ret
}

func includes(addresses []common.Address, a common.Address) bool {
	for _, addr := range addresses {
		if addr == a {
			return true
		}
	}

	return false
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
		block = GetBlockChain().CurrentBlock()
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
