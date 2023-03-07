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

// 插入数据
func InsertLogs(height uint64, receipts types.Receipts, hash common.Hash) {
	logger.Infof("start height: %d, receipts: %d, blockhash: %s", height, len(receipts), hash.String())
	vals := []interface{}{}
	sqlStr := "replace INTO contractlogs(height,logindex,blockhash, txhash, contractaddress, topic, data, topic0,topic1,topic2,topic3) values "
	for _, receipt := range receipts {
		if nil != receipt.Logs && 0 != len(receipt.Logs) {
			for i, log := range receipt.Logs {
				sqlStr += "(?, ?, ?, ?, ?, ?, ?,?,?,?,?),"
				topicData, _ := json.Marshal(log.Topics)
				var topic0 string
				var topic1 string
				var topic2 string
				var topic3 string
				switch len(log.Topics) {
				case 0:
					break
				case 1:
					topic0 = log.Topics[0].String()
				case 2:
					topic0 = log.Topics[0].String()
					topic1 = log.Topics[1].String()
				case 3:
					topic0 = log.Topics[0].String()
					topic1 = log.Topics[1].String()
					topic2 = log.Topics[2].String()
				default:
					topic0 = log.Topics[0].String()
					topic1 = log.Topics[1].String()
					topic2 = log.Topics[2].String()
					topic3 = log.Topics[3].String()
				}
				vals = append(vals, height, uint64(i), hash.Hex(), receipt.TxHash.Hex(), log.Address.GetHexString(), utility.BytesToStr(topicData), common.ToHex(log.Data), topic0, topic1, topic2, topic3)
			}
		}
	}

	if 0 != len(vals) {
		sqlStr = sqlStr[0 : len(sqlStr)-1]
		//prepare the statement
		stmt, err := mysqlDBLog.Prepare(sqlStr)
		if err != nil {
			logger.Errorf("fail to insert, err: ", err)
			return
		}
		defer stmt.Close()

		//format all vals at once
		res, err := stmt.Exec(vals...)
		if err != nil {
			logger.Errorf("fail to insert, err: ", err)
			return
		}

		//插入数据的主键id
		lastInsertID, _ := res.LastInsertId()

		//影响行数
		rowsAffected, _ := res.RowsAffected()

		logger.Infof("inserted height: %d, blockhash: %s, lines: %d, lastId: %d", height, hash.String(), rowsAffected, lastInsertID)
	}

	logger.Infof("end height: %d, receipts: %d, blockhash: %s", height, len(receipts), hash.String())
}

func DeleteLogs(height uint64, blockHash common.Hash) {
	psmt, err := mysqlDBLog.Prepare("DELETE FROM contractlogs WHERE height = ? and blockhash = ?")
	if err != nil {
		logger.Error(err)
		return
	}
	defer psmt.Close()

	result, err := psmt.Exec(height, blockHash.Hex())
	if err != nil {
		logger.Error(err)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	logger.Debugf("deleted:%d %s, rowsAffected %d", height, blockHash.Hex(), rowsAffected)

}
