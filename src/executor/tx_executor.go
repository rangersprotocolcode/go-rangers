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

package executor

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/middleware/log"
	"com.tuntun.rangers/node/src/middleware/types"
	"strconv"
)

type txExecutors struct {
	executors map[int32]executor
}

var txExecutorsImpl *txExecutors
var logger log.Logger

func GetTxExecutor(txType int32) executor {
	return txExecutorsImpl.executors[txType]
}

func InitExecutors() {
	logger = log.GetLoggerByIndex(log.TxLogConfig, strconv.Itoa(common.InstanceIndex))

	executors := make(map[int32]executor)

	executors[types.TransactionTypeOperatorEvent] = &operatorExecutor{logger: logger}

	executors[types.TransactionTypeMinerApply] = &minerApplyExecutor{logger: logger}
	executors[types.TransactionTypeMinerAdd] = &minerAddExecutor{logger: logger}
	executors[types.TransactionTypeMinerRefund] = &minerRefundExecutor{logger: logger}
	executors[types.TransactionTypeMinerChangeAccount] = &minerChangeAccountExecutor{logger: logger}
	executors[types.TransactionTypeOperatorNode] = &minerNodeExecutor{logger: logger}

	contractExecutorInstance := &contractExecutor{logger: logger}
	executors[types.TransactionTypeContract] = contractExecutorInstance
	executors[types.TransactionTypeETHTX] = &jsonrpcExecutor{logger: logger, contractExecutor: *contractExecutorInstance}

	txExecutorsImpl = &txExecutors{executors: executors}
}
