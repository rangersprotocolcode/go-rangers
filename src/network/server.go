package network

import (
	"x/src/middleware/notify"
	"x/src/middleware/log"
)

var instance server

type server struct {
	// 节点间消息传递
	worker WorkerConn

	// 客户端消息
	reader ClientConn
	writer ClientConn

	// coiner消息
	coiner CoinerConn
}

func (s *server) Init(logger log.Logger, selfMinerId string, consensusHandler MsgHandler) {
	s.reader.Init(gateAddr, "/srv/worker_reader", notify.ClientTransactionRead, methodCodeClientReader, Logger)
	s.writer.Init(gateAddr, "/srv/worker_writer", notify.ClientTransaction, methodCodeClientWriter, Logger)
	s.worker.Init(gateAddr, selfMinerId, consensusHandler, Logger)
	s.coiner.Init(gateAddr, Logger)
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

func (s *server) SendToCoinConnector(msg []byte) {
	s.coiner.Send(msg)
}

func (s *server) Notify(isunicast bool, gameId string, userid string, msg string) {
	s.reader.Notify(isunicast, gameId, userid, msg)
}
