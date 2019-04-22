package network

import (
	"encoding/hex"
	"x/src/utility"
	"bytes"
	"strconv"
	"github.com/gorilla/websocket"
	"hash/fnv"
)

var methodCodeClientSend, _ = hex.DecodeString("80000000")
var methodCodeSend, _ = hex.DecodeString("80000001")
var methodCodeBroadcast, _ = hex.DecodeString("80000002")
var methodCodeSendToGroup, _ = hex.DecodeString("80000003")
var methodCodeJoinGroup, _ = hex.DecodeString("80000004")
var methodCodeQuitGroup, _ = hex.DecodeString("80000005")

type header struct {
	method   []byte
	sourceId uint64
	targetId uint64
	nonce    uint64
}

func loadWebSocketMsg(header header, body []byte) []byte {
	h := header.toBytes()

	message := make([]byte, protocolHeaderSize+len(body))
	copy(message[:protocolHeaderSize], h[:])
	copy(message[protocolHeaderSize:], body)
	return message
}

func unloadWebSocketMsg(m []byte) (header header, body []byte) {
	if len(m) < protocolHeaderSize {
		return header, nil
	}

	header = byteToHeader(m[:protocolHeaderSize])
	body = m[protocolHeaderSize:]
	Logger.Debugf("Rcv msg header:%v,body:%v", header, body)
	return
}

func (h *header) toBytes() []byte {
	byte := make([]byte, protocolHeaderSize)
	copy(byte[0:4], h.method)
	copy(byte[12:20], utility.UInt64ToByte(h.targetId))
	copy(byte[20:28], utility.UInt64ToByte(h.nonce))
	return byte
}

func byteToHeader(b []byte) header {
	header := header{}
	header.method = b[0:4]
	header.sourceId = utility.ByteToUInt64(b[4:12])
	header.nonce = utility.ByteToUInt64(b[20:])
	return header
}

func (s *server) send(method []byte, targetId string, msg Message, nonce uint64) {
	m, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("marshal message error:%s", err.Error())
		return
	}

	header := header{method: method, nonce:nonce}

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
	header.targetId = target
	message := loadWebSocketMsg(header, m)

	Logger.Debugf("Send msg:%v", message)
	s.sendChan <- message
}

func (s *server) receiveMessage() {
	for {
		_, message, err := s.conn.ReadMessage()
		if err != nil {
			Logger.Errorf("Rcv msg error:%s", err.Error())
			continue
		}
		Logger.Debugf("Rcv msg:%v", message)
		s.rcvChan <- message
	}
}

func (s *server) loop() {
	for {
		select {
		case message := <-s.rcvChan:
			header, data := unloadWebSocketMsg(message)
			if bytes.Equal(header.method, methodCodeClientSend) {
				s.handleClientMessage(data, strconv.FormatUint(header.sourceId, 10), header.nonce)
			} else {
				go s.handleMinerMessage(data, strconv.FormatUint(header.sourceId, 10))
			}
		case message := <-s.sendChan:
			err := s.conn.WriteMessage(websocket.BinaryMessage, message)
			if err != nil {
				Logger.Errorf("Send msg error:%s", err.Error())
			}

		}
	}
}
