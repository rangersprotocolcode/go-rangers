// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package network

var instance server

type server struct {
	// 节点间消息传递
	worker WorkerConn

	// 客户端消息
	reader ClientConn

	// tx
	tx TxConn

	isSending bool
}

func (s *server) Init(gateAddr, outerGateAddr string, selfMinerId []byte, consensusHandler MsgHandler, isSending bool) {
	s.worker.Init(gateAddr, selfMinerId, consensusHandler, bizLogger)
	s.isSending = isSending
	if s.isSending {
		s.reader.Init(outerGateAddr, "/srv/node", bizLogger)
	}
}

func (s *server) InitTx(tx string) {
	if 0 == len(tx) {
		return
	}

	s.tx.Init(tx)
}

func (s *server) SendToJSONRPC(msg []byte, sessionId string, requestId uint64) {
	if s.isSending {
		s.reader.Send(sessionId, methodClientJSONRpc, msg, requestId)
	}
}

func (s *server) SendToClientReader(id string, msg []byte, nonce uint64) {
	if s.isSending {
		s.reader.Send(id, methodClientReader, msg, nonce)
	}
}

func (s *server) SendToClientWriter(id string, msg []byte, nonce uint64) {
	s.SendToClientReader(id, msg, nonce)
}

func (s *server) Send(id string, msg Message) {
	s.worker.SendToOne(id, msg)
}

func (s *server) SpreadToGroup(groupId string, msg Message) {
	s.worker.SendToGroup(groupId, msg)
}

func (s *server) Broadcast(msg Message) {
	s.worker.SendToEveryone(msg)
}

func (s *server) JoinGroupNet(groupId string) {
	s.worker.JoinGroupNet(groupId)
}

func (s *server) QuitGroupNet(groupId string) {
	s.worker.QuitGroupNet(groupId)
}

func (s *server) SendToStranger(strangerId []byte, msg Message) {
	s.worker.SendToStranger(strangerId, msg)
}
