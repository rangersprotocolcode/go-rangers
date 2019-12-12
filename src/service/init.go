package service

import (
	"x/src/middleware/log"
	"x/src/common"
)

var logger, txLogger log.Logger

func InitService(nodeType byte) {
	logger = log.GetLoggerByIndex(log.CoreLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	txLogger = log.GetLoggerByIndex(log.TxLogConfig, common.GlobalConf.GetString("instance", "index", ""))

	initTransactionPool(nodeType)
	initTxManager()
	initFTManager()
	initNFTManager()
	initAccountDBManager()
}
