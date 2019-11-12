package middleware

import (
	"x/src/middleware/notify"
	"x/src/middleware/types"
	"x/src/middleware/log"
	"x/src/common"
)

var PerfLogger log.Logger

func InitMiddleware() error {

	types.InitSerialzation()
	PerfLogger = log.GetLoggerByIndex(log.PerformanceLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	notify.BUS = notify.NewBus()

	return nil
}
