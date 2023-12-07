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

package middleware

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/middleware/notify"
	"container/heap"
)

type Item struct {
	Value *notify.ClientTransactionMessage

	index int
}

type PriorityQueue struct {
	data      []*Item
	threshold uint64

	handler func(message *Item)
}

func NewPriorityQueue() *PriorityQueue {
	pq := new(PriorityQueue)
	pq.data = make([]*Item, 0)
	heap.Init(pq)
	return pq
}

func (pq PriorityQueue) Len() int { return len(pq.data) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq.data[i].Value.Nonce < pq.data[j].Value.Nonce
}

func (pq PriorityQueue) Swap(i, j int) {
	pq.data[i], pq.data[j] = pq.data[j], pq.data[i]
	pq.data[i].index = i
	pq.data[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(pq.data)
	item := x.(*Item)
	item.index = n
	pq.data = append(pq.data, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := pq.data
	n := len(old)
	item := old[n-1]
	item.index = -1
	pq.data = old[0 : n-1]
	return item
}

func (pq *PriorityQueue) heapPush(value *notify.ClientTransactionMessage) {
	if value == nil {
		return
	}

	LockBlockchain("HeapPush")
	defer UnLockBlockchain("HeapPush")

	if 0 != value.Nonce && value.Nonce <= pq.threshold {
		return
	}

	x := new(Item)
	x.Value = value
	heap.Push(pq, x)

	pq.tryPop()
}

func (pq *PriorityQueue) tryPop() {
	if 0 == len(pq.data) || nil == pq.data[0] {
		return
	}

	for 0 < len(pq.data) && nil != pq.data[0] && pq.data[0].Value.Nonce <= pq.threshold {
		item := heap.Pop(pq).(*Item)
		if 0 == item.Value.Nonce && nil != pq.handler {
			pq.handler(item)
		}
	}

	for 0 < len(pq.data) && nil != pq.data[0] && pq.data[0].Value.Nonce == pq.threshold+1 {
		pq.threshold++
		item := heap.Pop(pq).(*Item)
		if nil != pq.handler {
			pq.handler(item)
		}
	}

	return
}

func (pq *PriorityQueue) SetThreshold(value uint64) {
	common.DefaultLogger.Debugf("setThreshold: %d", value)
	pq.threshold = value
	pq.tryPop()
}

func (pq *PriorityQueue) GetThreshold() uint64 {
	return pq.threshold
}

func (pq *PriorityQueue) SetHandle(handler func(message *Item)) {
	pq.handler = handler
}
