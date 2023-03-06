package network

import (
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/log"
	"os"
	"testing"
	"time"
)

func TestTxConn_Init(t *testing.T) {
	os.RemoveAll("logs")
	middleware.InitMiddleware()

	p2pLogger = log.GetLoggerByIndex(log.P2PLogConfig, "1")
	logger := log.GetLoggerByIndex(log.TxRcvLogConfig, "1")
	var tx TxConn
	tx.Init("ws://192.168.2.14:7777", logger)

	time.Sleep(10 * time.Hour)
}
