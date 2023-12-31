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
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package network

import (
	"bytes"
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/middleware/log"
	"com.tuntun.rangers/node/src/middleware/notify"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/service"
	"com.tuntun.rangers/node/src/utility"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	protocolHeaderSize = 28

	defaultRcvSize  = 10000
	defaultSendSize = 10000

	defaultBufferSize = 1024 * 1024 * 16

	netIdSize = 32
)

type wsHeader struct {
	method   []byte
	sourceId uint64
	targetId uint64
	nonce    uint64
}

type baseConn struct {
	url      string
	path     string
	conn     *websocket.Conn
	connLock sync.Mutex

	rcvChan      chan []byte
	sendChan     chan []byte
	sendTextChan chan []byte

	rcvSize  int
	sendSize int

	doRcv func(wsHeader wsHeader, msg []byte)

	afterReconnected func()

	rcv func(msg []byte)

	isSend func(method []byte, target uint64, msg []byte, nonce uint64) bool

	logger log.Logger

	rcvCount, sendCount uint64
}

func (base *baseConn) init(ipPort, path string, logger log.Logger) {
	base.connLock = sync.Mutex{}

	base.logger = logger
	base.path = path

	ipPortString, _ := url.QueryUnescape(ipPort)
	base.url = fmt.Sprintf("%s%s", ipPortString, path)
	base.conn = base.getWSConn()

	if 0 == base.rcvSize {
		base.rcvSize = defaultRcvSize
	}
	if 0 == base.sendSize {
		base.sendSize = defaultSendSize
	}
	base.rcvChan = make(chan []byte, base.rcvSize)
	base.sendChan = make(chan []byte, base.sendSize)
	base.sendTextChan = make(chan []byte, defaultSendSize)
	base.rcvCount = 0
	base.sendCount = 0

	base.start()
}

func (base *baseConn) getWSConn() *websocket.Conn {
	base.logger.Debugf("connecting to %s", base.url)
	d := websocket.Dialer{ReadBufferSize: defaultBufferSize, WriteBufferSize: defaultBufferSize}
	conn, _, err := d.Dial(base.url, nil)
	if err != nil {
		base.logger.Errorf("Dial to " + base.url + " err:" + err.Error())
		time.Sleep(100 * time.Millisecond)
		return nil
	}

	base.logger.Debugf("connected to %s", base.url)
	return conn
}

func (base *baseConn) start() {
	go base.receiveMessage()
	go base.loop()
	go base.logChannel()
}

func (base *baseConn) logChannel() {
	for range time.Tick(time.Millisecond * 500) {
		rcv, send := len(base.rcvChan), len(base.sendChan)
		if rcv > 0 || send > 0 {
			p2pLogger.Errorf("%s channel size. receive: %d, send: %d", base.path, rcv, send)
		}

		rcvResult := atomic.LoadUint64(&base.rcvCount) * 2 / 1000
		sentResult := atomic.LoadUint64(&base.sendCount) * 2 / 1000
		atomic.StoreUint64(&base.rcvCount, 0)
		atomic.StoreUint64(&base.sendCount, 0)
		p2pLogger.Errorf("%s, rcv: %dKB/s, sent: %dKB/s", base.path, rcvResult, sentResult)
	}
}

func (base *baseConn) loop() {
	for {
		select {
		case message := <-base.rcvChan:
			atomic.AddUint64(&base.rcvCount, uint64(len(message)))

			// goroutine read process
			if base.doRcv != nil {
				header, msg := base.unloadMsg(message)
				go base.doRcv(header, msg)
			}

		case message := <-base.sendChan:
			atomic.AddUint64(&base.sendCount, uint64(len(message)))

			conn := base.getConn()
			if nil == conn {
				continue
			}

			err := conn.WriteMessage(websocket.BinaryMessage, message)
			if err != nil {
				base.logger.Errorf("send binary msg error:%s", err.Error())
				base.closeConn()
			}
		case message := <-base.sendTextChan:
			atomic.AddUint64(&base.sendCount, uint64(len(message)))

			conn := base.getConn()
			if nil != conn {
				err := conn.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					base.logger.Errorf("send tx msg error:%s", err.Error())
					base.closeConn()
				}
			}

		}
	}
}

func (base *baseConn) receiveMessage() {
	for {
		conn := base.getConn()
		if nil == conn {
			continue
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			base.logger.Errorf("%s Rcv msg error:%s", base.url, err.Error())
			base.closeConn()
			continue
		}

		if base.rcv == nil {
			base.rcvChan <- message
		} else {
			go base.rcv(message)
		}

	}
}

func (base *baseConn) getConn() *websocket.Conn {
	if nil != base.conn {
		return base.conn
	}

	base.connLock.Lock()
	defer base.connLock.Unlock()

	if nil != base.conn {
		return base.conn
	}
	base.conn = base.getWSConn()
	if nil != base.conn && nil != base.afterReconnected {
		base.afterReconnected()
	}
	return base.conn
}

func (base *baseConn) closeConn() {
	base.connLock.Lock()
	defer base.connLock.Unlock()

	if base.conn != nil {
		base.conn.Close()
	}
	base.conn = nil
}

func (base *baseConn) send(method []byte, target uint64, msg []byte, nonce uint64) {
	header := wsHeader{method: method, nonce: nonce, targetId: target}

	if base.isSend != nil && !base.isSend(method, target, msg, nonce) {
		base.logger.Errorf("%s did not accept send. wsHeader: %v, length: %d", base.path, header, len(msg))
	}

	base.sendChan <- base.loadMsg(header, msg)
	base.logger.Debugf("send message. wsHeader: %v, body length: %d", header, len(msg))
}

func (base *baseConn) unicast(method []byte, strangerId []byte, msg []byte, nonce uint64) {
	byteArray := make([]byte, protocolHeaderSize+netIdSize+len(msg))
	copy(byteArray[0:4], method)
	copy(byteArray[20:28], utility.UInt64ToByte(nonce))
	copy(byteArray[protocolHeaderSize:protocolHeaderSize+netIdSize], strangerId)
	copy(byteArray[protocolHeaderSize+netIdSize:], msg)

	base.sendChan <- byteArray
	base.logger.Debugf("unicast message. strangerId:%s, length: %d", common.ToHex(strangerId), len(byteArray))
}

func (base *baseConn) loadMsg(header wsHeader, body []byte) []byte {
	h := base.headerToBytes(header)

	message := make([]byte, protocolHeaderSize+len(body))
	copy(message[:protocolHeaderSize], h[:])
	copy(message[protocolHeaderSize:], body)
	return message
}

func (base *baseConn) unloadMsg(m []byte) (header wsHeader, body []byte) {
	if len(m) < protocolHeaderSize {
		return header, nil
	}

	header = base.bytesToHeader(m[:protocolHeaderSize])
	body = m[protocolHeaderSize:]

	return
}

func (base *baseConn) headerToBytes(h wsHeader) []byte {
	byteArray := make([]byte, protocolHeaderSize)
	copy(byteArray[0:4], h.method)
	//copy(byteArray[4:12], utility.UInt64ToByte(h.targetId))
	copy(byteArray[12:20], utility.UInt64ToByte(h.targetId))
	copy(byteArray[20:28], utility.UInt64ToByte(h.nonce))
	return byteArray
}

func (base *baseConn) bytesToHeader(b []byte) wsHeader {
	header := wsHeader{}
	header.method = b[0:4]
	header.sourceId = utility.ByteToUInt64(b[4:12])
	header.targetId = utility.ByteToUInt64(b[12:20]) // reader nonce
	header.nonce = utility.ByteToUInt64(b[20:])
	return header
}

func (base *baseConn) generateTarget(targetId string) (uint64, error) {
	target, err := strconv.ParseUint(targetId, 10, 64)
	if err != nil {
		base.logger.Errorf("Parse target id %s error:%s", targetId, err.Error())
		return 0, err
	}

	return target, nil
}

var (
	methodNotify, _          = hex.DecodeString("20000000")
	methodNotifyBroadcast, _ = hex.DecodeString("20000001")
	methodNotifyGroup, _     = hex.DecodeString("20000002")
	methodNotifyInit, _      = hex.DecodeString("20000003")

	methodClientReader, _  = hex.DecodeString("00000001")
	methodClientJSONRpc, _ = hex.DecodeString("00000000")
	methodHandShake, _     = hex.DecodeString("000007e9")
	methodAck, _           = hex.DecodeString("00000003")
)

type ClientConn struct {
	baseConn

	notifyNonce uint64
	nonceLock   sync.Mutex
}

func (clientConn *ClientConn) Send(targetId string, method, msg []byte, nonce uint64) {
	target, err := clientConn.generateTarget(targetId)
	if err != nil {
		return
	}

	clientConn.send(method, target, msg, nonce)
}

func (clientConn *ClientConn) Init(ipPort, path string, logger log.Logger) {
	clientConn.nonceLock = sync.Mutex{}
	clientConn.notifyNonce = 0

	clientConn.doRcv = func(wsHeader wsHeader, body []byte) {
		clientConn.logger.Debugf("received. header: %s, from: %d, nonce: %d, bodyLength: %d", common.ToHex(wsHeader.method), wsHeader.sourceId, wsHeader.nonce, len(body))

		// ws /api/reader
		if bytes.Equal(wsHeader.method, methodClientReader) {
			clientConn.handleClientMessage(body, strconv.FormatUint(wsHeader.sourceId, 10), wsHeader.nonce)
			return
		}

		// http client: /api/jsonrpc
		if bytes.Equal(wsHeader.method, methodClientJSONRpc) {
			clientConn.handleJSONClientMessage(body, strconv.FormatUint(wsHeader.sourceId, 10), wsHeader.nonce)
			return
		}

		msg := fmt.Sprintf("%s received wrong method. wsHeader: %v", clientConn.path, wsHeader)
		clientConn.logger.Error(msg)
	}

	clientConn.rcv = func(msg []byte) {
		if len(clientConn.rcvChan) == clientConn.rcvSize {
			clientConn.logger.Errorf("client rcvChan full, remove it, msg size: %d", len(msg))
			return
		}

		clientConn.rcvChan <- msg
	}

	clientConn.isSend = func(method []byte, target uint64, msg []byte, nonce uint64) bool {
		return len(clientConn.sendChan) < clientConn.sendSize
	}

	clientConn.afterReconnected = func() {
		nonce := uint64(0)
		if nil != service.GetTransactionPool() {
			nonce = service.GetTransactionPool().GetGateNonce()
		}

		header := wsHeader{method: methodHandShake, nonce: nonce}
		bytes := clientConn.headerToBytes(header)
		err := clientConn.conn.WriteMessage(websocket.BinaryMessage, bytes)
		if nil != err {
			p2pLogger.Errorf("afterReconnected. err: %s", err)
		}
	}
	clientConn.init(ipPort, path, logger)
	if nil != clientConn.conn {
		clientConn.afterReconnected()
	}
}

func (clientConn *ClientConn) handleClientMessage(body []byte, userId string, nonce uint64) {
	var txJson types.TxJson
	err := json.Unmarshal(body, &txJson)
	if nil != err {
		msg := fmt.Sprintf("handleClientMessage json unmarshal client message error:%s", err.Error())
		clientConn.logger.Errorf(msg)
		return
	}

	tx := txJson.ToTransaction()
	tx.RequestId = nonce
	clientConn.logger.Debugf("Rcv event: %s from client.Tx info:%s", notify.ClientTransactionRead, tx.ToTxJson().ToString())
	msg := notify.ClientTransactionMessage{Tx: tx, UserId: userId, Nonce: nonce}
	notify.BUS.Publish(notify.ClientTransactionRead, &msg)
}

func (clientConn *ClientConn) handleJSONClientMessage(body []byte, userId string, gateNonce uint64) {
	message := notify.ETHRPCMessage{}
	err := json.Unmarshal(body, &message.Message)
	if err == nil {
		clientConn.logger.Debugf("get body from jsonrpcConn, bodyHex: %s, publishing", string(body))
		message.SessionId = userId
		message.GateNonce = gateNonce
		notify.BUS.Publish(notify.ClientETHRPC, &message)
		return
	}

	messageBatch := notify.ETHRPCBatchMessage{}
	err = json.Unmarshal(body, &messageBatch.Message)
	if err == nil {
		messageBatch.SessionId = userId
		messageBatch.GateNonce = gateNonce
		clientConn.logger.Debugf("get body from jsonrpcConn, bodyHex: %s, publishing", string(body))
		notify.BUS.Publish(notify.ClientETHRPC, &messageBatch)
		return
	}

	// for error response
	wrong := notify.ETHRPCWrongMessage{}
	wrong.Sid = userId
	wrong.Rid = gateNonce
	notify.BUS.Publish(notify.ClientETHRPC, &wrong)

	clientConn.logger.Errorf("fail to get body from jsonrpcConn, bodyHex: %s,err:%s", string(body), err.Error())
}

func (clientConn *ClientConn) Notify(isUniCast bool, gameId string, userId string, msg string) {
	if 0 == len(gameId) {
		return
	}

	method := methodNotify
	if !isUniCast {
		if 0 == len(userId) {
			method = methodNotifyBroadcast
		} else {
			method = methodNotifyGroup
		}
	}

	clientConn.nonceLock.Lock()
	defer clientConn.nonceLock.Unlock()

	clientConn.notifyNonce = clientConn.notifyNonce + 1
	notifyId := clientConn.generateNotifyId(gameId, userId)

	clientConn.send(method, notifyId, []byte(msg), clientConn.notifyNonce)
}

func (clientConn *ClientConn) generateNotifyId(gameId string, userId string) uint64 {
	data := []byte(gameId)
	if 0 != len(userId) {
		data = append(data, []byte(userId)...)
	}

	md5Result := md5.Sum(data)
	idBytes := md5Result[4:12]
	return binary.BigEndian.Uint64(idBytes)
}
