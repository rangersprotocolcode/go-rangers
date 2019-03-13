package core

import (
	"x/src/middleware/types"
	"x/src/middleware/log"
	"x/src/common"
)

var (
	logger          log.Logger
	consensusLogger log.Logger
)

func InitCore(helper types.ConsensusHelper) error {
	logger = log.GetLoggerByIndex(log.CoreLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	consensusLogger = log.GetLoggerByIndex(log.ConsensusLogConfig, common.GlobalConf.GetString("instance", "index", ""))

	initPeerManager()
	if nil == BlockChainImpl {
		err := initBlockChain(helper)
		if err != nil {
			return err
		}
	}

	if nil == GroupChainImpl {
		err := initGroupChain(helper.GenerateGenesisInfo(), helper)
		if err != nil {
			return err
		}
	}
	return nil
}
