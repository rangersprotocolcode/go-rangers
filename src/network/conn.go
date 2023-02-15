// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package network

import (
	"bytes"
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/notify"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/utility"
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
	// ws协议头大小
	protocolHeaderSize = 28

	// 默认等待队列大小
	defaultRcvSize  = 10000
	defaultSendSize = 10000

	// ws读写缓存
	defaultBufferSize = 1024 * 1024 * 16

	//追加在HEADER之后的网络id的大小
	netIdSize = 32
)

type wsHeader struct {
	method   []byte
	sourceId uint64
	targetId uint64
	nonce    uint64
}

type baseConn struct {
	// 链接
	url      string
	path     string
	conn     *websocket.Conn
	connLock sync.Mutex

	// 读写缓冲区
	rcvChan  chan []byte
	sendChan chan []byte

	// 读写缓冲区大小
	rcvSize  int
	sendSize int

	// 处理消息的业务逻辑
	doRcv func(wsHeader wsHeader, msg []byte)

	//断线重连后的处理
	afterReconnected func()

	// 处理[]byte原始消息，用于流控
	rcv func(msg []byte)

	// 判断是否要发送，用于流控
	isSend func(method []byte, target uint64, msg []byte, nonce uint64) bool

	logger log.Logger

	rcvCount, sendCount uint64
}

// 根据url初始化
func (base *baseConn) init(ipPort, path string, logger log.Logger) {
	base.logger = logger
	base.path = path

	ipPortString, _ := url.QueryUnescape(ipPort)
	base.url = fmt.Sprintf("%s%s", ipPortString, path)
	base.conn = base.getWSConn()
	base.connLock = sync.Mutex{}

	// 初始化读写缓存
	if 0 == base.rcvSize {
		base.rcvSize = defaultRcvSize
	}
	if 0 == base.sendSize {
		base.sendSize = defaultSendSize
	}
	base.rcvChan = make(chan []byte, base.rcvSize)
	base.sendChan = make(chan []byte, base.sendSize)
	base.rcvCount = 0
	base.sendCount = 0

	// 开启goroutine
	base.start()
}

// 建立ws连接
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

// 开工
func (base *baseConn) start() {
	go base.receiveMessage()
	go base.loop()
	go base.logChannel()
}

// 定时检查channel堆积情况
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
		p2pLogger.Errorf("rcv: %dKB, sent: %dKB", rcvResult, sentResult)
	}
}

// 调度器
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
				base.logger.Errorf("Send binary msg error:%s", err.Error())
				base.closeConn()
			}
		}
	}
}

// 读消息
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
			base.rcv(message)
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
		go base.afterReconnected()
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

// 发送消息
func (base *baseConn) send(method []byte, target uint64, msg []byte, nonce uint64) {
	header := wsHeader{method: method, nonce: nonce, targetId: target}

	if base.isSend != nil && !base.isSend(method, target, msg, nonce) {
		base.logger.Errorf("%s did not accept send. wsHeader: %v, length: %d", base.path, header, len(msg))
	}

	base.sendChan <- base.loadMsg(header, msg)
	base.logger.Debugf("send message. wsHeader: %v, body length: %d", header, len(msg))
}

//新的单播接口使用
func (base *baseConn) unicast(method []byte, strangerId []byte, msg []byte, nonce uint64) {
	byteArray := make([]byte, protocolHeaderSize+netIdSize+len(msg))
	copy(byteArray[0:4], method)
	copy(byteArray[20:28], utility.UInt64ToByte(nonce))
	copy(byteArray[protocolHeaderSize:protocolHeaderSize+netIdSize], strangerId)
	copy(byteArray[protocolHeaderSize+netIdSize:], msg)

	//todo 这里流控方法的参数不一致，暂不使用流控
	base.sendChan <- byteArray
	base.logger.Debugf("unicast message. strangerId:%v,msg:%v,byte: %v", strangerId, msg, byteArray)
}

// 构建网络消息
func (base *baseConn) loadMsg(header wsHeader, body []byte) []byte {
	h := base.headerToBytes(header)

	message := make([]byte, protocolHeaderSize+len(body))
	copy(message[:protocolHeaderSize], h[:])
	copy(message[protocolHeaderSize:], body)
	return message
}

// 解包消息
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

// 处理客户端的read/write请求
var (
	methodNotify, _          = hex.DecodeString("20000000")
	methodNotifyBroadcast, _ = hex.DecodeString("20000001")
	methodNotifyGroup, _     = hex.DecodeString("20000002")
	methodNotifyInit, _      = hex.DecodeString("20000003")

	methodClientReader, _  = hex.DecodeString("00000001")
	methodClientJSONRpc, _ = hex.DecodeString("00000000")
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

		if bytes.Equal(wsHeader.method, methodClientReader) {
			clientConn.handleClientMessage(body, strconv.FormatUint(wsHeader.sourceId, 10), wsHeader.nonce)
			return
		}

		if bytes.Equal(wsHeader.method, methodClientJSONRpc) {
			clientConn.handleJSONClientMessage(body, strconv.FormatUint(wsHeader.sourceId, 10), wsHeader.nonce)
			return
		}

		msg := fmt.Sprintf("%s received wrong method. wsHeader: %v", clientConn.path, wsHeader)
		clientConn.logger.Error(msg)
	}

	//流控方法
	clientConn.rcv = func(msg []byte) {
		if len(clientConn.rcvChan) == clientConn.rcvSize {
			clientConn.logger.Errorf("client rcvChan full, remove it, msg size: %d", len(msg))
			return
		}

		clientConn.rcvChan <- msg
	}

	// 流控方法
	clientConn.isSend = func(method []byte, target uint64, msg []byte, nonce uint64) bool {
		return len(clientConn.sendChan) < clientConn.sendSize
	}

	clientConn.init(ipPort, path, logger)

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

func (clientConn *ClientConn) handleJSONClientMessage(body []byte, userId string, nonce uint64) {
	message := notify.ETHRPCMessage{}
	err := json.Unmarshal(body, &message.Message)
	if err == nil {
		clientConn.logger.Debugf("get body from jsonrpcConn, bodyHex: %s, publishing", string(body))
		message.SessionId = userId
		message.RequestId = nonce
		notify.BUS.Publish(notify.ClientETHRPC, &message)
		return
	}

	messageBatch := notify.ETHRPCBatchMessage{}
	err = json.Unmarshal(body, &messageBatch.Message)
	if err == nil {
		messageBatch.SessionId = userId
		messageBatch.RequestId = nonce
		clientConn.logger.Debugf("get body from jsonrpcConn, bodyHex: %s, publishing", string(body))
		notify.BUS.Publish(notify.ClientETHRPC, &messageBatch)
		return
	}

	// for error response
	wrong := notify.ETHRPCWrongMessage{}
	wrong.Sid = userId
	wrong.Rid = nonce
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
