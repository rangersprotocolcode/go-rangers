package service

import (
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/notify"
	"container/heap"
)

type Item struct {
	Value *notify.ClientTransactionMessage

	index int // 元素在堆中的索引。
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
	item.index = -1 // 为了安全性考虑而做的设置
	pq.data = old[0 : n-1]
	return item
}

func (pq *PriorityQueue) heapPush(value *notify.ClientTransactionMessage) {
	if value == nil {
		return
	}

	middleware.LockBlockchain("HeapPush")
	defer middleware.UnLockBlockchain("HeapPush")

	if value.Nonce < pq.threshold {
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

	for i := pq.data[0].Value.Nonce; i <= pq.threshold && 0 < len(pq.data); i++ {
		heap.Pop(pq)
	}

	for 0 < len(pq.data) && nil != pq.data[0] && pq.data[0].Value.Nonce == pq.threshold+1 {
		pq.threshold++
		if nil != pq.handler {
			pq.handler(heap.Pop(pq).(*Item))
		}
	}

	return
}

func (pq *PriorityQueue) SetThreshold(value uint64) {
	pq.threshold = value
	pq.tryPop()
}

func (pq *PriorityQueue) GetThreshold() uint64 {
	return pq.threshold
}

func (pq *PriorityQueue) SetHandle(handler func(message *Item)) {
	pq.handler = handler
}
