package network

import (
	"x/src/middleware/log"
	"x/src/common"
)

const (
	gateAddrProduction = "47.96.99.105:8848"
	gateAddrDaily      = "47.96.99.105:8848"
)

var gateAddr string
var Logger log.Logger

func InitNetwork(selfMinerId string, consensusHandler MsgHandler, env string, gate string) {
	Logger = log.GetLoggerByIndex(log.P2PLogConfig, common.GlobalConf.GetString("instance", "index", ""))

	if len(gate) != 0 {
		gateAddr = gate
	} else {
		if env == "production" {
			gateAddr = gateAddrProduction
		} else {
			gateAddr = gateAddrDaily
		}
	}

	getNetMemberInfo("")

	var s server
	s.Init(Logger, selfMinerId, consensusHandler)

	instance = s
}
