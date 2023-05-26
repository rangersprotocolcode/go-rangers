package network

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware"
	"os"
	"testing"
	"time"
)

func TestTxConn_Init(t *testing.T) {
	os.RemoveAll("logs")
	os.RemoveAll("1.ini")
	os.RemoveAll("storage0")

	common.Init(0, "1.ini", "dev")
	middleware.InitMiddleware()

	var tx TxConn
	tx.Init("ws://192.168.2.14:8888")

	time.Sleep(10 * time.Hour)
}
