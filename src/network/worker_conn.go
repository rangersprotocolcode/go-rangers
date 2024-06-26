// Copyright 2020 The RangersProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RangersProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RangersProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package network

import (
	"bytes"
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/common/sha3"
	"com.tuntun.rangers/node/src/middleware"
	"com.tuntun.rangers/node/src/middleware/log"
	"com.tuntun.rangers/node/src/middleware/notify"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/service"
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
	methodCodeTxBroadcast, _ = hex.DecodeString("80000007")
)

type WorkerConn struct {
	baseConn
	consensusHandler MsgHandler

	joinedGroup     map[string]byte
	joinedGroupLock sync.Mutex
}

func (workerConn *WorkerConn) Init(ipPort string, selfId []byte, consensusHandler MsgHandler, logger log.Logger) {
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
	message, err := unMarshalMessage(data)
	if err != nil {
		workerConn.logger.Errorf("Proto unmarshal node message error: %s", err.Error())
		return
	}

	workerConn.logger.Debugf("Rcv from node: %s,code: %d, msg size: %d", from, message.Code, len(data))

	code := message.Code
	switch code {
	case CurrentGroupCastMsg, CastVerifyMsg, VerifiedCastMsg, AskSignPkMsg, AnswerSignPkMsg, ReqSharePiece, ResponseSharePiece,
		GroupInitMsg, KeyPieceMsg, SignPubkeyMsg, GroupInitDoneMsg, CreateGroupaRaw, CreateGroupSign, GroupPing, GroupPong:
		if nil != workerConn.consensusHandler {
			go workerConn.consensusHandler.Handle(from, *message)
		}
	case NewBlockMsg:
		msgHash := sha3.Sum256(message.Body)
		middleware.PerfLogger.Infof("new block, msghash: %s", common.ToHex(msgHash[:]))
		msg := notify.NewBlockMessage{BlockByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.NewBlock, &msg)
	case ReqTransactionMsg:
		msg := notify.TransactionReqMessage{TransactionReqByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.TransactionReq, &msg)
	case TransactionGotMsg:
		txs, e := types.UnMarshalTransactions(message.Body)
		if e != nil {
			workerConn.logger.Errorf("Unmarshal got transactions error:%s", e.Error())
			return
		}

		for _, tx := range txs {
			if nil == tx {
				continue
			}

			if err := service.GetTransactionPool().VerifyTransaction(tx, common.GetBlockHeight()); nil == err {
				service.GetTransactionPool().AddTransaction(tx)
			}

		}

		m := notify.TransactionGotAddSuccMessage{Transactions: txs, Peer: from}
		notify.BUS.Publish(notify.TransactionGotAddSucc, &m)

	case TopBlockInfoMsg:
		msg := notify.ChainInfoMessage{ChainInfo: message.Body, Peer: from}
		notify.BUS.Publish(notify.TopBlockInfo, &msg)
	case BlockChainPieceReqMsg:
		msg := notify.BlockChainPieceReqMessage{BlockChainPieceReq: message.Body, Peer: from}
		notify.BUS.Publish(notify.BlockChainPieceReq, &msg)
	case BlockChainPieceMsg:
		msg := notify.BlockChainPieceMessage{BlockChainPieceByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.BlockChainPiece, &msg)
	case ReqBlockMsg:
		msg := notify.BlockReqMessage{ReqInfoByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.BlockReq, &msg)
	case BlockResponseMsg:
		msg := notify.BlockResponseMessage{BlockResponseByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.BlockResponse, &msg)
	case ReqGroupMsg:
		msg := notify.GroupReqMessage{ReqInfoByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.GroupReq, &msg)
	case GroupResponseMsg:
		msg := notify.GroupResponseMessage{GroupResponseByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.GroupResponse, &msg)
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

func (workerConn *WorkerConn) SendToOne(id string, message Message) {
	target, err := workerConn.generateTarget(id)
	if err != nil {
		return
	}

	workerConn.sendMessage(methodCodeSend, target, message, 0)
}

func (workerConn *WorkerConn) SendToGroup(groupId string, msg Message) {
	target := workerConn.generateTargetForGroup(groupId)
	workerConn.sendMessage(methodCodeSendToGroup, target, msg, 0)
}

func (workerConn *WorkerConn) SendToEveryone(msg Message) {
	workerConn.sendMessage(methodCodeBroadcast, 0, msg, 0)
}

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
