package network

import (
	"com.tuntun.rocket/node/src/middleware/pb"

	"com.tuntun.rocket/node/src/common"
	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/sha3"
)

// rocket Protocol 消息
type Message struct {
	Code uint32

	Body []byte
}

func marshalMessage(m Message) ([]byte, error) {
	message := middleware_pb.Message{Code: &m.Code, Body: m.Body}
	return proto.Marshal(&message)
}

func unMarshalMessage(b []byte) (*Message, error) {
	message := new(middleware_pb.Message)
	e := proto.Unmarshal(b, message)
	if e != nil {
		Logger.Errorf("Unmarshal message error:%s", e.Error())
		return nil, e
	}
	m := Message{Code: *message.Code, Body: message.Body}
	return &m, nil
}

func (m Message) Hash() string {
	bytes, err := marshalMessage(m)
	if err != nil {
		return ""
	}

	var h common.Hash
	sha3Hash := sha3.Sum256(bytes)
	if len(sha3Hash) == common.HashLength {
		copy(h[:], sha3Hash[:])
	} else {
		panic("Data2Hash failed, size error.")
	}
	return h.String()
}
