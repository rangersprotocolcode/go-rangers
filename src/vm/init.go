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

/**
chainID:
0  main net
1  dev net
2  test net
*/
var chainID *big.Int
var vmTracer Tracer
var logger log.Logger

func InitVM() {
	//todo configurable?
	chainID = big.NewInt(0)

	index := common.GlobalConf.GetString("instance", "index", "")
	logger = log.GetLoggerByIndex(log.VMLogConfig, index)

	config := LogConfig{false, false, false, false}
	vmTracer = NewStructLogger(&config, logger)
}

// CanTransfer checks whether there are enough funds in the address' account to make a transfer.
// This does not take the necessary gas in to account to make the transfer valid.
func CanTransfer(db StateDB, addr common.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Cmp(amount) >= 0
}

// Transfer subtracts amount from sender and adds amount to recipient using the given Db
func Transfer(db StateDB, sender, recipient common.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}
