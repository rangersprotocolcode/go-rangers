package core

import (
	"x/src/middleware/types"
	"x/src/middleware/log"
	"x/src/common"
)

var (
	logger          log.Logger
	consensusLogger log.Logger
	consensusHelper types.ConsensusHelper
)

func InitCore(helper types.ConsensusHelper) error {
	logger = log.GetLoggerByIndex(log.CoreLogConfig, common.GlobalConf.GetString("instance", "index", ""))
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
	initChainHandler()
	return nil
}
