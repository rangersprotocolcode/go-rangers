package eth_rpc

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/notify"
	"encoding/json"
	"testing"
)

func TestHandler(t *testing.T) {
	common.InitConf("1.ini")
	middleware.InitMiddleware("")
	InitEthMsgHandler()

	//p := []string{"0x1111"}
	//b, _ := json.Marshal(p)
	//msg := notify.ETHRPCMessage{Id: 1, RequestId: 100, Method: "eth_sendRawTransaction", Params: b}
	//handler.process(&msg)

	p := []string{}
	b, _ := json.Marshal(p)
	getChainIdMsg := notify.ETHRPCMessage{Id: 1, RequestId: 100, Method: "eth_chainId", Params: b}
	handler.process(&getChainIdMsg)
}
