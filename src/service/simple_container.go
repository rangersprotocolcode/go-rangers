// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package service

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"sync"
)

type simpleContainer struct {
	limit  int
	txs    types.Transactions
	txsMap map[common.Hash]*types.Transaction
	lock   sync.RWMutex
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
	//c.lock.RLock()
	//defer c.lock.RUnlock()

	return c.txs
}

func (c *simpleContainer) push(tx *types.Transaction) {
	//c.lock.Lock()
	//defer c.lock.Unlock()

	if c.txs.Len() < c.limit {
		c.txs = append(c.txs, tx)
		c.txsMap[tx.Hash] = tx
		return
	}
}

func (c *simpleContainer) remove(txHashList []common.Hash) {
	//c.lock.Lock()
	//defer c.lock.Unlock()
	for _, txHash := range txHashList {
		delete(c.txsMap, txHash)
		for i, tx := range c.txs {
			if tx.Hash == txHash {
				c.txs = append(c.txs[:i], c.txs[i+1:]...)
				break
			}
		}
	}
}
