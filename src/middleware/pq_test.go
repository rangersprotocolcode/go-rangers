package middleware

import (
	"com.tuntun.rocket/node/src/middleware/notify"
	"fmt"
	"testing"
)

func TestPriorityQueue_Swap(t *testing.T) {
	InitLock()

	pq := NewPriorityQueue()
	pq.SetThreshold(3)
	pq.SetHandle(printItem)

	x0 := notify.ClientTransactionMessage{Nonce: 0, UserId: "0"}
	pq.heapPush(&x0)

	x := notify.ClientTransactionMessage{Nonce: 5, UserId: "a"}
	pq.heapPush(&x)

	x1 := notify.ClientTransactionMessage{Nonce: 3, UserId: "b"}
	pq.heapPush(&x1)

	x00 := notify.ClientTransactionMessage{Nonce: 0, UserId: "00"}
	pq.heapPush(&x00)

	x000 := notify.ClientTransactionMessage{Nonce: 0, UserId: "000"}
	pq.heapPush(&x000)

	x2 := notify.ClientTransactionMessage{Nonce: 4, UserId: "c"}
	pq.heapPush(&x2)

	x0000 := notify.ClientTransactionMessage{Nonce: 0, UserId: "0000"}
	pq.heapPush(&x0000)

	x3 := notify.ClientTransactionMessage{Nonce: 2, UserId: "c"}
	pq.heapPush(&x3)

	x4 := notify.ClientTransactionMessage{Nonce: 100, UserId: "c"}
	pq.heapPush(&x4)

}

func printItem(item *Item) {
	fmt.Printf("nonce: %d, value: %s\n", item.Value.Nonce, item.Value.UserId)
}
