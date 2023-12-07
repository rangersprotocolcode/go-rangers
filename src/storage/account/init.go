package account

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/middleware/log"
	"strconv"
)

var accountLog log.Logger

func Init() {
	accountLog = log.GetLoggerByIndex(log.AccountLogConfig, strconv.Itoa(common.InstanceIndex))
}
