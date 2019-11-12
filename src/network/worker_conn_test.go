package network

import (
	"testing"
	"x/src/middleware/log"
)

func TestWorkerConn_Init(t *testing.T) {
	var worker1, worker2 WorkerConn
	logger := log.GetLoggerByIndex(log.P2PLogConfig, "1")

	worker1.Init("39.104.113.9", "1", nil, logger)
	worker2.Init("39.104.113.9", "2", nil, logger)

}
