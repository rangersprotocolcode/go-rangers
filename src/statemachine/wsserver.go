package statemachine

import (
	"fmt"
	"net"
	"net/http"
	"github.com/gorilla/websocket"
	"x/src/middleware/log"
	"x/src/common"
	"strings"
	"math/rand"
	"time"
	"encoding/json"
	"x/src/middleware/types"
)

type wsServer struct {
	listener net.Listener
	uri      string
	addr     string
	port     int
	upgrade  *websocket.Upgrader

	conn *websocket.Conn

	// 读写缓冲区
	rcvChan  chan []byte
	sendChan chan []byte

	logger log.Logger
}

func newWSServer(uri string) *wsServer {
	ws := new(wsServer)
	ws.uri = fmt.Sprintf("/%s", uri)
	ws.rcvChan = make(chan []byte, 1)
	ws.sendChan = make(chan []byte, 1)
	ws.logger = log.GetLoggerByIndex(log.WSLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	ws.upgrade = &websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			if r.Method != "GET" {
				ws.logger.Errorf("method is not GET")
				return false
			}
			if r.URL.Path != ws.uri {
				ws.logger.Errorf("path error")
				return false
			}
			// 只接受本地
			if !strings.HasPrefix(r.RemoteAddr, "127.0.0.1") && !strings.HasPrefix(r.RemoteAddr, "0.0.0.0") && !strings.HasPrefix(r.RemoteAddr, "172.17.0.") {
				ws.logger.Errorf("not local call error. %s", r.RemoteAddr)
				return false
			}

			return true
		},
	}

	return ws
}

func (self *wsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != self.uri {
		httpCode := http.StatusInternalServerError
		reasePhrase := http.StatusText(httpCode)
		self.logger.Errorf("path error, %s", reasePhrase)
		http.Error(w, reasePhrase, httpCode)
		return
	}

	conn, err := self.upgrade.Upgrade(w, r, nil)
	if err != nil {
		self.logger.Errorf("websocket error: %s", err.Error())
		return
	}

	self.logger.Debugf("client connect: %v", conn.RemoteAddr())

	go self.connHandle(conn)
}

func (self *wsServer) connHandle(conn *websocket.Conn) {
	defer func() {
		conn.Close()
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			// 判断是不是超时
			if netErr, ok := err.(net.Error); ok {
				if netErr.Timeout() {
					self.logger.Debugf("ReadMessage timeout remote: %v\n", conn.RemoteAddr())
					return
				}
			}
			// 其他错误，如果是 1001 和 1000 就不打印日志
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				self.logger.Debugf("ReadMessage other remote:%v error: %v \n", conn.RemoteAddr(), err)
			}
			return
		}

		self.logger.Debugf("receive msg: %d, string: %s", data, string(data))

		// 处理msg
		var (
			msg    wsMessage
			flag   bool
			answer string
			reply  []byte
		)

		err = json.Unmarshal(data, &msg)
		if err != nil {
			// 错误回复
			reply = self.buildResponse(false, err.Error())
		} else {
			switch msg.Type {

			// FT 部分
			case types.TransactionTypeMintFT:
				answer, flag = self.mintFT(msg.Data)
				break
			case types.TransactionTypeTransferBNT:
				answer, flag = self.transferBNT(msg.Data)
				break
			case types.TransactionTypeTransferFT:
				answer, flag = self.transferFT(msg.Data)
				break
			case types.TransactionTypePublishFT:
				answer, flag = self.publishFTSet(msg.Data)
				break

				// NFT部分
			case types.TransactionTypeMintNFT:
				answer, flag = self.mintNFT(msg.Data)
				break
			case types.TransactionTypePublishNFTSet:
				answer, flag = self.publishNFTSet(msg.Data)
				break
			case types.TransactionTypeLockNFT:
				answer, flag = self.lockNFT(msg.Data)
				break
			case types.TransactionTypeUnLockNFT:
				answer, flag = self.unLockNFT(msg.Data)
				break
			case types.TransactionTypeApproveNFT:
				answer, flag = self.approveNFT(msg.Data)
				break
			case types.TransactionTypeRevokeNFT:
				answer, flag = self.revokeNFT(msg.Data)
				break
			case types.TransactionTypeTransferNFT:
				answer, flag = self.transferNFT(msg.Data)
				break
			case types.TransactionTypeUpdateNFT:
				answer, flag = self.updateNFT(msg.Data)
				break
			case types.TransactionTypeBatchUpdateNFT:
				answer, flag = self.batchUpdateNFT(msg.Data)
				break

				// 查询部分
			case types.TransactionTypeGetCoin:
				answer, flag = self.getBNTBalance(msg.Data)
				break
			case types.TransactionTypeGetAllCoin:
				answer, flag = self.getAllCoinInfo(msg.Data)
				break
			case types.TransactionTypeFT:
				answer, flag = self.getFTBalance(msg.Data)
				break
			case types.TransactionTypeFTSet:
				answer, flag = self.getFTSet(msg.Data)
				break
			case types.TransactionTypeAllFT:
				answer, flag = self.getAllFT(msg.Data)
				break
			case types.TransactionTypeNFTCount:
				answer, flag = self.getNFTCount(msg.Data)
				break
			case types.TransactionTypeNFT:
				answer, flag = self.getNFT(msg.Data)
				break
			case types.TransactionTypeNFTListByAddress:
				answer, flag = self.getAllNFT(msg.Data)
				break
			case types.TransactionTypeNFTSet:
				answer, flag = self.getNFTSet(msg.Data)
				break
			case types.TransactionTypeNotify:
				answer, flag = self.notify(msg.Data)
				break
			case types.TransactionTypeNotifyGroup:
				answer, flag = self.notifyGroup(msg.Data)
				break
			case types.TransactionTypeNotifyBroadcast:
				answer, flag = self.notifyBroadcast(msg.Data)
				break
			default:
				answer = "wrong msg type"
				flag = false
			}

		}

		reply = self.buildResponse(flag, answer)

		// 回复client
		conn.WriteMessage(websocket.TextMessage, reply)
	}
}

func (self *wsServer) Start() (err error) {
	for {
		self.port = self.randomPort()
		self.addr = fmt.Sprintf("0.0.0.0:%d", self.port)
		self.listener, err = net.Listen("tcp", self.addr)
		if err == nil {
			self.logger.Infof("net listen success, %s", self.getURL())
			break
		}
		self.logger.Infof("net listen error:", err)
	}

	go http.Serve(self.listener, self)

	self.logger.Errorf("wsserver started, url: %s%s", self.addr, self.uri)
	return nil
}

func (self *wsServer) randomPort() int {
	rand.Seed(int64(time.Now().UnixNano()))
	port := 9000 + int(rand.Float32()*1000)
	return port
}

func (self *wsServer) getURL() string {
	return fmt.Sprintf("%s%s", self.addr, self.uri)
}

func (self *wsServer) GetURL() string {
	return fmt.Sprintf("ws://%s:%d%s", "172.17.0.1", self.port, self.uri)
}

// 暂时不实现layer2主动调用STM
func (self *wsServer) Send(msg []byte) {

}

func (self *wsServer) buildResponse(status bool, data string) []byte {
	resp := response{Data: data}
	if status {
		resp.Status = 0
	} else {
		resp.Status = 1
	}

	result, _ := json.Marshal(resp)
	return result
}

type wsMessage struct {
	Type int               `json:"type"`
	Data map[string]string `json:"data"`
}

type response struct {
	Status int    `json:"status"` // 0 成功 1失败
	Data   string `json:"data"`
}
