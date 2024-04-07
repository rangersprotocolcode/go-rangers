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
	"sync"
)

type Message interface {
	GetRaw() []byte
	GetData() interface{}
}

type Handler interface {
	HandleNetMessage(topic string, message Message)
}

type Topic struct {
	Id       string
	handlers []Handler
	lock     sync.RWMutex
}

func (topic *Topic) Subscribe(h Handler) {
	topic.lock.Lock()
	defer topic.lock.Unlock()

	// check duplicated
	for _, handler := range topic.handlers {
		if h == handler {
			busLogger.Errorf("fail to subscribe, id: %s", topic.Id)
			return
		}
	}

	topic.handlers = append(topic.handlers, h)
	busLogger.Infof("subscribe, id: %s, len: %d", topic.Id, len(topic.handlers))
}

func (topic *Topic) UnSubscribe(h Handler) {
	topic.lock.Lock()
	defer topic.lock.Unlock()

	length := len(topic.handlers)
	for i, handler := range topic.handlers {
		if h == handler {
			topic.handlers = append(topic.handlers[:i], topic.handlers[i+1:]...)
			busLogger.Infof("unsubscribe, id: %s, len: %d-> %d", topic.Id, length, len(topic.handlers))
			return
		}
	}

	busLogger.Infof("unsubscribe, id: %s, len: %d-> %d", topic.Id, length, len(topic.handlers))
}

func (topic *Topic) Handle(message Message) {
	topic.lock.RLock()
	defer topic.lock.RUnlock()

	if 0 == len(topic.handlers) {
		return
	}
	for _, h := range topic.handlers {
		go h.HandleNetMessage(topic.Id, message)
	}
	busLogger.Debugf("handle, id: %s, len: %d", topic.Id, len(topic.handlers))
}
