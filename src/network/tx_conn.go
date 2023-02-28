package network

import (
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/log"
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

func (conn *TxConn) Init(ipPort string, logger log.Logger) {
	conn.rcv = func(body []byte) {
		var data txMessage
		err := json.Unmarshal(body, &data)
		if nil != err || nil == data.Data {
			logger.Errorf("fail to unmarshal tx json,err: %s, body: %s", err, string(body))
			return
		}

		var msg notify.ClientTransactionMessage
		err = json.Unmarshal(data.Data, &msg)
		if nil == err {
			conn.logger.Debugf("rcv tx. hash: %s, nonce: %d", msg.Tx.Hash.String(), msg.Nonce)
			middleware.DataChannel.GetRcvedTx() <- &msg
		}
	}

	conn.afterReconnected = func() {
		var m txMessage
		m.MsgType = 0
		m.Id = middleware.AccountDBManagerInstance.GetThreshold()

		data, _ := json.Marshal(m)
		logger.Warnf("sent to %s, data: %s", conn.url, string(data))
		conn.sendTextChan <- data
	}

	conn.init(ipPort, "/tx", logger)
	conn.afterReconnected()
}
