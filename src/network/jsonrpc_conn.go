package network

import (
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/notify"
	"encoding/json"
	"github.com/gorilla/websocket"
)

type JSONRPCConn struct {
	baseConn
}

func (conn *JSONRPCConn) Init(ipPort string, logger log.Logger) {
	// worker 链接加大发送队列长度
	conn.rcvSize = 1
	conn.rcv = func(body []byte) {
		if nil == body || 0 == len(body) {
			logger.Errorf("fail to get body from jsonrpcConn, empty body")
			return
		}

		message := notify.ETHRPCMessage{}
		err := json.Unmarshal(body, &message)
		if err == nil {
			conn.logger.Debugf("get body from jsonrpcConn, bodyHex: %s, publishing", string(body))
			notify.BUS.Publish(notify.ETHRPC, &message)
		} else {
			message := notify.ETHRPCBatchMessage{}
			err := json.Unmarshal(body, &message)
			if err == nil {
				conn.logger.Debugf("get body from jsonrpcConn, bodyHex: %s, publishing", string(body))
				notify.BUS.Publish(notify.ETHRPC, &message)
			} else {
				conn.logger.Errorf("fail to get body from jsonrpcConn, bodyHex: %s,err:%s", string(body), err.Error())
			}
		}
	}

	conn.init(ipPort, "/srv/jsonrpc", logger)
}

func (conn *JSONRPCConn) Send(msg string, sessionId, requestId uint64) {
	wsConn := conn.getConn()
	if nil == wsConn {
		conn.logger.Errorf("Send TextMessage msg error. no jsonrpc connection")
		return
	}

	result := jsonrpcResult{Msg: msg, SessionId: sessionId, RequestId: requestId}
	data, _ := json.Marshal(result)
	conn.logger.Debugf("sending jsonrpc: %s", string(data))
	err := wsConn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		conn.logger.Errorf("Send TextMessage msg, error:%s", err.Error())
		conn.closeConn()
	}
}

type jsonrpcResult struct {
	Msg       string `json:"msg"`
	SessionId uint64 `json:"session_id"`
	RequestId uint64 `json:"request_id"`
}
