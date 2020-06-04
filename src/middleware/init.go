package middleware

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/notify"
	"com.tuntun.rocket/node/src/middleware/types"
)

var PerfLogger log.Logger
var HeightLogger log.Logger

func InitMiddleware() error {

	types.InitSerialzation()
	PerfLogger = log.GetLoggerByIndex(log.PerformanceLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	HeightLogger = log.GetLoggerByIndex(log.HeightLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	notify.BUS = notify.NewBus()

	threshold := common.GlobalConf.GetInt("coiner", "threshold", 2)
	privateKey := common.GlobalConf.GetString("coiner", "privateKey", "10695fe7b429427aa01044d97f48e14e1244d206eda8dfa812996310100f4cd1")
	signer0 := common.GlobalConf.GetString("coiner", "signer0", "0xf89eebcc07e820f5a8330f52111fa51dd9dfb925")
	signer1 := common.GlobalConf.GetString("coiner", "signer1", "0x9951146d4fdbd0903d450b315725880a90383f38")
	signer2 := common.GlobalConf.GetString("coiner", "signer2", "0x7edd0ef9da9cec334a7887966cc8dd71d590eeb7")

	types.CoinerSignVerifier = types.Ecc{SignLimit: threshold, Privkey: privateKey, Whitelist: []string{signer0, signer1, signer2}}
	if nil != common.DefaultLogger {
		common.DefaultLogger.Debugf("coiner sign verifier:%v", types.CoinerSignVerifier)
	}

	return nil
}
