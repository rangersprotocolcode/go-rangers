package eth_rpc

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/notify"
	"encoding/json"
	"testing"
)

func TestSendRawTransaction(t *testing.T) {
	common.InitConf("1.ini")
	middleware.InitMiddleware("")
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
	p := []string{"0xf8692501827b0c945643ede07854c47a5bb482e6910cd716aeb4b57e881bc16d674ec8000080824a92a0b425452a506afdb4e4958333a2aa513d67baecb95cfa45647d40b377ca613ac9a020e737bd2bfa94c2d801ff33ea3a12b6b4743839f32de34d9fefc501a7035621"}
	b, _ := json.Marshal(p)
	piece := notify.ETHRPCPiece{Id: 1, Method: "eth_sendRawTransaction", Params: b}
	msg := notify.ETHRPCMessage{RequestId: 100, Message: piece}
	handler.process(&msg)
	handler.process(&msg)
}
