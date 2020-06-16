package service

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
)

var logger, txLogger, txPoolLogger log.Logger

func InitService() {
	logger = log.GetLoggerByIndex(log.CoreLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	txLogger = log.GetLoggerByIndex(log.TxLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	txPoolLogger = log.GetLoggerByIndex(log.TxPoolLogConfig, common.GlobalConf.GetString("instance", "index", ""))

	initTransactionPool()
	initTxManager()
	initFTManager()
	initNFTManager()
	initAccountDBManager()
}
