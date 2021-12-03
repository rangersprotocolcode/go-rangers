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
	MysqlDB  *sql.DB
	mysqlErr error
	logger   log.Logger
)

// 初始化链接
func InitMySql(dbDSN string) {
	logger = log.GetLoggerByIndex(log.MysqlLogConfig, strconv.Itoa(common.InstanceIndex))
	if 0 == len(dbDSN) {
		return
	}

	//dbDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s", USERNAME, PASSWORD, HOST, PORT, DATABASE, CHARSET)
	MysqlDB, mysqlErr = sql.Open("mysql", dbDSN)

	// 打开连接失败
	if mysqlErr != nil {
		logger.Errorf("fail to connect dbDSN: " + dbDSN)
		panic("dbDSN: " + dbDSN + mysqlErr.Error())
	}

	logger.Infof("connected dbDSN: " + dbDSN)
	// 最大连接数
	MysqlDB.SetMaxOpenConns(40)
	// 闲置连接数
	MysqlDB.SetMaxIdleConns(5)
	// 最大连接周期
	MysqlDB.SetConnMaxLifetime(100 * time.Second)

	if mysqlErr = MysqlDB.Ping(); nil != mysqlErr {
		MysqlDB.Close()
		panic(mysqlErr.Error())
	}
}