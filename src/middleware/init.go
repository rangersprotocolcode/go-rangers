package middleware

import (
	"x/src/middleware/notify"
	"x/src/middleware/log"
	"x/src/common"
)

// middleware模块统一logger
var Logger log.Logger

func InitMiddleware() error {
	Logger = log.GetLoggerByIndex(log.MiddlewareLogConfig, common.GlobalConf.GetString("instance", "index", ""))

	notify.BUS = notify.NewBus()

	return nil
}
