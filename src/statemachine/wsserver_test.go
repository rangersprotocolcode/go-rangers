package statemachine

import (
	"com.tuntun.rocket/node/src/common"
	"testing"
)

func TestWsServer_Start(t *testing.T) {
	common.InitConf("/Users/daijia/go/src/x/deploy/daily/x1.ini")
	ws := newWSServer("wstest")
	err := ws.Start()
	if err != nil {
		t.Fatalf("fail to start ws. %s", err.Error())
	}

	//time.Sleep(1000 * time.Minute)
}
