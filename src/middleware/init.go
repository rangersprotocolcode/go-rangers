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

package middleware

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/mysql"
	"com.tuntun.rocket/node/src/middleware/notify"
	"com.tuntun.rocket/node/src/middleware/types"
	"strconv"
	"time"
)

var (
	PerfLogger    = log.GetLoggerByIndex(log.PerformanceLogConfig, strconv.Itoa(common.InstanceIndex))
	MonitorLogger = log.GetLoggerByIndex(log.MonitorLogConfig, strconv.Itoa(common.InstanceIndex))
)

func InitMiddleware() error {
	start := time.Now()
	common.DefaultLogger.Infof("start InitMiddleware")
	defer func() {
		common.DefaultLogger.Infof("end InitMiddleware, cost: %s", time.Now().Sub(start).String())
	}()

	types.InitSerialzation()
	notify.BUS = notify.NewBus()
	mysql.InitMySql()

	InitLock()
	InitDataChannel()
	initAccountDBManager()
	return nil
}

func InitLock() {
	lock = NewLoglock("blockchain")
}
