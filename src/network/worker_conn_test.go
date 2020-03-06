package network

import (
	"testing"
	"x/src/middleware/log"
	"hash/fnv"
	"fmt"
	"strconv"
)

func TestWorkerConn_Init(t *testing.T) {
	var worker1, worker2 WorkerConn
	logger := log.GetLoggerByIndex(log.P2PLogConfig, "1")

	worker1.Init("39.104.113.9", "1", nil, logger)
	worker2.Init("39.104.113.9", "2", nil, logger)

}

func TestGenTargetForgroup(t *testing.T) {
	groupId := "0x2a63497b8b48bc85ae6f61576d4a2988e7b71e1c02898ea2a02ead17f076bf92"
	hash64 := fnv.New64()
	hash64.Write([]byte(groupId))

	idInt := hash64.Sum64()
	fmt.Printf("%v\n", idInt)

	s16 := strconv.FormatUint(idInt, 16) //10 yo 16
	fmt.Printf("%v\n", s16)
}
