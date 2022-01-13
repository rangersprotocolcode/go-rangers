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

package service

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
)

var logger, txLogger, txPoolLogger log.Logger

func InitService() {
	index := common.GlobalConf.GetString("instance", "index", "")
	logger = log.GetLoggerByIndex(log.CoreLogConfig, index)
	txLogger = log.GetLoggerByIndex(log.TxLogConfig, index)
	txPoolLogger = log.GetLoggerByIndex(log.TxPoolLogConfig, index)

	InitMinerManager()
	initTransactionPool()
	initFTManager()
	initAccountDBManager()
}
