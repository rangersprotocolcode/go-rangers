package service

import (
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/notify"
	"fmt"
	"testing"
)

func TestPriorityQueue_Swap(t *testing.T) {
	middleware.InitLock()

	pq := NewPriorityQueue()
	pq.SetThreshold(3)
	pq.SetHandle(printItem)

	x := notify.ClientTransactionMessage{Nonce: 5, UserId: "a"}
	pq.heapPush(&x)

	x1 := notify.ClientTransactionMessage{Nonce: 3, UserId: "b"}
	pq.heapPush(&x1)

	x2 := notify.ClientTransactionMessage{Nonce: 4, UserId: "c"}
	pq.heapPush(&x2)

	x3 := notify.ClientTransactionMessage{Nonce: 2, UserId: "c"}
	pq.heapPush(&x3)

	x4 := notify.ClientTransactionMessage{Nonce: 100, UserId: "c"}
	pq.heapPush(&x4)

}

func printItem(item *Item) {
	fmt.Println(item.Value.Nonce)
}
