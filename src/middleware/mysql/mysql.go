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
	"com.tuntun.rangers/node/src/middleware/log"
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"strconv"
	"time"
)

var (
	mysqlDBLog *sql.DB
	mysqlErr   error
	logger     log.Logger
)

func InitMySql() {
	mkWorkingDir()
	logger = log.GetLoggerByIndex(log.MysqlLogConfig, strconv.Itoa(common.InstanceIndex))
	dsn := fmt.Sprintf("file:storage%s/logs/logs.db?mode=rwc&_journal_mode=WAL&_cache_size=-500000", strconv.Itoa(common.InstanceIndex))
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		panic(err)
	}

	_, err = db.Exec("CREATE TABLE if NOT EXISTS `contractlogs`( id INTEGER PRIMARY KEY AUTOINCREMENT,`height` INTEGER NOT NULL, `logindex` bigint NOT NULL, `blockhash` varchar(66) NOT NULL, `txhash` varchar(66) NOT NULL, `contractaddress` varchar(66) NOT NULL, `topic` varchar(800) NOT NULL, `data` text, `topic0` varchar(66) DEFAULT '', `topic1` varchar(66) DEFAULT '', `topic2` varchar(66) DEFAULT '', `topic3` varchar(66) DEFAULT '', UNIQUE (`logindex`,`txhash`, `topic`));")
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("CREATE INDEX if NOT EXISTS height ON contractlogs (height);")
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("CREATE INDEX if NOT EXISTS blockhash ON contractlogs (blockhash);")
	if err != nil {
		panic(err)
	}

	_, err = db.Exec("CREATE TABLE if NOT EXISTS `groupIndex`( id INTEGER PRIMARY KEY AUTOINCREMENT,`workheight` INTEGER NOT NULL, `dismissheight` INTEGER NOT NULL, `groupheight` INTEGER NOT NULL,`hash` varchar(100) UNIQUE NOT NULL);")
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("CREATE INDEX if NOT EXISTS workheight ON groupIndex (workheight);")
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("CREATE INDEX if NOT EXISTS dismissheight ON groupIndex (dismissheight);")
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("CREATE INDEX if NOT EXISTS groupHeight ON groupIndex (groupheight);")
	if err != nil {
		panic(err)
	}

	mysqlDBLog = db
	logger.Infof("connected sqlite")

	mysqlDBLog.SetMaxOpenConns(5)
	mysqlDBLog.SetMaxIdleConns(5)
	mysqlDBLog.SetConnMaxLifetime(100 * time.Second)

	if mysqlErr = mysqlDBLog.Ping(); nil != mysqlErr {
		mysqlDBLog.Close()
		panic(mysqlErr.Error())
	}
}

func mkWorkingDir() {
	path := "storage" + strconv.Itoa(common.InstanceIndex) + "/logs"
	_, err := os.Stat(path)
	if err == nil {
		return
	}

	os.MkdirAll(path, os.ModePerm)
}

func CloseMysql() {
	if nil != mysqlDBLog {
		mysqlDBLog.Close()
	}

}
