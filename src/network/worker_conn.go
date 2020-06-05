package network

import (
	"bytes"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/notify"
	"encoding/hex"
	"hash/fnv"
	"strconv"
	"sync"
)

var (
	methodCodeSend, _        = hex.DecodeString("80000001")
	methodCodeBroadcast, _   = hex.DecodeString("80000002")
	methodCodeSendToGroup, _ = hex.DecodeString("80000003")
	methodCodeJoinGroup, _   = hex.DecodeString("80000004")
	methodCodeQuitGroup, _   = hex.DecodeString("80000005")
	methodSetNetId, _        = hex.DecodeString("10000000")
	methodSendToManager, _   = hex.DecodeString("80000006")
)

type WorkerConn struct {
	baseConn
	consensusHandler MsgHandler

	joinedGroup     map[string]byte
	joinedGroupLock sync.Mutex
}

func (workerConn *WorkerConn) Init(ipPort string, selfId []byte, consensusHandler MsgHandler, logger log.Logger) {
	// worker 链接加大发送队列长度
	workerConn.sendSize = 1000
	workerConn.consensusHandler = consensusHandler
	workerConn.joinedGroup = make(map[string]byte)
	workerConn.joinedGroupLock = sync.Mutex{}

	workerConn.doRcv = func(wsHeader wsHeader, body []byte) {
		method := wsHeader.method
		if !bytes.Equal(method, methodCodeSend) && !bytes.Equal(method, methodCodeBroadcast) && !bytes.Equal(method, methodCodeSendToGroup) && !bytes.Equal(method, methodSendToManager) {
			workerConn.logger.Error("received wrong method, wsHeader: %v,body:%v", wsHeader, body)
			return
		}

		if bytes.Equal(method, methodSendToManager) {
			body = body[netIdSize:]
		}
		workerConn.handleMessage(body, strconv.FormatUint(wsHeader.sourceId, 10))
	}

	workerConn.afterReconnected = func() {
		workerConn.setNetId(selfId)

		workerConn.joinedGroupLock.Lock()
		defer workerConn.joinedGroupLock.Unlock()

		for key := range workerConn.joinedGroup {
			workerConn.logger.Warnf("rejoin group: %s", key)
			workerConn.joinGroupNet(key)
		}
	}

	workerConn.init(ipPort, "/srv/worker_worker", logger)
	workerConn.setNetId(selfId)
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
	case CurrentGroupCastMsg, CastVerifyMsg, VerifiedCastMsg, AskSignPkMsg, AnswerSignPkMsg, ReqSharePiece, ResponseSharePiece,
		GroupInitMsg, KeyPieceMsg, SignPubkeyMsg, GroupInitDoneMsg, CreateGroupaRaw, CreateGroupSign, GroupPing, GroupPong:
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
	case STMStorageReady:
		msg := notify.STMStorageReadyMessage{FileName: message.Body}
		notify.BUS.Publish(notify.STMStorageReady, &msg)
		break
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

//加入组网络
func (workerConn *WorkerConn) JoinGroupNet(groupId string) {
	workerConn.joinedGroupLock.Lock()
	defer workerConn.joinedGroupLock.Unlock()
	workerConn.joinedGroup[groupId] = 0

	workerConn.joinGroupNet(groupId)
}

func (workerConn *WorkerConn) joinGroupNet(groupId string) {
	header := wsHeader{method: methodCodeJoinGroup, targetId: workerConn.generateTargetForGroup(groupId)}
	workerConn.sendChan <- workerConn.headerToBytes(header)
	workerConn.logger.Debugf("Join group: %v,targetId:%v,hex:%v", groupId, header.targetId, strconv.FormatUint(header.targetId, 16))
}

//退出组网络
func (workerConn *WorkerConn) QuitGroupNet(groupId string) {
	workerConn.joinedGroupLock.Lock()
	defer workerConn.joinedGroupLock.Unlock()
	delete(workerConn.joinedGroup, groupId)

	header := wsHeader{method: methodCodeQuitGroup, targetId: workerConn.generateTargetForGroup(groupId)}
	workerConn.sendChan <- workerConn.headerToBytes(header)
	workerConn.logger.Debugf("Quit group: %v,targetId:%v,hex:%v", groupId, header.targetId, strconv.FormatUint(header.targetId, 16))
}

func (workerConn *WorkerConn) setNetId(netId []byte) {
	header := wsHeader{method: methodSetNetId}
	bytes := workerConn.headerToBytes(header)

	headerBytes := make([]byte, len(bytes)+netIdSize)
	copy(headerBytes[:len(bytes)], bytes[:])
	copy(headerBytes[len(bytes):], netId[:])

	workerConn.sendChan <- headerBytes
	workerConn.logger.Debugf("Set net id: %v,header:%v", netId, headerBytes)
}

func (workerConn *WorkerConn) SendToStranger(strangerId []byte, msg Message) {
	msgByte, err := marshalMessage(msg)
	if err != nil {
		workerConn.logger.Errorf("worker sendMessage error. invalid message: %v", msg)
		return
	}
	workerConn.unicast(methodSendToManager, strangerId, msgByte, 0)
}
