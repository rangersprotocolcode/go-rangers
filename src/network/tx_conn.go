package network

import (
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/notify"
	"encoding/json"
)

// Message
// msgType 0: 从某个id开始发送，无需ack确认
// msgType 1: 获取某个id
// msgType 2: ack获取某个id，发送下一个。客户端状态，待实现
type txMessage struct {
	MsgType byte   `json:"type"`
	Id      uint64 `json:"id"`
	Data    []byte `json:"data"`
}

type TxConn struct {
	baseConn
}

// Init txConn 处理 /api/writer
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
