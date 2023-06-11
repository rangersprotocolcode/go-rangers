package eth_rpc

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/notify"
	"encoding/json"
	"os"
	"testing"
)

func TestSendRawTransaction(t *testing.T) {
	defer func() {
		os.RemoveAll("logs")
		os.RemoveAll("1.ini")
		os.RemoveAll("storage0")
	}()
	common.Init(0, "1.ini", "dev")
	middleware.InitMiddleware()
	InitEthMsgHandler()

	/**


	  var rawTx = {
	            nonce: '0x00',
	            gasPrice: '0x09184e72a000',
	            gasLimit: '0x2710',
	            to: '0x407d73d8a49eeb85d32cf465507dd71d507100c1',
	            value: '0x001234',(4660)
	            data: '0x7f7465737432000000000000000000000000000000000000000000000000000000600057'
	        }
	*/
	//p := []string{"0xf88b808609184e72a00082271094407d73d8a49eeb85d32cf465507dd71d507100c1821234a47f746573743200000000000000000000000000000000000000000000000000000060005725a0468be9f58932931d9c7256b0937496d24035dc4bcc7c6223bb85c0b965035df6a067d42b7cf2e56d50082499754f4f4a18c050ac50a41f581462e3786392cadd14"}
	p := []string{"0xf8640601832dc6c0942f4f09b722a6e5b77be17c9a99c785fa7035a09f8203e880824a5ba0458a14eed6c67985231ac2f61e540ff4f1b3facba6aed5e81b1da3f011b57e6fa006308ce6da36de5037f3ad10ce26b7e54cdf43791afda824871cc0989e931938"}
	b, _ := json.Marshal(p)
	piece := notify.ETHRPCPiece{Id: 1, Method: "eth_sendRawTransaction", Params: b}
	msg := notify.ETHRPCMessage{GateNonce: 100, Message: piece}
	handler.process(&msg)
}
