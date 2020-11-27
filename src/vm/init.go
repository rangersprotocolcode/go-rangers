package vm

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
	"math/big"
)

// LogConfig are the configuration options for structured logger the EVM
type LogConfig struct {
	DisableMemory     bool // disable memory capture
	DisableStack      bool // disable stack capture
	DisableStorage    bool // disable storage capture
	DisableReturnData bool // disable return data capture
}

var chainID *big.Int
var vmTracer Tracer

/**
chainID:
0  main net
1  dev net
2  test net
*/
func InitVM(chainID *big.Int) {
	if chainID == nil {
		chainID = big.NewInt(0)
	} else {
		chainID = chainID
	}

	index := common.GlobalConf.GetString("instance", "index", "")
	logger := log.GetLoggerByIndex(log.VMLogConfig, index)

	config := LogConfig{false, false, false, false}
	vmTracer = &mdLogger{
		cfg:    &config,
		logger: logger,
	}
}
