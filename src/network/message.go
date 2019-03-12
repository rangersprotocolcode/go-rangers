package network

import (
	"middleware/pb"
	
	"github.com/gogo/protobuf/proto"
)

type Message struct {
	Code uint32

	Body []byte
}

func MarshalMessage(m Message) ([]byte, error) {
	message := middleware_pb.Message{Code: &m.Code, Body: m.Body}
	return proto.Marshal(&message)
}

func UnMarshalMessage(b []byte) (*Message, error) {
	message := new(middleware_pb.Message)
	e := proto.Unmarshal(b, message)
	if e != nil {
		logger.Errorf("Unmarshal message error:%s", e.Error())
		return nil, e
	}
	m := Message{Code: *message.Code, Body: message.Body}
	return &m, nil
}
