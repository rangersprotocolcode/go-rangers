// Copyright 2020 The RangersProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RangersProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RangersProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package network

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/middleware/log"
	"fmt"
	"strconv"
	"time"
)

var (
	p2pLogger   log.Logger
	bizLogger   log.Logger
	txRcvLogger log.Logger
)

func InitNetwork(consensusHandler MsgHandler, selfMinerId []byte, env, gate, outerGateAddr string, isSending bool) {
	p2pLogger = log.GetLoggerByIndex(log.P2PLogConfig, strconv.Itoa(common.InstanceIndex))
	bizLogger = log.GetLoggerByIndex(log.P2PBizLogConfig, strconv.Itoa(common.InstanceIndex))
	txRcvLogger = log.GetLoggerByIndex(log.TxRcvLogConfig, strconv.Itoa(common.InstanceIndex))

	start := time.Now()
	common.DefaultLogger.Infof("start InitNetwork")
	defer func() {
		common.DefaultLogger.Infof("end InitNetwork, cost: %s", time.Now().Sub(start).String())
	}()

	fmt.Println("Connecting to: " + gate)
	if !common.IsSub() {
		fmt.Print("isSending: ")
		fmt.Println(isSending)
	}

	var s server
	s.Init(gate, outerGateAddr, selfMinerId, consensusHandler, isSending)

	instance = s
	bizLogger.Warnf("connected gate: %s, env: %s", gate, env)
}
