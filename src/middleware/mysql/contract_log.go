package mysql

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/utility"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"time"
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
	for _, receipt := range receipts {
		if nil != receipt.Logs && 0 != len(receipt.Logs) {
			for i, log := range receipt.Logs {
				sqlStr := "replace INTO contractlogs(height,logindex,blockhash, txhash, contractaddress, topic, data, topic0,topic1,topic2,topic3) values (?, ?, ?, ?, ?, ?, ?,?,?,?,?)"
				var vals []interface{}

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

				vals = append(vals, height, uint64(i), hash.Hex(), receipt.TxHash.Hex(), log.Address.GetHexString(), utility.BytesToStr(topicData), common.ToHex(log.Data), topic0, topic1, topic2, topic3)

				stmt, err := mysqlDBLog.Prepare(sqlStr)
				if err != nil {
					logger.Errorf("fail to insert, err: %s. sql: %s", err, sqlStr)
					continue
				}

				//format all vals at once
				res, err := stmt.Exec(vals...)
				if err != nil {
					logger.Errorf("fail to insert exec, err: %s. sql: %s", err, sqlStr)
					stmt.Close()
					continue
				}

				//插入数据的主键id
				lastInsertID, _ := res.LastInsertId()

				//影响行数
				rowsAffected, _ := res.RowsAffected()

				logger.Infof("inserted height: %d, blockhash: %s, lines: %d, lastId: %d", height, hash.String(), rowsAffected, lastInsertID)
				stmt.Close()
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

func SyncOldData() {
	dsn := common.LocalChainConfig.MysqlDSN
	if 0 == len(dsn) {
		logger.Errorf("no mysql DSN")
		return
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		logger.Errorf("open mysql error. DSN: %s, error: %s", dsn, err)
		return
	}
	// 最大连接数
	db.SetMaxOpenConns(5)
	// 闲置连接数
	db.SetMaxIdleConns(5)
	// 最大连接周期
	db.SetConnMaxLifetime(100 * time.Second)
	if err = db.Ping(); nil != err {
		db.Close()
		logger.Errorf("ping mysql error. DSN: %s, error: %s", dsn, err)
		return
	}

	i, j := countLogs(), uint64(100)
	logger.Warnf("start sync logs. %s. from: %d", dsn, i)

	base := "select height,logindex, blockhash,txhash,contractaddress,topic," +
		"data,topic0,topic1,topic2,topic3 FROM contractlogs order by height, logindex limit"
	for {
		sql := fmt.Sprintf("%s %d,%d", base, i, j)

		rows, err := db.Query(sql)
		if err != nil {
			logger.Errorf("query mysql error. sql: %s, DSN: %s, error: %s", sql, dsn, err)
			rows.Close()
			continue
		}

		count := uint64(0)
		for rows.Next() {
			var (
				height, index                                                                   uint64
				blockhash, txhash, contractaddress, topic, data, topic0, topic1, topic2, topic3 string
			)
			err := rows.Scan(&height, &index, &blockhash, &txhash, &contractaddress, &topic, &data, &topic0, &topic1, &topic2, &topic3)
			if err != nil {
				logger.Errorf("scan mysql error. sql: %s, DSN: %s, error: %s", sql, dsn, err)
				panic(err)
			}
			insertLog(height, index, blockhash, txhash, contractaddress, topic, data, topic0, topic1, topic2, topic3)
			count++
		}
		rows.Close()

		if count < j {
			logger.Warnf("rows less than 100, try again %d-%d", i, count)
			i += count
			time.Sleep(5 * time.Second)
		} else {
			i += j
		}

		if i%1000 == 0 {
			logger.Infof("sync old data. %d", i)
		}
	}
}

func countLogs() uint64 {
	if nil == mysqlDBLog {
		return 0
	}

	sql := "select count(*) as num FROM contractlogs"
	rows, err := mysqlDBLog.Query(sql)
	if err != nil {
		return 0
	}
	defer rows.Close()

	// 循环读取结果集中的数据
	var result uint64
	for rows.Next() {
		err := rows.Scan(&result)
		if err != nil {
			logger.Errorf("scan failed, err: %v", err)
			return 0
		}
		return result
	}

	return result
}

func insertLog(height, index uint64, blockhash, txhash, contractaddress, topic, data, topic0, topic1, topic2, topic3 string) {
	sql := "replace INTO contractlogs(height,logindex,blockhash, txhash, contractaddress, topic, data, topic0,topic1,topic2,topic3) values (?,?,?,?,?,?,?,?,?,?,?)"

	stmt, err := mysqlDBLog.Prepare(sql)
	if err != nil {
		logger.Errorf("fail to prepare insert log, err: ", err)
		return
	}
	defer stmt.Close()

	//format all vals at once
	_, err = stmt.Exec(height, index, blockhash, txhash, contractaddress, topic, data, topic0, topic1, topic2, topic3)
	if err != nil {
		logger.Errorf("fail to insert log, err: ", err)
		return
	}
}
