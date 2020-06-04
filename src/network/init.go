package network

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
)

//默认
const (
	gateAddrProduction = "47.96.99.105:8848"
	gateAddrDaily      = "47.96.99.105:80"
)

var Logger log.Logger

func InitNetwork(consensusHandler MsgHandler, selfMinerId, env, gate string) {
	Logger = log.GetLoggerByIndex(log.P2PLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	gateAddr := gate
	if len(gateAddr) == 0 {
		if env == "production" {
			gateAddr = gateAddrProduction
		} else {
			gateAddr = gateAddrDaily
		}
	}


	var s server
	s.Init(Logger, gateAddr, selfMinerId, consensusHandler)

	instance = s
	Logger.Warnf("connected gate: %s, env: %s", gateAddr, env)
}
