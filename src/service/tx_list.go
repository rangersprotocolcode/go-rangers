package service

import (
	"com.tuntun.rangers/node/src/middleware/types"
	"sort"
)

type nonceQueue []uint64

func (queue nonceQueue) Len() int           { return len(queue) }
func (queue nonceQueue) Less(i, j int) bool { return queue[i] < queue[j] }
func (queue nonceQueue) Swap(i, j int)      { queue[i], queue[j] = queue[j], queue[i] }

func (queue *nonceQueue) Push(nonce uint64) {
	*queue = append(*queue, nonce)
	sort.Sort(*queue)
}

func (queue *nonceQueue) Pop() uint64 {
	old := *queue
	x := old[0]
	*queue = old[1:]
	return x
}

// txList is a nonce->transaction  map with a heap based index to allow
// iterating over the contents in a nonce-incrementing way.
type txList struct {
	items map[uint64]*types.Transaction
	index *nonceQueue
}

func newTxList() *txList {
	list := txList{
		items: make(map[uint64]*types.Transaction),
		index: new(nonceQueue),
	}
	return &list
}

// Get retrieves the current transactions associated with the given nonce.
func (l *txList) Get(nonce uint64) *types.Transaction {
	return l.items[nonce]
}

func (l *txList) Size() int {
	return len(*l.index)
}

func (l *txList) GetHeadNonce() uint64 {
	return (*l.index)[0]
}

func (l *txList) GetTailNonce() uint64 {
	return (*l.index)[len(*l.index)-1]
}

// Put inserts a new transaction into the map, also updating the map's nonce
// index. If a transaction already exists with the same nonce, it's overwritten.
func (l *txList) Put(tx *types.Transaction) {
	if tx == nil {
		return
	}
	nonce := tx.Nonce
	if l.items[nonce] == nil {
		l.index.Push(nonce)
	}
	l.items[nonce] = tx
}

// Remove the first(smallest nonce) tx and return
func (l *txList) Pop() *types.Transaction {
	nonce := l.index.Pop()
	tx := l.items[nonce]
	delete(l.items, nonce)
	return tx
}

// Forward removes all transactions from the map with a nonce lower than the
// provided threshold. Every removed transaction is returned for any post-removal
// maintenance.
func (l *txList) Forward(threshold uint64) {
	for l.index.Len() > 0 && (*l.index)[0] <= threshold {
		nonce := l.index.Pop()
		delete(l.items, nonce)
	}
}
