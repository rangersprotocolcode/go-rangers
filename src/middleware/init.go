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

package middleware

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/middleware/log"
	"com.tuntun.rangers/node/src/middleware/mysql"
	"com.tuntun.rangers/node/src/middleware/notify"
	"com.tuntun.rangers/node/src/middleware/types"
	"strconv"
	"time"
)

var (
	PerfLogger log.Logger

	MonitorLogger log.Logger
)

func InitMiddleware() error {
	start := time.Now()
	common.DefaultLogger.Infof("start InitMiddleware")
	defer func() {
		common.DefaultLogger.Infof("end InitMiddleware, cost: %s", time.Now().Sub(start).String())
	}()

	PerfLogger = log.GetLoggerByIndex(log.PerformanceLogConfig, strconv.Itoa(common.InstanceIndex))
	MonitorLogger = log.GetLoggerByIndex(log.MonitorLogConfig, strconv.Itoa(common.InstanceIndex))

	types.InitSerialzation()
	notify.BUS = notify.NewBus()
	mysql.InitMySql()

	InitLock()
	InitDataChannel()
	initAccountDBManager()
	return nil
}

func Close() {
	AccountDBManagerInstance.Close()
	mysql.CloseMysql()
}

func InitLock() {
	lock = NewLoglock("blockchain")
}
