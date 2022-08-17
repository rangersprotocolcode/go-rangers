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

package network

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
	"fmt"
)

var p2pLogger log.Logger
var bizLogger log.Logger

func InitNetwork(consensusHandler MsgHandler, selfMinerId []byte, env, gate, outerGateAddr string, isSending bool) {
	p2pLogger = log.GetLoggerByIndex(log.P2PLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	bizLogger = log.GetLoggerByIndex(log.P2PBizLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	fmt.Println("Connecting to: " + gate)
	fmt.Println(isSending)

	var s server
	s.Init(bizLogger, gate, outerGateAddr, selfMinerId, consensusHandler, isSending)

	instance = s
	bizLogger.Warnf("connected gate: %s, env: %s", gate, env)
}
