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

package network

import (
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/notify"
	"encoding/json"
)

// Message
// msgType 0
// msgType 1
type txMessage struct {
	MsgType byte   `json:"type"`
	Id      uint64 `json:"id"`
	Data    []byte `json:"data"`
}

type TxConn struct {
	baseConn
}

func (conn *TxConn) Init(ipPort string) {
	conn.rcv = func(body []byte) {
		var data txMessage
		err := json.Unmarshal(body, &data)
		if nil != err || nil == data.Data {
			txRcvLogger.Errorf("fail to unmarshal tx json, err: %s, body: %s", err, string(body))
			conn.afterReconnected()
			return
		}

		var msg notify.ClientTransactionMessage
		err = json.Unmarshal(data.Data, &msg)
		if nil == err {
			conn.logger.Debugf("rcv tx. hash: %s, nonce: %d", msg.Tx.Hash.String(), msg.Nonce)
			middleware.DataChannel.GetRcvedTx() <- &msg
		}
		conn.ack(data.Id)
	}

	conn.afterReconnected = func() {
		conn.ack(middleware.AccountDBManagerInstance.GetThreshold())
	}

	conn.init(ipPort, "/tx", txRcvLogger)
	conn.afterReconnected()
}

func (conn *TxConn) ack(id uint64) {
	var m txMessage
	m.MsgType = 1
	m.Id = id + 1

	data, _ := json.Marshal(m)
	txRcvLogger.Warnf("sent ack to %s, data: %s", conn.url, string(data))
	conn.sendTextChan <- data
}
