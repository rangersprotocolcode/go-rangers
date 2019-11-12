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
	"time"
)

const (
	// ws协议头大小
	protocolHeaderSize = 28

	// 默认等待队列大小
	defaultRcvSize  = 1000
	defaultSendSize = 100

	// ws读写缓存
	defaultBufferSize = 1024 * 1024 * 16
)

type wsHeader struct {
	method   []byte
	sourceId uint64
	targetId uint64
	nonce    uint64
}

type baseConn struct {
	// 链接
	url  string
	path string
	conn *websocket.Conn

	// 读写缓冲区
	rcvChan  chan []byte
	sendChan chan []byte

	// 读写缓冲区大小
	rcvSize  int
	sendSize int

	// 处理消息的业务逻辑
	doRcv func(wsHeader wsHeader, msg []byte)

	// 处理[]byte原始消息，用于流控
	rcv func(msg []byte)

	// 判断是否要发送，用于流控
	isSend func(method []byte, target uint64, msg []byte, nonce uint64) bool

	logger log.Logger
}

// 根据url初始化
func (base *baseConn) init(ipPort, path string, logger log.Logger) {
	base.logger = logger
	base.path = path

	url := url.URL{Scheme: "ws", Host: ipPort, Path: path}
	base.url = url.String()

	// 获取链接
	base.conn = base.getWSConn()

	// 初始化读写缓存
	if 0 == base.rcvSize {
		base.rcvSize = defaultRcvSize
	}
	if 0 == base.sendSize {
		base.sendSize = defaultSendSize
	}
	base.rcvChan = make(chan []byte, base.rcvSize)
	base.sendChan = make(chan []byte, base.sendSize)

	// 开启goroutine
	base.start()
}

// 建立ws连接
func (base *baseConn) getWSConn() *websocket.Conn {
	base.logger.Debugf("connecting to %s", base.url)
	d := websocket.Dialer{ReadBufferSize: defaultBufferSize, WriteBufferSize: defaultBufferSize,}
	conn, _, err := d.Dial(base.url, nil)
	if err != nil {
		panic("Dial to" + base.url + " err:" + err.Error())
	}

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
	for range time.Tick(time.Millisecond * 300) {
		rcv, send := len(base.rcvChan), len(base.sendChan)
		if rcv > 0 || send > 0 {
			base.logger.Errorf("%s channel size. receive: %d, send: %d", base.path, rcv, send)
		}
	}
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

		if base.rcv == nil {
			base.rcvChan <- message
		} else {
			base.rcv(message)
		}

	}
}

// 发送消息
func (base *baseConn) send(method []byte, target uint64, msg []byte, nonce uint64) {
	header := wsHeader{method: method, nonce: nonce, targetId: target}

	if base.isSend != nil && !base.isSend(method, target, msg, nonce) {
		base.logger.Errorf("%s did not accept send. wsHeader: %v, length: %d", base.path, header, len(msg))
	}

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
	byteArray := make([]byte, protocolHeaderSize)
	copy(byteArray[0:4], h.method)
	copy(byteArray[12:20], utility.UInt64ToByte(h.targetId))
	copy(byteArray[20:28], utility.UInt64ToByte(h.nonce))
	return byteArray
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
var (
	methodCodeClientReader, _ = hex.DecodeString("60000000")
	methodCodeClientWriter, _ = hex.DecodeString("60000001")
	methodNotify, _           = hex.DecodeString("20000000")
	methodNotifyBroadcast, _  = hex.DecodeString("20000001")
	methodNotifyGroup, _      = hex.DecodeString("20000002")
)

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
			clientConn.logger.Error("%s received wrong method. wsHeader: %v", clientConn.path, wsHeader)
			return
		}

		clientConn.handleClientMessage(body, strconv.FormatUint(wsHeader.sourceId, 10), wsHeader.nonce, event)
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
	return uint64(binary.BigEndian.Uint64(idBytes))
}

// 处理coiner的请求
var (
	methodSendToCoinConnector, _  = hex.DecodeString("30000003")
	methodRcvFromCoinConnector, _ = hex.DecodeString("30000002")
)

type CoinerConn struct {
	baseConn
}

func (coinerConn *CoinerConn) Send(msg []byte) {
	coinerConn.send(methodSendToCoinConnector, 0, msg, 0)
}

func (coinerConn *CoinerConn) Init(ipPort string, logger log.Logger) {
	coinerConn.doRcv = func(wsHeader wsHeader, body []byte) {
		if !bytes.Equal(wsHeader.method, methodRcvFromCoinConnector) {
			coinerConn.logger.Error("%s received wrong method. wsHeader: %v", coinerConn.path, wsHeader)
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
		Logger.Errorf("Json unmarshal coiner msg err:", err.Error())
		return
	}
	Logger.Debugf("Rcv message from coiner.Tx info:%s", txJson.ToString())
	if !types.CoinerSignVerifier.VerifyDeposit(txJson) {
		Logger.Infof("Tx from coiner verify sign error!Tx Info:%s", txJson.ToString())
		return
	}
	tx := txJson.ToTransaction()
	tx.RequestId = nonce

	if tx.Type == types.TransactionTypeCoinDepositAck || tx.Type == types.TransactionTypeFTDepositAck || tx.Type == types.TransactionTypeNFTDepositAck {
		msg := notify.CoinProxyNotifyMessage{Tx: tx}
		notify.BUS.Publish(notify.CoinProxyNotify, &msg)
		return
	}
	Logger.Infof("Unknown type from coiner:%d", txJson.Type)
}
