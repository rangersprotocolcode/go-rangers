package network

import (
	"log"

	"github.com/golang/protobuf/proto"
	"x/src/middleware/pb"
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
		log.Printf("Unmarshal message error:%s", e.Error())
		return nil, e
	}
	m := Message{Code: *message.Code, Body: message.Body}
	return &m, nil
}
