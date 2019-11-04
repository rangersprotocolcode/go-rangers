package network

import (
	"github.com/gorilla/websocket"
	"x/src/utility"
	"net/url"
	"strconv"
	"encoding/hex"
	"bytes"
	"x/src/middleware/types"
	"encoding/json"
	"x/src/middleware/notify"
	"sync"
	"crypto/md5"
	"encoding/binary"
	"x/src/middleware/log"
)

const (
	// ws协议头大小
	protocolHeaderSize = 28
	// 读写channel数量
	channelSize = 1000
	// ws读写缓存
	bufferSize = 1024 * 1024 * 32
)

var methodCodeClientReader, _ = hex.DecodeString("60000000")
var methodCodeClientWriter, _ = hex.DecodeString("60000001")

var methodSendToCoinConnector, _ = hex.DecodeString("30000003")
var methodRcvFromCoinConnector, _ = hex.DecodeString("30000002")

var methodNotify, _ = hex.DecodeString("20000000")
var methodNotifyBroadcast, _ = hex.DecodeString("20000001")
var methodNotifyGroup, _ = hex.DecodeString("20000002")

type wsHeader struct {
	method   []byte
	sourceId uint64
	targetId uint64
	nonce    uint64
}

type baseConn struct {
	path string
	conn *websocket.Conn

	sendChan chan []byte
	rcvChan  chan []byte

	// 收到消息后的回调
	doRcv func(wsHeader wsHeader, msg []byte)

	logger log.Logger
}

// 根据url初始化
func (base *baseConn) init(ipPort, path string, logger log.Logger) {
	base.logger = logger

	url := url.URL{Scheme: "ws", Host: ipPort, Path: path}
	base.logger.Debugf("connecting to %s", url.String())

	d := websocket.Dialer{ReadBufferSize: bufferSize, WriteBufferSize: bufferSize,}
	conn, _, err := d.Dial(url.String(), nil)
	if err != nil {
		panic("Dial to" + url.String() + " err:" + err.Error())
	}

	base.path = path
	base.conn = conn
	base.sendChan = make(chan []byte, channelSize)
	base.rcvChan = make(chan []byte, channelSize)

	base.start()
}

// 开工
func (base *baseConn) start() {
	go base.receiveMessage()
	go base.loop()
}

// 调度器
func (base *baseConn) loop() {
	for {
		select {
		case message := <-base.rcvChan:
			// goroutine read process
			if base.doRcv != nil {
				header, msg := base.unloadMsg(message)
				go base.doRcv(header, msg)
			}

		case message := <-base.sendChan:
			err := base.conn.WriteMessage(websocket.BinaryMessage, message)
			if err != nil {
				base.logger.Errorf("Send binary msg error:%s", err.Error())
			}
		}
	}
}

// 读消息
func (base *baseConn) receiveMessage() {
	for {
		_, message, err := base.conn.ReadMessage()
		if err != nil {
			base.logger.Errorf("Rcv msg error:%s", err.Error())
			continue
		}

		base.rcvChan <- message
	}
}

// 发送消息
func (base *baseConn) send(method []byte, target uint64, msg []byte, nonce uint64) {
	header := wsHeader{method: method, nonce: nonce, targetId: target}
	base.sendChan <- base.loadMsg(header, msg)
	base.logger.Debugf("send message. wsHeader: %v, length: %d", header, len(msg))
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
	byte := make([]byte, protocolHeaderSize)
	copy(byte[0:4], h.method)
	copy(byte[12:20], utility.UInt64ToByte(h.targetId))
	copy(byte[20:28], utility.UInt64ToByte(h.nonce))
	return byte
}

func (base *baseConn) bytesToHeader(b []byte) wsHeader {
	header := wsHeader{}
	header.method = b[0:4]
	header.sourceId = utility.ByteToUInt64(b[4:12])
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
type ClientConn struct {
	baseConn
	method []byte
	event  string

	notifyNonce uint64
	nonceLock   sync.Mutex
}

func (clientConn *ClientConn) Send(targetId string, msg []byte, nonce uint64) {
	target, err := clientConn.generateTarget(targetId)
	if err != nil {
		return
	}

	clientConn.send(clientConn.method, target, msg, nonce)
}

func (clientConn *ClientConn) Init(ipPort, path, event string, method []byte, logger log.Logger) {
	clientConn.method = method
	clientConn.nonceLock = sync.Mutex{}
	clientConn.notifyNonce = 0

	clientConn.doRcv = func(wsHeader wsHeader, body []byte) {
		if !bytes.Equal(wsHeader.method, clientConn.method) {
			clientConn.logger.Error("received wrong method: %v", method)
			return
		}

		clientConn.handleClientMessage(body, strconv.FormatUint(wsHeader.sourceId, 10), wsHeader.nonce, event)
	}

	clientConn.init(ipPort, path, logger)
}

func (clientConn *ClientConn) handleClientMessage(body []byte, userId string, nonce uint64, event string) {
	var txJson types.TxJson
	err := json.Unmarshal(body, &txJson)
	if nil != err {
		clientConn.logger.Errorf("Json unmarshal client message error:%s", err.Error())
		return
	}
	tx := txJson.ToTransaction()
	clientConn.logger.Debugf("Rcv from client.Tx info:%s", txJson.ToString())

	msg := notify.ClientTransactionMessage{Tx: tx, UserId: userId, Nonce: nonce}
	notify.BUS.Publish(event, &msg)
}

func (clientConn *ClientConn) Notify(isunicast bool, gameId string, userid string, msg string) {
	if 0 == len(gameId) {
		return
	}

	method := methodNotify
	if !isunicast {
		if 0 == len(userid) {
			method = methodNotifyBroadcast
		} else {
			method = methodNotifyGroup
		}
	}

	clientConn.nonceLock.Lock()
	defer clientConn.nonceLock.Unlock()

	clientConn.notifyNonce = clientConn.notifyNonce + 1
	notifyId := clientConn.generateNotifyId(gameId, userid)

	clientConn.send(method, notifyId, []byte(msg), clientConn.notifyNonce)
}

func (clientConn *ClientConn) generateNotifyId(gameId string, userId string) uint64 {
	data := []byte(gameId)
	if 0 != len(userId) {
		data = append(data, []byte(userId)...)
	}

	md5Result := md5.Sum(data)
	idBytes := md5Result[4:12]
	return uint64(binary.BigEndian.Uint64(idBytes))
}

// 处理coiner的请求
type CoinerConn struct {
	baseConn
}

func (coinerConn *CoinerConn) Send(msg []byte) {
	coinerConn.send(methodSendToCoinConnector, 0, msg, 0)
}

func (coinerConn *CoinerConn) Init(ipPort string, logger log.Logger) {
	coinerConn.doRcv = func(wsHeader wsHeader, body []byte) {
		if !bytes.Equal(wsHeader.method, methodRcvFromCoinConnector) {
			coinerConn.logger.Error()
			return
		}
		coinerConn.handleCoinConnectorMessage(body, wsHeader.nonce)
	}

	coinerConn.init(ipPort, "/srv/worker_coiner", logger)
}

func (coinerConn *CoinerConn) handleCoinConnectorMessage(data []byte, nonce uint64) {
	var txJson types.TxJson
	err := json.Unmarshal(data, &txJson)
	if err != nil {
		Logger.Errorf("Json unmarshal coin connector msg err:", err.Error())
		return
	}
	Logger.Debugf("Rcv message from coin connector.Tx info:%s", txJson.ToString())
	tx := txJson.ToTransaction()
	tx.RequestId = nonce

	if tx.Type == types.TransactionTypeCoinDepositAck || tx.Type == types.TransactionTypeFTDepositAck || tx.Type == types.TransactionTypeNFTDepositAck {
		msg := notify.CoinProxyNotifyMessage{Tx: tx}
		notify.BUS.Publish(notify.CoinProxyNotify, &msg)
	}

}
