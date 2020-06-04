package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/types"
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
	initRewardCalculator(MinerManagerImpl, blockChainImpl, groupChainImpl)
	initRefundManager()

	initChainHandler()

	initGameExecutor(blockChainImpl)

	return nil
}
