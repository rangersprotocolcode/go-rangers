package network

import (
	"bytes"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/notify"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/utility"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"github.com/gorilla/websocket"
	"net/url"
	"strconv"
	"sync"
	"time"
)

const (
	// ws协议头大小
	protocolHeaderSize = 28

	// 默认等待队列大小
	defaultRcvSize  = 10000
	defaultSendSize = 100

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
}

// 根据url初始化
func (base *baseConn) init(ipPort, path string, logger log.Logger) {
	base.logger = logger
	base.path = path

	url := url.URL{Scheme: "ws", Host: ipPort, Path: path}
	base.url = url.String()
	base.connLock = sync.Mutex{}
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
	d := websocket.Dialer{ReadBufferSize: defaultBufferSize, WriteBufferSize: defaultBufferSize}
	conn, _, err := d.Dial(base.url, nil)
	if err != nil {
		base.logger.Errorf("Dial to" + base.url + " err:" + err.Error())
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
	base.conn.Close()
	base.conn = nil
}

// 发送消息
func (base *baseConn) send(method []byte, target uint64, msg []byte, nonce uint64) {
	header := wsHeader{method: method, nonce: nonce, targetId: target}

	if base.isSend != nil && !base.isSend(method, target, msg, nonce) {
		base.logger.Errorf("%s did not accept send. wsHeader: %v, length: %d", base.path, header, len(msg))
	}

	base.sendChan <- base.loadMsg(header, msg)
	base.logger.Debugf("send message. wsHeader: %v, length: %d,body:%s", header, len(msg), string(msg))
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
	methodNotifyInit, _       = hex.DecodeString("20000003")
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

func (clientConn *ClientConn) Init(ipPort, path, event string, method []byte, logger log.Logger, isNotify bool) {
	clientConn.method = method
	clientConn.nonceLock = sync.Mutex{}
	clientConn.notifyNonce = 0

	clientConn.doRcv = func(wsHeader wsHeader, body []byte) {
		if bytes.Equal(wsHeader.method, methodNotifyInit) {
			clientConn.logger.Errorf("refresh notify nonce: %d", wsHeader.nonce)
			clientConn.notifyNonce = wsHeader.nonce
			return
		}

		if !bytes.Equal(wsHeader.method, clientConn.method) {
			clientConn.logger.Error("%s received wrong method. wsHeader: %v", clientConn.path, wsHeader)
			return
		}

		clientConn.handleClientMessage(body, strconv.FormatUint(wsHeader.sourceId, 10), wsHeader.nonce, event)
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

	if isNotify {
		clientConn.initNotify()
	}
}

func (clientConn *ClientConn) initNotify() {
	clientConn.logger.Errorf("initNotify")
	clientConn.send(methodNotifyInit, 0, []byte{}, 0)
}

func (clientConn *ClientConn) handleClientMessage(body []byte, userId string, nonce uint64, event string) {
	var txJson types.TxJson
	err := json.Unmarshal(body, &txJson)
	if nil != err {
		clientConn.logger.Errorf("Json unmarshal client message error:%s", err.Error())
		return
	}
	clientConn.logger.Debugf("Rcv from client.Tx json:%s", txJson.ToString())

	tx := txJson.ToTransaction()
	tx.RequestId = nonce
	clientConn.logger.Debugf("Rcv from client.Tx info:%s", tx.ToTxJson().ToString())

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

type ConnectorConn struct {
	baseConn
}

func (connectorConn *ConnectorConn) Send(msg []byte) {
	connectorConn.send(methodSendToCoinConnector, 0, msg, 0)
}

func (connectorConn *ConnectorConn) Init(ipPort string, logger log.Logger) {
	connectorConn.doRcv = func(wsHeader wsHeader, body []byte) {
		if !bytes.Equal(wsHeader.method, methodRcvFromCoinConnector) {
			connectorConn.logger.Error("%s received wrong method. wsHeader: %v", connectorConn.path, wsHeader)
			return
		}
		connectorConn.handleConnectorMessage(body, wsHeader.nonce)
	}

	connectorConn.init(ipPort, "/srv/worker_coiner", logger)
}

func (connectorConn *ConnectorConn) handleConnectorMessage(data []byte, nonce uint64) {
	var txJson types.TxJson
	err := json.Unmarshal(data, &txJson)
	if err != nil {
		Logger.Errorf("Json unmarshal coiner msg err:", err.Error())
		return
	}
	tx := txJson.ToTransaction()
	tx.RequestId = nonce
	Logger.Debugf("Rcv message from coiner.Tx info:%s", tx.ToTxJson().ToString())
	if !types.CoinerSignVerifier.VerifyDeposit(txJson) {
		Logger.Infof("Tx from coiner verify sign error!Tx Info:%s", txJson.ToString())
		return
	}

	if tx.Type == types.TransactionTypeCoinDepositAck || tx.Type == types.TransactionTypeFTDepositAck || tx.Type == types.TransactionTypeNFTDepositAck {
		msg := notify.CoinProxyNotifyMessage{Tx: tx}
		notify.BUS.Publish(notify.CoinProxyNotify, &msg)
		return
	}
	Logger.Infof("Unknown type from coiner:%d", txJson.Type)
}
