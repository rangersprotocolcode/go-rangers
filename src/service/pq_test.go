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

	x := notify.ClientTransactionMessage{Nonce: 5, UserId: "a"}
	pq.HeapPush(&x)

	x1 := notify.ClientTransactionMessage{Nonce: 3, UserId: "b"}
	pq.HeapPush(&x1)

	x2 := notify.ClientTransactionMessage{Nonce: 4, UserId: "c"}
	pq.HeapPush(&x2)

	x3 := notify.ClientTransactionMessage{Nonce: 2, UserId: "c"}
	pq.HeapPush(&x3)

	x4 := notify.ClientTransactionMessage{Nonce: 100, UserId: "c"}
	pq.HeapPush(&x4)

	pq.SetThreshold(3)
	pq.TryPop(printItem)

}

func printItem(item *Item) {
	fmt.Println(item.Value.Nonce)
}
