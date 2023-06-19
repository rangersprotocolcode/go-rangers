// Copyright 2020 The RangersProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RangersProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RangersProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package eth_rpc

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/notify"
	"encoding/json"
	"errors"
	"fmt"
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
	p := []string{"0xf8a58085174876e800830186a08080b853604580600e600039806000f350fe7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe03601600081602082378035828234f58015156039578182fd5b8082525050506014600cf31ba02222222222222222222222222222222222222222222222222222222222222222a02222222222222222222222222222222222222222222222222222222222222222"}
	b, _ := json.Marshal(p)
	piece := notify.ETHRPCPiece{Id: 1, Method: "eth_sendRawTransaction", Params: b}
	msg := notify.ETHRPCMessage{GateNonce: 100, Message: piece}
	handler.process(&msg)
}

func TestRevertError(t *testing.T) {

	e := getErr()
	ec, ok := e.(Error)
	if ok {
		fmt.Printf("code:%d\n", ec.ErrorCode())
	}
	de, ok := e.(DataError)
	if ok {
		fmt.Printf("msg:%s\n", de.ErrorData())
	}

}

func getErr() error {
	err := errors.New("123")
	e := revertError{err, "reason here"}
	return &e
}
