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

	types.CoinerSignVerifier = types.Ecc{SignLimit: 2, Privkey: "10695fe7b429427aa01044d97f48e14e1244d206eda8dfa812996310100f4cd1",
		Whitelist: []string{"0xf89eebcc07e820f5a8330f52111fa51dd9dfb925", "0x9951146d4fdbd0903d450b315725880a90383f38", "0x7edd0ef9da9cec334a7887966cc8dd71d590eeb7"}}
	return nil
}
