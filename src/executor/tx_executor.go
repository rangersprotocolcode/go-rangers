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

package executor

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/types"
)

type txExecutors struct {
	executors map[int32]executor
}

var txExecutorsImpl *txExecutors

func GetTxExecutor(txType int32) executor {
	return txExecutorsImpl.executors[txType]
}

func InitExecutors() {
	logger := log.GetLoggerByIndex(log.TxLogConfig, common.GlobalConf.GetString("instance", "index", ""))

	executors := make(map[int32]executor)

	executors[types.TransactionTypeOperatorEvent] = &operatorExecutor{logger: logger}

	executors[types.TransactionTypeMinerApply] = &minerApplyExecutor{logger: logger}
	executors[types.TransactionTypeMinerAdd] = &minerAddExecutor{logger: logger}
	executors[types.TransactionTypeMinerRefund] = &minerRefundExecutor{logger: logger}

	executors[types.TransactionTypeSetExchangeRate] = &exchangeRateExecutor{}

	executors[types.TransactionTypeContract] = &contractExecutor{}
	executors[types.TransactionTypeETHTX] = &contractExecutor{}

	txExecutorsImpl = &txExecutors{executors: executors}
}
