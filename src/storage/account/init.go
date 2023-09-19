package account

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
	"strconv"
)

var accountLog log.Logger

func Init() {
	accountLog = log.GetLoggerByIndex(log.AccountLogConfig, strconv.Itoa(common.InstanceIndex))
}
