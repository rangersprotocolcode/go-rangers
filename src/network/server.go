package network

import (
	"x/src/middleware/notify"
	"github.com/gorilla/websocket"
	"x/src/utility"
	"strconv"
	"encoding/hex"
	"bytes"
	"hash/fnv"
	"sync"
)

var methodCodeSend, _ = hex.DecodeString("80000001")
var methodCodeBroadcast, _ = hex.DecodeString("80000002")
var methodCodeSendToGroup, _ = hex.DecodeString("80000003")
var methodCodeJoinGroup, _ = hex.DecodeString("80000004")
var methodCodeQuitGroup, _ = hex.DecodeString("80000005")

var Server server

type server struct {
	conn             *websocket.Conn
	consensusHandler MsgHandler

	lock sync.RWMutex
}

func (s *server) Send(id string, msg Message) {
	s.send(methodCodeSend, id, msg)
}

func (s *server) SpreadToGroup(groupId string, msg Message) {
	s.send(methodCodeSendToGroup, groupId, msg)
}

func (s *server) Broadcast(msg Message) {
	s.send(methodCodeBroadcast, "0", msg)
}

func (s *server) send(method []byte, targetId string, msg Message) {
	m, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("marshal message error:%s", err.Error())
		return
	}

	header := make([]byte, protocolHeaderSize)
	copy(header[0:4], method)

	var target uint64
	if bytes.Equal(method, methodCodeSendToGroup) {
		hash64 := fnv.New64()
		hash64.Write([]byte(targetId))
		target = hash64.Sum64()
		Logger.Debugf("Send group to:%d,%v", target, utility.UInt64ToByte(target))
	} else {
		target, err = strconv.ParseUint(targetId, 10, 64)
		Logger.Debugf("Send  to:%d", target)
		if err != nil {
			Logger.Errorf("Parse target id %s error:%s", targetId, err.Error())
			return
		}
	}
	copy(header[12:20], utility.UInt64ToByte(target))

	message := make([]byte, protocolHeaderSize+len(m))
	copy(message[:protocolHeaderSize-1], header[:])
	copy(message[protocolHeaderSize:], m)

	Logger.Debugf("Send msg:%v", message)
	s.lock.Lock()
	err = s.conn.WriteMessage(websocket.BinaryMessage, message)
	s.lock.Unlock()
	if err != nil {
		Logger.Errorf("Send msg error:%s", err.Error())
	}
}

func (s *server) parseWebSocketMsg(m []byte) (from string, data []byte) {
	if len(m) < protocolHeaderSize {
		return "", nil
	}

	header := m[0 : protocolHeaderSize-1]
	data = m[protocolHeaderSize:]
	srcByte := header[12:20]
	from = strconv.FormatUint(utility.ByteToUInt64(srcByte), 10)
	return
}

func (s *server) receiveMessage() {
	for {
		_, message, err := s.conn.ReadMessage()
		if err != nil {
			Logger.Errorf("Rcv msg error:%s", err.Error())
			continue
		}
		Logger.Debugf("Rcv msg:%v", message)
		from, data := s.parseWebSocketMsg(message)
		go s.handleMessage(data, from)
	}
}

func (s *server) handleMessage(data []byte, from string) {
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
	header := make([]byte, protocolHeaderSize)
	copy(header[0:4], methodCodeJoinGroup)

	hash64 := fnv.New64()
	hash64.Write([]byte(groupID))
	target := hash64.Sum64()
	Logger.Debugf("Join group:%d", target)
	copy(header[12:20], utility.UInt64ToByte(target))

	err := s.conn.WriteMessage(websocket.BinaryMessage, header)
	if err != nil {
		Logger.Errorf("Send msg error:%s", err.Error())
	}
}
