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

package mysql

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/utility"
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

	rows, err := mysqlDBLog.Query(sql, from, to)
	if err != nil {
		return nil
	}
	defer rows.Close()

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

	rows, err := mysqlDBLog.Query(sql, blockhash.Hex())
	if err != nil {
		return nil
	}
	defer rows.Close()

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

func InsertLogs(height uint64, receipts types.Receipts, hash common.Hash) {
	logger.Infof("start height: %d, receipts: %d, blockhash: %s", height, len(receipts), hash.String())

	tx, err := mysqlDBLog.Begin()
	if err != nil {
		logger.Errorf("fail to begin. end height: %d, receipts: %d, blockhash: %s", height, len(receipts), hash.String())
		return
	}
	defer tx.Commit()

	sqlStr := "replace INTO contractlogs(height,logindex,blockhash, txhash, contractaddress, topic, data, topic0,topic1,topic2,topic3) values (?, ?, ?, ?, ?, ?, ?,?,?,?,?)"
	stmt, err := tx.Prepare(sqlStr)
	if err != nil {
		logger.Errorf("fail to prepare. end height: %d, receipts: %d, blockhash: %s", height, len(receipts), hash.String())
		return
	}
	defer stmt.Close()

	for _, receipt := range receipts {
		if nil != receipt && nil != receipt.Logs && 0 != len(receipt.Logs) {
			for i, log := range receipt.Logs {
				var topic0, topic1, topic2, topic3 string
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
				case 4:
					topic0 = log.Topics[0].String()
					topic1 = log.Topics[1].String()
					topic2 = log.Topics[2].String()
					topic3 = log.Topics[3].String()
				}

				topicData, _ := json.Marshal(log.Topics)

				res, err := stmt.Exec(height, uint64(i), hash.Hex(), receipt.TxHash.Hex(), log.Address.GetHexString(), utility.BytesToStr(topicData), common.ToHex(log.Data), topic0, topic1, topic2, topic3)
				if err != nil {
					logger.Errorf("fail to insert exec, err: %s. sql: %s", err, sqlStr)
					continue
				}

				lastInsertID, _ := res.LastInsertId()

				rowsAffected, _ := res.RowsAffected()

				logger.Infof("inserted height: %d, blockhash: %s, lines: %d, lastId: %d", height, hash.String(), rowsAffected, lastInsertID)
			}
		}
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
