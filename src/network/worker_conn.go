package network

import (
	"bytes"
	"strconv"
	"x/src/middleware/notify"
	"encoding/hex"
	"hash/fnv"
	"x/src/middleware/log"
)

var (
	methodCodeSend, _        = hex.DecodeString("80000001")
	methodCodeBroadcast, _   = hex.DecodeString("80000002")
	methodCodeSendToGroup, _ = hex.DecodeString("80000003")
	methodCodeJoinGroup, _   = hex.DecodeString("80000004")
	methodCodeQuitGroup, _   = hex.DecodeString("80000005")
)

type WorkerConn struct {
	baseConn
	consensusHandler MsgHandler
	selfId           string
}

func (workerConn *WorkerConn) Init(ipPort, selfId string, consensusHandler MsgHandler, logger log.Logger) {
	// worker 链接加大发送队列长度
	workerConn.sendSize = 1000
	workerConn.consensusHandler = consensusHandler
	workerConn.selfId = selfId
	workerConn.doRcv = func(wsHeader wsHeader, body []byte) {
		method := wsHeader.method
		if !bytes.Equal(method, methodCodeSend) && !bytes.Equal(method, methodCodeBroadcast) && !bytes.Equal(method, methodCodeSendToGroup) {
			workerConn.logger.Error("received wrong method, wsHeader: %v", wsHeader)
			return
		}

		workerConn.handleMessage(body, strconv.FormatUint(wsHeader.sourceId, 10))
	}

	workerConn.init(ipPort, "/srv/worker_worker", logger)
	workerConn.joinGroup()
}

func (workerConn *WorkerConn) handleMessage(data []byte, from string) {
	message, error := unMarshalMessage(data)
	if error != nil {
		workerConn.logger.Errorf("Proto unmarshal node message error: %s", error.Error())
		return
	}

	workerConn.logger.Debugf("Rcv from node: %s,code: %d,msg size: %d,hash: %s", from, message.Code, len(data), message.Hash())

	code := message.Code
	switch code {
	case CurrentGroupCastMsg, CastVerifyMsg, VerifiedCastMsg2, AskSignPkMsg, AnswerSignPkMsg, ReqSharePiece, ResponseSharePiece:
		if nil != workerConn.consensusHandler {
			workerConn.consensusHandler.Handle(from, *message)
		}
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

// 节点加入组，订阅组消息
func (workerConn *WorkerConn) joinGroup() {
	for _, group := range netMemberInfo.VerifyGroupList {
		for _, member := range group.Members {
			if workerConn.selfId != member {
				continue
			}

			header := wsHeader{method: methodCodeJoinGroup, targetId: workerConn.generateTargetForGroup(group.GroupId)}
			workerConn.sendChan <- workerConn.headerToBytes(header)
			workerConn.logger.Warnf("Join group: %d", header.targetId)
			break
		}
	}
}

func (workerConn *WorkerConn) generateTargetForGroup(groupId string) uint64 {
	hash64 := fnv.New64()
	hash64.Write([]byte(groupId))
	return hash64.Sum64()
}

func (workerConn *WorkerConn) sendMessage(method []byte, target uint64, message Message, nonce uint64) {
	msg, err := marshalMessage(message)
	if err != nil {
		workerConn.logger.Errorf("worker sendMessage error. invalid message: %v", message)
		return
	}

	workerConn.send(method, target, msg, nonce)
}

// 单发
func (workerConn *WorkerConn) SendToOne(id string, message Message) {
	target, err := workerConn.generateTarget(id)
	if err != nil {
		return
	}

	workerConn.sendMessage(methodCodeSend, target, message, 0)
}

// 组播
func (workerConn *WorkerConn) SendToGroup(groupId string, msg Message) {
	target := workerConn.generateTargetForGroup(groupId)
	workerConn.sendMessage(methodCodeSendToGroup, target, msg, 0)
}

// 广播
func (workerConn *WorkerConn) SendToEveryone(msg Message) {
	workerConn.sendMessage(methodCodeBroadcast, 0, msg, 0)
}
