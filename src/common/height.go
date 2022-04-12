package common

import (
	"com.tuntun.rocket/node/src/middleware/log"
	"sync/atomic"
)

var localChainInfo chainInfo
var blockHeightLogger log.Logger

type chainInfo struct {
	currentBlockHeight atomic.Value
}

func GetBlockHeight() uint64 {
	return localChainInfo.currentBlockHeight.Load().(uint64)
}

func SetBlockHeight(height uint64) {
	localChainInfo.currentBlockHeight.Store(height)
	blockHeightLogger.Debugf("set height:%d", GetBlockHeight())
}
