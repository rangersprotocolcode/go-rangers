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

import (
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/notify"
)

var instance server

type server struct {
	// 节点间消息传递
	worker WorkerConn

	// 客户端消息
	reader ClientConn
	writer ClientConn

	jsonrpc JSONRPCConn
}

func (s *server) Init(logger log.Logger, gateAddr, outerGateAddr string, selfMinerId []byte, consensusHandler MsgHandler) {
	s.worker.Init(gateAddr, selfMinerId, consensusHandler, logger)

	s.reader.Init(outerGateAddr, "/srv/worker_reader", notify.ClientTransactionRead, methodCodeClientReader, logger, true, true)
	s.writer.Init(outerGateAddr, "/srv/worker_writer", notify.ClientTransaction, methodCodeClientWriter, logger, false, false)
	s.jsonrpc.Init(outerGateAddr, logger)
}

func (s *server) SendToJSONRPC(msg string, sessionId, requestId uint64) {
	s.jsonrpc.Send(msg, sessionId, requestId)
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

func (s *server) SendToClientReader(id string, msg []byte, nonce uint64) {
	s.reader.Send(id, msg, nonce)
}

func (s *server) SendToClientWriter(id string, msg []byte, nonce uint64) {
	s.writer.Send(id, msg, nonce)
}

func (s *server) Notify(isUniCast bool, gameId string, userid string, msg string) {
	s.reader.Notify(isUniCast, gameId, userid, msg)
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
