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
	coiner ConnectorConn
}

func (s *server) Init(logger log.Logger, gateAddr, selfMinerId string, consensusHandler MsgHandler) {
	s.reader.Init(gateAddr, "/srv/worker_reader", notify.ClientTransactionRead, methodCodeClientReader, logger, true)
	s.writer.Init(gateAddr, "/srv/worker_writer", notify.ClientTransaction, methodCodeClientWriter, logger, false)
	s.worker.Init(gateAddr, selfMinerId, consensusHandler, logger)
	s.coiner.Init(gateAddr, logger)
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

func (s *server) Notify(isUniCast bool, gameId string, userid string, msg string) {
	s.reader.Notify(isUniCast, gameId, userid, msg)
}


func (s *server)JoinGroupNet(groupId string){
	s.worker.JoinGroupNet(groupId)
}
