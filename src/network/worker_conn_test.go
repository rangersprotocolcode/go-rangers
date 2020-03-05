package network

import (
	"testing"
	"x/src/middleware/log"
	"hash/fnv"
	"fmt"
)

func TestWorkerConn_Init(t *testing.T) {
	var worker1, worker2 WorkerConn
	logger := log.GetLoggerByIndex(log.P2PLogConfig, "1")

	worker1.Init("39.104.113.9", "1", nil, logger)
	worker2.Init("39.104.113.9", "2", nil, logger)

}


func TestGenTargetForgroup(t *testing.T) {
	groupId:="0x2d4f9bd77fb95cdbe857b615c7bd2e21f20d3bd1e73974a7943d11a289bc3ac4"
	hash64 := fnv.New64()
	hash64.Write([]byte(groupId))

	fmt.Printf("%v\n",hash64.Sum64())

}