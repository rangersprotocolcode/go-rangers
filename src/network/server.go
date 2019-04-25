package network

import (
	"x/src/middleware/notify"
	"github.com/gorilla/websocket"
	"hash/fnv"
	"encoding/json"
	"x/src/middleware/types"
)

var Server server

type server struct {
	conn             *websocket.Conn
	consensusHandler MsgHandler

	sendChan chan []byte
	rcvChan  chan []byte
}

func (s *server) Send(id string, msg Message) {
	s.send(methodCodeSend, id, msg, 0)
}

func (s *server) SpreadToGroup(groupId string, msg Message) {
	s.send(methodCodeSendToGroup, groupId, msg, 0)
}

func (s *server) Broadcast(msg Message) {
	s.send(methodCodeBroadcast, "0", msg, 0)
}

func (s *server) SendToClient(id string, msg Message, nonce uint64) {
	s.send(methodCodeClientSend, id, msg, nonce)
}

func (s *server) SendToCoinProxy(msg Message) {
	s.send(methodCodeCoinProxySend, "0", msg, 0)
}

func (s *server) handleMinerMessage(data []byte, from string) {
	message, error := unMarshalMessage(data)
	if error != nil {
		Logger.Errorf("Proto unmarshal error:%s", error.Error())
		return
	}
	Logger.Debugf("Receive message from %s,code:%d,msg size:%d,hash:%s", from, message.Code, len(data), message.Hash())

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
		Logger.Debugf("Rcv ChainPieceInfoReq from %s", from)
		msg := notify.ChainPieceInfoReqMessage{HeightByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.ChainPieceInfoReq, &msg)
	case ChainPieceInfo:
		Logger.Debugf("Rcv ChainPieceInfo from %s", from)
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

func (s *server) handleClientMessage(data []byte, userId string, nonce uint64) {
	var txJson types.TxJson
	err := json.Unmarshal(data, &txJson)
	if nil != err {
		Logger.Errorf("handleClientMessage json error:%s", err.Error())
		return
	}

	tx := txJson.ToTransaction()
	Logger.Debugf("Receive message from client.Tx:%v", txJson)
	// 记录返回地址
	msg := notify.ClientTransactionMessage{Tx: tx, UserId: userId, Nonce: nonce}
	notify.BUS.Publish(notify.ClientTransaction, &msg)

}

func (s *server) handleCoinProxyMessage(data []byte) {
	message, error := unMarshalMessage(data)
	if error != nil {
		Logger.Errorf("Proto unmarshal error:%s", error.Error())
		return
	}

	code := message.Code
	switch code {
	case CoinProxyNotify:
		var txJson types.TxJson
		err := json.Unmarshal(message.Body, &txJson)
		if err != nil {
			Logger.Errorf("Coin proxy msg unmarshal err:", err.Error())
			return
		}
		Logger.Debugf("Receive message from coin proxy.Tx:%v", txJson)
		tx := txJson.ToTransaction()
		Logger.Debugf(".Tx:%v", tx)
		if tx.Type == types.TransactionTypeDepositAck {
			msg := notify.CoinProxyNotifyMessage{Tx: tx}
			notify.BUS.Publish(notify.CoinProxyNotify, &msg)
		}
	}
}
