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
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package notify

import (
	"fmt"
	"testing"
)

// hello world2
// hello world
func TestTopic_Subscribe(t *testing.T) {
	topic := &Topic{
		Id: "test",
	}

	topic.Subscribe(handler1)
	topic.Subscribe(handler2)
	topic.Handle(&DummyMessage{})
}

// hello world2
func TestTopic_UnSubscribe0(t *testing.T) {
	topic := &Topic{
		Id: "test",
	}

	topic.Subscribe(handler1)
	topic.Subscribe(handler2)

	topic.UnSubscribe(handler1)
	topic.Handle(&DummyMessage{})
}

// hello world3
// hello world
func TestTopic_UnSubscribe1(t *testing.T) {
	topic := &Topic{
		Id: "test",
	}

	topic.Subscribe(handler1)
	topic.Subscribe(handler2)
	topic.Subscribe(handler3)

	topic.UnSubscribe(handler2)
	topic.Handle(&DummyMessage{})
}

// hello world
// hello world2
func TestTopic_UnSubscribe2(t *testing.T) {
	topic := &Topic{
		Id: "test",
	}

	topic.Subscribe(handler1)
	topic.Subscribe(handler2)
	topic.Subscribe(handler3)

	topic.UnSubscribe(handler3)
	topic.Handle(&DummyMessage{})
}

func handler1(message Message) {
	fmt.Println("hello world")
}

func handler2(message Message) {
	fmt.Println("hello world2")
}

func handler3(message Message) {
	fmt.Println("hello world3")
}
