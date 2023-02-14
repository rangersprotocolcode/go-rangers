package mysql

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"strconv"
	"time"
)

var (
	mysqlDBLog *sql.DB
	mysqlErr   error
	logger     log.Logger
)

// 初始化链接
func InitMySql(dbDSNLog string) {
	logger = log.GetLoggerByIndex(log.MysqlLogConfig, strconv.Itoa(common.InstanceIndex))
	if 0 == len(dbDSNLog) {
		return
	}

	mysqlDBLog, mysqlErr = sql.Open("mysql", dbDSNLog)

	// 打开连接失败
	if mysqlErr != nil {
		logger.Errorf("fail to connect dbDSN: " + dbDSNLog)
		panic("dbDSN: " + dbDSNLog + mysqlErr.Error())
	}

	logger.Infof("connected dbDSN: " + dbDSNLog)
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
