// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

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
	//signer0 := common.GlobalConf.GetString("coiner", "signer0", "0xf89eebcc07e820f5a8330f52111fa51dd9dfb925")
	//signer1 := common.GlobalConf.GetString("coiner", "signer1", "0x9951146d4fdbd0903d450b315725880a90383f38")
	//signer2 := common.GlobalConf.GetString("coiner", "signer2", "0x7edd0ef9da9cec334a7887966cc8dd71d590eeb7")
	signer0 := common.GlobalConf.GetString("coiner", "signer0", "0x02380c84420993B7A3A90111eE3b3CCDa15D8A27")

	types.CoinerSignVerifier = types.Ecc{SignLimit: threshold, Privkey: privateKey, Whitelist: []string{signer0}}
	if nil != common.DefaultLogger {
		common.DefaultLogger.Debugf("coiner sign verifier:%v", types.CoinerSignVerifier)
	}

	return nil
}
