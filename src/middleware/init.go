package middleware

import (
	"x/src/middleware/notify"
	"x/src/middleware/log"
	"x/src/common"
	"x/src/middleware/types"
)

func InitMiddleware() error {
	types.Logger = log.GetLoggerByIndex(log.MiddlewareLogConfig, common.GlobalConf.GetString("instance", "index", ""))

	notify.BUS = notify.NewBus()

	return nil
}
