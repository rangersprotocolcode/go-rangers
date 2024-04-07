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
	"reflect"
	"sync"
)

type Message interface {
	GetRaw() []byte
	GetData() interface{}
}

type DummyMessage struct {
}

func (d *DummyMessage) GetRaw() []byte {
	return []byte{}
}
func (d *DummyMessage) GetData() interface{} {
	return struct{}{}
}

type Handler func(message Message)

type Topic struct {
	Id       string
	handlers []Handler
	lock     sync.RWMutex
}

func (topic *Topic) Subscribe(h Handler) {
	topic.lock.Lock()
	defer topic.lock.Unlock()

	// check duplicated
	v := reflect.ValueOf(h).Pointer()
	for _, handler := range topic.handlers {
		v1 := reflect.ValueOf(handler).Pointer()
		if v == v1 {
			return
		}
	}

	topic.handlers = append(topic.handlers, h)
}

func (topic *Topic) UnSubscribe(h Handler) {
	topic.lock.Lock()
	defer topic.lock.Unlock()

	v := reflect.ValueOf(h).Pointer()
	for i, handler := range topic.handlers {
		v1 := reflect.ValueOf(handler).Pointer()
		if v == v1 {
			topic.handlers = append(topic.handlers[:i], topic.handlers[i+1:]...)
			return
		}
	}

}

func (topic *Topic) Handle(message Message) {
	topic.lock.RLock()
	defer topic.lock.RUnlock()

	if 0 == len(topic.handlers) {
		return
	}
	for _, h := range topic.handlers {
		go h(message)
	}
}
