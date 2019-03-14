package network

import (
	"fmt"
	"context"
	"strings"
	"sync"
	"bufio"

	"x/src/utility"
	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-protocol"
	"github.com/libp2p/go-libp2p-peer"
	inet "github.com/libp2p/go-libp2p-net"
	"time"
	"math/rand"
	"x/src/middleware/notify"
)

const (
	packageLengthSize             = 4
	protocolID        protocol.ID = "/x/1.0.0"
)

var proposerList = []string{"111", "2222", "333"}

var verifyGroupList = []string{"group1"}
var verifyGroupsInfo = map[string][]string{"group1": {"memberA", "memberB"}}

var header = []byte{84, 85, 78}

var Server server

type server struct {
	host host.Host

	dht *dht.IpfsDHT

	streams map[string]inet.Stream

	consensusHandler MsgHandler

	streamMapLock sync.RWMutex
}

func initServer(host host.Host, dht *dht.IpfsDHT, consensusHandler MsgHandler) {
	host.SetStreamHandler(protocolID, swarmStreamHandler)
	Server = server{host: host, dht: dht, streams: make(map[string]inet.Stream), streamMapLock: sync.RWMutex{}, consensusHandler: consensusHandler}
}

func (s *server) Send(id string, msg Message) {
	go func() {
		bytes, e := marshalMessage(msg)
		if e != nil {
			Logger.Errorf("Marshal message error:%s", e.Error())
			return
		}

		length := len(bytes)
		b2 := utility.UInt32ToByte(uint32(length))

		b := make([]byte, len(bytes)+len(b2)+3, len(bytes)+len(b2)+3)
		copy(b[:3], header[:])
		copy(b[3:7], b2)
		copy(b[7:], bytes)

		s.send(b, id)
	}()

}

func (s *server) SpreadAmongGroup(groupId string, msg Message) {
	members := s.getMembers(groupId)
	if members == nil || len(members) == 0 {
		Logger.Errorf("Unknown group:%s,discard sending message", groupId)
		return
	}

	for _, member := range members {
		s.Send(member, msg)
	}
}

func (s *server) SpreadToRandomGroupMember(groupId string, groupMembers []string, msg Message) {
	members := s.getMembers(groupId)
	if members == nil || len(members) == 0 {
		Logger.Errorf("Unknown group:%s,discard sending message", groupId)
		return
	}

	rand := rand.New(rand.NewSource(time.Now().Unix()))
	index := rand.Intn(len(members))
	randMembers := groupMembers[index:]

	for _, member := range randMembers {
		s.Send(member, msg)
	}
}

func (s *server) TransmitToNeighbor(msg Message) {
	conns := s.host.Network().Conns()
	for _, conn := range conns {
		id := conn.RemotePeer()
		if id == "" {
			continue
		}
		Logger.Debugf("transmit to neighbor:%s", idToString(id))
		s.Send(idToString(id), msg)
	}
}

func (s *server) Broadcast(msg Message) {
	for _, proposer := range proposerList {
		s.Send(proposer, msg)
	}

	for _, verifyMembers := range verifyGroupsInfo {
		for _, verifier := range verifyMembers {
			s.Send(verifier, msg)
		}
	}
}

func (s *server) ConnInfo() []Conn {
	conns := s.host.Network().Conns()
	result := make([]Conn, 0, 0)
	for _, conn := range conns {
		id := conn.RemotePeer()
		if id == "" {
			continue
		}
		addr := conn.RemoteMultiaddr().String()
		//addr /ip4/127.0.0.1/udp/1234"
		split := strings.Split(addr, "/")
		if len(split) != 5 {
			continue
		}
		ip := split[2]
		port := split[4]
		c := Conn{Id: idToString(id), Ip: ip, Port: port}
		result = append(result, c)
	}
	return result
}

func (s *server) getMembers(groupId string) []string {
	if groupId == FullNodeVirtualGroupId {
		return proposerList
	}
	for _, g := range verifyGroupList {
		if g == groupId {
			return verifyGroupsInfo[groupId]
		}
	}
	return nil
}

func (s *server) send(b []byte, id string) {
	if id == idToString(s.host.ID()) {
		s.sendSelf(b, id)
		return
	}
	c := context.Background()

	s.streamMapLock.Lock()
	stream := s.streams[id]
	if stream == nil {
		var e error
		stream, e = s.host.NewStream(c, strToId(id), protocolID)
		if e != nil {
			Logger.Errorf("New stream for %s error:%s", id, e.Error())
			fmt.Printf("New stream for %s error:%s", id, e.Error())
			s.streamMapLock.Unlock()
			s.send(b, id)
			return
		}
		s.streams[id] = stream
	}

	l := len(b)
	r, err := stream.Write(b)
	if err != nil {
		Logger.Errorf("Write stream for %s error:%s", id, err.Error())
		stream.Close()
		s.streams[id] = nil
		s.streamMapLock.Unlock()
		s.send(b, id)
		return
	}
	s.streamMapLock.Unlock()
	if r != l {
		Logger.Errorf("Stream  should write %d byte ,bu write %d bytes", l, r)
		return
	}
}

func (s *server) sendSelf(b []byte, id string) {
	pkgBodyBytes := b[7:]
	s.handleMessage(pkgBodyBytes, id, b[3:7])
}

//TODO 考虑读写超时
func swarmStreamHandler(stream inet.Stream) {
	go func() {
		for {
			e := handleStream(stream)
			if e != nil {
				stream.Close()
				break
			}
		}
	}()
}
func handleStream(stream inet.Stream) error {
	id := idToString(stream.Conn().RemotePeer())
	reader := bufio.NewReader(stream)
	//defer stream.Close()
	headerBytes := make([]byte, 3)
	h, e1 := reader.Read(headerBytes)
	if e1 != nil {
		Logger.Errorf("steam read 3 from %s error:%s!", id, e1.Error())
		return e1
	}
	if h != 3 {
		Logger.Errorf("Stream  should read %d byte, but received %d bytes", 3, h)
		return nil
	}
	//校验 header
	if !(headerBytes[0] == header[0] && headerBytes[1] == header[1] && headerBytes[2] == header[2]) {
		Logger.Errorf("validate header error from %s! ", id)
		return nil
	}

	pkgLengthBytes := make([]byte, packageLengthSize)
	n, err := reader.Read(pkgLengthBytes)
	if err != nil {
		Logger.Errorf("Stream  read4 error:%s", err.Error())
		return nil
	}
	if n != 4 {
		Logger.Errorf("Stream  should read %d byte, but received %d bytes", 4, n)
		return nil
	}
	pkgLength := int(utility.ByteToUInt32(pkgLengthBytes))
	b := make([]byte, pkgLength)
	e := readMessageBody(reader, b, 0)
	if e != nil {
		Logger.Errorf("Stream  readMessageBody error:%s", e.Error())
		return e
	}
	go Server.handleMessage(b, id, pkgLengthBytes)
	return nil
}

func readMessageBody(reader *bufio.Reader, body []byte, index int) error {
	if index == 0 {
		n, err1 := reader.Read(body)
		if err1 != nil {
			return err1
		}
		if n != len(body) {
			return readMessageBody(reader, body, n)
		}
		return nil
	} else {
		b := make([]byte, len(body)-index)
		n, err2 := reader.Read(b)
		if err2 != nil {
			return err2
		}
		copy(body[index:], b[:])
		if n != len(b) {
			return readMessageBody(reader, body, index+n)
		}
		return nil
	}

}
func (s *server) handleMessage(b []byte, from string, lengthByte []byte) {
	message, error := unMarshalMessage(b)
	if error != nil {
		Logger.Errorf("Proto unmarshal error:%s", error.Error())
		return
	}
	Logger.Debugf("Receive message from %s,code:%d,msg size:%d,hash:%s", from, message.Code, len(b), message.Hash())

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

func idToString(p peer.ID) string {
	return p.Pretty()
}
