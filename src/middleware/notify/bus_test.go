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
	"fmt"
	"testing"
	"time"
)

func TestBus_Publish(t *testing.T) {
	ch := make(chan int, 1)
	go produce(ch)
	go consumer(ch)
	go consumer2(ch)
	time.Sleep(1 * time.Second)
	bus := NewBus()
	bus.Publish("test", &DummyMessage{})

}

func produce(ch chan<- int) {
	for i := 0; i < 20; i++ {
		ch <- i
		fmt.Println("Send:", i)
	}
}

func consumer(ch <-chan int) {
	for i := 0; i < 10; i++ {

		fmt.Println("Receive:", <-ch)
	}
}

func consumer2(ch <-chan int) {
	for i := 0; i < 10; i++ {

		fmt.Println("Receive2:", <-ch)
	}
}

func TestBus(t *testing.T) {
	bus := NewBus()
	bus.Subscribe("topic1", handler1)
	bus.Subscribe("topic2", handler2)
	bus.Subscribe("topic3", handler3)

	bus.Publish("topic1", &DummyMessage{})
	bus.Publish("topic2", &DummyMessage{})
	bus.Publish("topic3", &DummyMessage{})
}
