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
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
)

var (
	logger          log.Logger
	txLogger        log.Logger
	consensusLogger log.Logger
	consensusHelper types.ConsensusHelper
)

func InitCore(helper types.ConsensusHelper) error {
	logger = log.GetLoggerByIndex(log.CoreLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	txLogger = log.GetLoggerByIndex(log.TxLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	consensusLogger = log.GetLoggerByIndex(log.ConsensusLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	consensusHelper = helper

	initPeerManager()
	if nil == blockChainImpl {
		err := initBlockChain()
		if err != nil {
			return err
		}
	}

	if nil == groupChainImpl {
		initGroupChain()
	}

	initExecutors()
	initRewardCalculator(service.MinerManagerImpl, blockChainImpl, groupChainImpl)
	initRefundManager()

	initChainHandler()

	initGameExecutor(blockChainImpl)

	return nil
}
