package network

import (
	"x/src/middleware/notify"
	"github.com/gorilla/websocket"
	"hash/fnv"
	"encoding/json"
	"x/src/middleware/types"
	"crypto/md5"
	"encoding/binary"
	"strconv"
	"sync"
)

var Server server

type server struct {
	conn             *websocket.Conn
	consensusHandler MsgHandler

	sendChan chan []byte
	rcvChan  chan []byte

	notifyNonce uint64
	nonceLock   *sync.Mutex
}

func (s *server) Send(id string, msg Message) {
	s.sendMessage(methodCodeSend, id, msg, 0)
}

func (s *server) SpreadToGroup(groupId string, msg Message) {
	s.sendMessage(methodCodeSendToGroup, groupId, msg, 0)
}

func (s *server) Broadcast(msg Message) {
	s.sendMessage(methodCodeBroadcast, "0", msg, 0)
}

func (s *server) SendToClientReader(id string, msg []byte, nonce uint64) {
	s.send(methodCodeClientReader, id, msg, nonce)
}

func (s *server) SendToClientWriter(id string, msg []byte, nonce uint64) {
	s.send(methodCodeClientWriter, id, msg, nonce)
}

func (s *server) SendToCoinConnector(msg []byte) {
	s.send(methodSendToCoinConnector, "0", msg, 0)
}

func (s *server) Notify(isunicast bool, gameId string, userid string, msg string) {
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

	s.nonceLock.Lock()
	defer s.nonceLock.Unlock()

	s.notifyNonce = s.notifyNonce + 1
	notifyId := s.generateNotifyId(gameId, userid)

	s.send(method, notifyId, []byte(msg), s.notifyNonce)

}

func (s *server) handleMinerMessage(data []byte, from string) {
	message, error := unMarshalMessage(data)
	if error != nil {
		Logger.Errorf("Proto unmarshal node message error:%s", error.Error())
		return
	}
	Logger.Debugf("Rcv from node: %s,code:%d,msg size:%d,hash:%s", from, message.Code, len(data), message.Hash())

	code := message.Code
	switch code {
	case CurrentGroupCastMsg, CastVerifyMsg, VerifiedCastMsg2, AskSignPkMsg, AnswerSignPkMsg, ReqSharePiece, ResponseSharePiece:
		s.consensusHandler.Handle(from, *message)
	case ReqTransactionMsg:
		msg := notify.TransactionReqMessage{TransactionReqByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.TransactionReq, &msg)
	case GroupChainCountMsg:
		msg := notify.GroupHeightMessage{HeightByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.GroupHeight, &msg)
	case ReqGroupMsg:
		msg := notify.GroupReqMessage{GroupIdByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.GroupReq, &msg)
	case GroupMsg:
		msg := notify.GroupInfoMessage{GroupInfoByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.Group, &msg)
	case TransactionGotMsg:
		msg := notify.TransactionGotMessage{TransactionGotByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.TransactionGot, &msg)
	case TransactionBroadcastMsg:
		msg := notify.TransactionBroadcastMessage{TransactionsByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.TransactionBroadcast, &msg)
	case BlockInfoNotifyMsg:
		msg := notify.BlockInfoNotifyMessage{BlockInfo: message.Body, Peer: from}
		notify.BUS.Publish(notify.BlockInfoNotify, &msg)
	case ReqBlock:
		msg := notify.BlockReqMessage{HeightByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.BlockReq, &msg)
	case BlockResponseMsg:
		msg := notify.BlockResponseMessage{BlockResponseByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.BlockResponse, &msg)
	case NewBlockMsg:
		msg := notify.NewBlockMessage{BlockByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.NewBlock, &msg)
	case ChainPieceInfoReq:
		msg := notify.ChainPieceInfoReqMessage{HeightByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.ChainPieceInfoReq, &msg)
	case ChainPieceInfo:
		msg := notify.ChainPieceInfoMessage{ChainPieceInfoByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.ChainPieceInfo, &msg)
	case ReqChainPieceBlock:
		msg := notify.ChainPieceBlockReqMessage{ReqHeightByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.ChainPieceBlockReq, &msg)
	case ChainPieceBlock:
		msg := notify.ChainPieceBlockMessage{ChainPieceBlockMsgByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.ChainPieceBlock, &msg)
	}
}

func (s *server) joinGroup(groupID string) {
	header := header{}
	header.method = methodCodeJoinGroup

	hash64 := fnv.New64()
	hash64.Write([]byte(groupID))
	target := hash64.Sum64()
	Logger.Debugf("Join group:%d", target)
	header.targetId = target

	s.sendChan <- header.toBytes()
}

func (s *server) handleClientMessage(data []byte, userId string, nonce uint64, event string) {
	var txJson types.TxJson
	err := json.Unmarshal(data, &txJson)
	if nil != err {
		Logger.Errorf("Json unmarshal client message error:%s", err.Error())
		return
	}
	tx := txJson.ToTransaction()
	Logger.Debugf("Rcv from client.Tx info:%s", txJson.ToString())

	msg := notify.ClientTransactionMessage{Tx: tx, UserId: userId, Nonce: nonce}
	notify.BUS.Publish(event, &msg)
}

func (s *server) handleCoinConnectorMessage(data []byte, nonce uint64) {
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
func (s *server) generateNotifyId(gameId string, userId string) string {
	data := []byte(gameId)
	if 0 != len(userId) {
		data = append(data, []byte(userId)...)
	}

	md5Result := md5.Sum(data)
	idBytes := md5Result[4:12]
	id := uint64(binary.BigEndian.Uint64(idBytes))

	return strconv.FormatUint(id, 10)
}
