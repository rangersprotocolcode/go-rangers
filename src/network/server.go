package network

import (
	"context"
	"strings"
	"sync"
	"bufio"

	"utility"
	"middleware/pb"

	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-protocol"
	"github.com/libp2p/go-libp2p-peer"
	inet "github.com/libp2p/go-libp2p-net"
	"fmt"
)

const (
	packageLengthSize             = 4
	protocolID        protocol.ID = "/x/1.0.0"
)

var header = []byte{84, 85, 78}

var Server server

type server struct {
	host host.Host

	dht *dht.IpfsDHT

	streams map[string]inet.Stream

	streamMapLock sync.RWMutex
}

func initServer(host host.Host, dht *dht.IpfsDHT) {
	host.SetStreamHandler(protocolID, swarmStreamHandler)
	Server = server{host: host, dht: dht, streams: make(map[string]inet.Stream), streamMapLock: sync.RWMutex{}}
}

func (s *server) SendMessage(m Message, id string) {
	go func() {
		bytes, e := MarshalMessage(m)
		if e != nil {
			logger.Errorf("Marshal message error:%s", e.Error())
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
			logger.Errorf("New stream for %s error:%s", id, e.Error())
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
		logger.Errorf("Write stream for %s error:%s", id, err.Error())
		stream.Close()
		s.streams[id] = nil
		s.streamMapLock.Unlock()
		s.send(b, id)
		return
	}
	s.streamMapLock.Unlock()
	if r != l {
		logger.Errorf("Stream  should write %d byte ,bu write %d bytes", l, r)
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
		logger.Errorf("steam read 3 from %s error:%s!", id, e1.Error())
		return e1
	}
	if h != 3 {
		logger.Errorf("Stream  should read %d byte, but received %d bytes", 3, h)
		return nil
	}
	//校验 header
	if !(headerBytes[0] == header[0] && headerBytes[1] == header[1] && headerBytes[2] == header[2]) {
		logger.Errorf("validate header error from %s! ", id)
		return nil
	}

	pkgLengthBytes := make([]byte, packageLengthSize)
	n, err := reader.Read(pkgLengthBytes)
	if err != nil {
		logger.Errorf("Stream  read4 error:%s", err.Error())
		return nil
	}
	if n != 4 {
		logger.Errorf("Stream  should read %d byte, but received %d bytes", 4, n)
		return nil
	}
	pkgLength := int(utility.ByteToUInt32(pkgLengthBytes))
	b := make([]byte, pkgLength)
	e := readMessageBody(reader, b, 0)
	if e != nil {
		logger.Errorf("Stream  readMessageBody error:%s", e.Error())
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
	message := new(middleware_pb.Message)
	error := proto.Unmarshal(b, message)
	if error != nil {
		logger.Errorf("[Network]Proto unmarshal error:%s", error.Error())
	}

	//code := message.Code
	//switch *code {
	//case GROUP_MEMBER_MSG, GROUP_INIT_MSG, KEY_PIECE_MSG, SIGN_PUBKEY_MSG, GROUP_INIT_DONE_MSG, CURRENT_GROUP_CAST_MSG, CAST_VERIFY_MSG,
	//	VARIFIED_CAST_MSG:
	//	consensusHandler.HandlerMessage(*code, message.Body, from)
	//case REQ_TRANSACTION_MSG, REQ_BLOCK_CHAIN_TOTAL_QN_MSG, BLOCK_CHAIN_TOTAL_QN_MSG, REQ_BLOCK_INFO, BLOCK_INFO,
	//	REQ_GROUP_CHAIN_HEIGHT_MSG, GROUP_CHAIN_HEIGHT_MSG, REQ_GROUP_MSG, GROUP_MSG, BLOCK_HASHES_REQ, BLOCK_HASHES:
	//	chainHandler.HandlerMessage(*code, message.Body, from)
	//case NEW_BLOCK_MSG:
	//	consensusHandler.HandlerMessage(*code, message.Body, from)
	//case TRANSACTION_MSG, TRANSACTION_GOT_MSG:
	//	_, e := chainHandler.HandlerMessage(*code, message.Body, from)
	//	if e != nil {
	//		return
	//	}
	//	consensusHandler.HandlerMessage(*code, message.Body, from)
	//}

	fmt.Printf("Reviced message from %s,code %d,msg len:%d\n", from, message.Code, len(message.Body))
}

type ConnInfo struct {
	Id      string `json:"id"`
	Ip      string `json:"ip"`
	TcpPort string `json:"tcp_port"`
}

func (s *server) GetConnInfo() []ConnInfo {
	conns := s.host.Network().Conns()
	result := []ConnInfo{}
	for _, conn := range conns {
		id := idToString(conn.RemotePeer())
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
		c := ConnInfo{Id: id, Ip: ip, TcpPort: port}
		result = append(result, c)
	}
	return result
}

func idToString(p peer.ID) string {
	return p.Pretty()
}
