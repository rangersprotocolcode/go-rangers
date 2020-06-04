package service

import (
	"sync"
	"x/src/common"
	"x/src/middleware/types"
)

type simpleContainer struct {
	limit  int
	txs    types.Transactions
	txsMap map[common.Hash]*types.Transaction

	lock sync.RWMutex
}

func newSimpleContainer(l int) *simpleContainer {
	c := &simpleContainer{
		lock:   sync.RWMutex{},
		limit:  l,
		txsMap: map[common.Hash]*types.Transaction{},
		txs:    types.Transactions{},
	}

	return c
}

func (c *simpleContainer) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return len(c.txs)
}

func (c *simpleContainer) contains(key common.Hash) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.txsMap[key] != nil
}

func (c *simpleContainer) get(key common.Hash) *types.Transaction {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.txsMap[key]
}

func (c *simpleContainer) asSlice() []*types.Transaction {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.txs
}

func (c *simpleContainer) push(tx *types.Transaction) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.txs.Len() < c.limit {
		c.txs = append(c.txs, tx)
		c.txsMap[tx.Hash] = tx
		return
	}
}

func (c *simpleContainer) remove(key common.Hash) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.txsMap[key] == nil {
		return
	}

	delete(c.txsMap, key)
	for i, tx := range c.txs {
		if tx.Hash == key {
			c.txs = append(c.txs[:i], c.txs[i+1:]...)
			break
		}
	}
}
