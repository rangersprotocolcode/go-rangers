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

package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/executor"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
)

var (
	logger           log.Logger
	txLogger         log.Logger
	consensusHelper  types.ConsensusHelper
	syncLogger       log.Logger
	syncHandleLogger log.Logger
)

func InitCore(helper types.ConsensusHelper, privateKey common.PrivateKey, id string) error {
	index := common.GlobalConf.GetString("instance", "index", "")
	logger = log.GetLoggerByIndex(log.CoreLogConfig, index)
	txLogger = log.GetLoggerByIndex(log.TxLogConfig, index)
	syncLogger = log.GetLoggerByIndex(log.SyncLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	syncHandleLogger = log.GetLoggerByIndex(log.SyncHandleLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	consensusHelper = helper

	initPeerManager(syncLogger)
	if nil == blockChainImpl {
		err := initBlockChain()
		if err != nil {
			return err
		}
	}

	if nil == groupChainImpl {
		initGroupChain()
	}
	InitSyncProcessor(privateKey, id)

	executor.InitExecutors()
	service.InitRewardCalculator(blockChainImpl, groupChainImpl, SyncProcessor)
	service.InitRefundManager(groupChainImpl)

	initChainHandler()

	initGameExecutor(blockChainImpl)

	return nil
}
