package middleware

import (
	"x/src/middleware/notify"
	"x/src/middleware/types"
	"x/src/middleware/log"
	"x/src/common"
)

var PerfLogger log.Logger
var HeightLogger log.Logger

func InitMiddleware() error {

	types.InitSerialzation()
	PerfLogger = log.GetLoggerByIndex(log.PerformanceLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	HeightLogger = log.GetLoggerByIndex(log.HeightLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	notify.BUS = notify.NewBus()

	threshold := common.GlobalConf.GetInt("coiner", "threshold", 2)
	privateKey := common.GlobalConf.GetString("coiner", "privateKey", "")
	signer0 := common.GlobalConf.GetString("coiner", "signer0", "")
	signer1 := common.GlobalConf.GetString("coiner", "signer1", "")
	signer2 := common.GlobalConf.GetString("coiner", "signer2", "")

	types.CoinerSignVerifier = types.Ecc{SignLimit: threshold, Privkey: privateKey, Whitelist: []string{signer0, signer1, signer2}}
	if nil != common.DefaultLogger {
		common.DefaultLogger.Debugf("coiner sign verifier:%v", types.CoinerSignVerifier)
	}

	return nil
}
