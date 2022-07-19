package mysql

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/utility"
	"encoding/json"
)

func SelectLogs(from, to uint64, contractAddresses []common.Address) []*types.Log {
	if nil == mysqlDBLog {
		return nil
	}

	sql := "select height,logindex, blockhash,txhash,contractaddress,topic,data FROM contractlogs WHERE (height>=? and height<=?) "
	if 0 != len(contractAddresses) {
		sql += "and( "
		for _, contractAddress := range contractAddresses {
			sql += " contractaddress = \"" + contractAddress.GetHexString() + "\" " + "or"
		}
		sql = sql[:len(sql)-2] + ")"
	}

	sql += " limit 100"

	rows, err := mysqlDBLog.Query(sql, from, to)
	if err != nil {
		return nil
	}
	defer rows.Close()

	// 循环读取结果集中的数据
	result := make([]*types.Log, 0)
	for rows.Next() {
		var (
			height, index                                   uint64
			blockhash, txhash, contractaddress, topic, data string
		)
		err := rows.Scan(&height, &index, &blockhash, &txhash, &contractaddress, &topic, &data)
		if err != nil {
			logger.Errorf("scan failed, err: %v", err)
			return nil
		}

		log := types.Log{
			Address:     common.HexToAddress(contractaddress),
			Data:        common.FromHex(data),
			TxHash:      common.HexToHash(txhash),
			BlockHash:   common.HexToHash(blockhash),
			BlockNumber: height,
			Index:       uint(index),
		}

		json.Unmarshal(utility.StrToBytes(topic), &log.Topics)
		result = append(result, &log)
	}

	return result
}

func SelectLogsByHash(blockhash common.Hash, contractAddresses []common.Address) []*types.Log {
	if nil == mysqlDBLog {
		return nil
	}

	sql := "select height,logindex, blockhash,txhash,contractaddress,topic,data FROM contractlogs WHERE blockhash = ? "
	if 0 != len(contractAddresses) {
		sql += "and( "
		for _, contractAddress := range contractAddresses {
			sql += " contractaddress = \"" + contractAddress.GetHexString() + "\" " + "or"
		}
		sql = sql[:len(sql)-2] + ")"
	}

	sql += " limit 100"

	rows, err := mysqlDBLog.Query(sql, blockhash.Hex())
	if err != nil {
		return nil
	}
	defer rows.Close()

	// 循环读取结果集中的数据
	result := make([]*types.Log, 0)
	for rows.Next() {
		var (
			height, index                                   uint64
			blockhash, txhash, contractaddress, topic, data string
		)
		err := rows.Scan(&height, &index, &blockhash, &txhash, &contractaddress, &topic, &data)
		if err != nil {
			logger.Errorf("scan failed, err: %v", err)
			return nil
		}

		log := types.Log{
			Address:     common.HexToAddress(contractaddress),
			Data:        common.FromHex(data),
			TxHash:      common.HexToHash(txhash),
			BlockHash:   common.HexToHash(blockhash),
			BlockNumber: height,
			Index:       uint(index),
		}
		json.Unmarshal(utility.StrToBytes(topic), &log.Topics)
		result = append(result, &log)
	}

	return result
}
