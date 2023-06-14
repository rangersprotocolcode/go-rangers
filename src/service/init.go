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
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package service

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
	"strconv"
	"time"
)

var (
	logger       = log.GetLoggerByIndex(log.CoreLogConfig, strconv.Itoa(common.InstanceIndex))
	txLogger     = log.GetLoggerByIndex(log.TxLogConfig, strconv.Itoa(common.InstanceIndex))
	txPoolLogger = log.GetLoggerByIndex(log.TxPoolLogConfig, strconv.Itoa(common.InstanceIndex))
)

func InitService() {
	start := time.Now()
	common.DefaultLogger.Infof("start InitService")
	defer func() {
		common.DefaultLogger.Infof("end InitService, cost: %s", time.Now().Sub(start).String())
	}()

	InitMinerManager()
	initTransactionPool()
	initFTManager()
}
