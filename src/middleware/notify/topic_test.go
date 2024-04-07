// Copyright 2020 The RangersProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RangersProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RangersProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package notify

import (
	"com.tuntun.rangers/node/src/middleware/log"
	"fmt"
	"testing"
	"time"
)

type DummyMessage struct {
}

func (d *DummyMessage) GetRaw() []byte {
	return []byte{}
}
func (d *DummyMessage) GetData() interface{} {
	return struct{}{}
}

func TestTopic_UnSubscribe2(t *testing.T) {
	busLogger = log.GetLoggerByIndex(log.BusLogConfig, "0")

	topic := &Topic{
		Id: "test",
	}

	h := handlerStruct{id: "1"}
	h1 := handlerStruct{id: "2"}
	topic.Subscribe(h)
	topic.Subscribe(h1)
	topic.Handle(&DummyMessage{})

	topic.UnSubscribe(h)
	topic.UnSubscribe(h)
	topic.Handle(&DummyMessage{})

	time.Sleep(2 * time.Second)
}

type handlerStruct struct {
	id string
}

func (h handlerStruct) HandleNetMessage(topic string, message Message) {
	fmt.Println("hello world: " + h.id + " " + topic)
}
