package mysql

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"strconv"
	"time"
)

var (
	mysqlDBLog *sql.DB
	mysqlErr   error
	logger     log.Logger
)

// 初始化链接
func InitMySql() {
	logger = log.GetLoggerByIndex(log.MysqlLogConfig, strconv.Itoa(common.InstanceIndex))
	dsn := fmt.Sprintf("file:logs-%s.db?mode=rwc&_journal_mode=WAL&_cache_size=-500000",strconv.Itoa(common.InstanceIndex))
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		panic(err)
	}

	_, err = db.Exec("CREATE TABLE if NOT EXISTS `contractlogs`( id INTEGER PRIMARY KEY AUTOINCREMENT,`height` INTEGER NOT NULL, `logindex` bigint NOT NULL, `blockhash` varchar(66) NOT NULL, `txhash` varchar(66) NOT NULL, `contractaddress` varchar(66) NOT NULL, `topic` varchar(800) NOT NULL, `data` text, `topic0` varchar(66) DEFAULT '', `topic1` varchar(66) DEFAULT '', `topic2` varchar(66) DEFAULT '', `topic3` varchar(66) DEFAULT '', UNIQUE (`logindex`,`txhash`, `topic`));")
	_, err = db.Exec("CREATE INDEX if NOT EXISTS height ON contractlogs (height);")
	_, err = db.Exec("CREATE INDEX if NOT EXISTS blockhash ON contractlogs (blockhash);")
	if err != nil {
		panic(err)
	}

	mysqlDBLog = db
	logger.Infof("connected sqlite")

	// 最大连接数
	mysqlDBLog.SetMaxOpenConns(5)
	// 闲置连接数
	mysqlDBLog.SetMaxIdleConns(5)
	// 最大连接周期
	mysqlDBLog.SetConnMaxLifetime(100 * time.Second)

	if mysqlErr = mysqlDBLog.Ping(); nil != mysqlErr {
		mysqlDBLog.Close()
		panic(mysqlErr.Error())
	}
}
