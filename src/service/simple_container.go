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

package service

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/middleware/db"
	"com.tuntun.rangers/node/src/middleware/types"
	"encoding/json"
	"github.com/gogf/gf/container/gmap"
	"sync"
	"time"
)

const (
	expiredRing     = 2
	txCycleInterval = time.Minute * 1
)

var (
	pendingTxListKey    = []byte("pending")
	pendingTxListPrefix = "tx_list_"
)

type simpleContainer struct {
	limit int
	data  *gmap.ListMap
	db    db.Database

	txAnnualRingMap sync.Map
	txCycleTicker   *time.Ticker
}

func newSimpleContainer(l int) *simpleContainer {
	db, err := db.NewDatabase(pendingTxListPrefix)
	if err != nil {
		txPoolLogger.Error("init simple container db error:%s", err.Error())
	}

	c := &simpleContainer{
		data:  gmap.NewListMap(true),
		limit: l,
		db:    db,

		txAnnualRingMap: sync.Map{},
		txCycleTicker:   time.NewTicker(txCycleInterval),
	}
	c.loadPendingTxList()
	go c.loop()
	return c
}

func (c *simpleContainer) Close() {
	if c.db != nil {
		c.db.Close()
	}
}

func (c *simpleContainer) Len() int {
	return c.data.Size()
}

func (c *simpleContainer) contains(key common.Hash) bool {
	return c.data.Contains(key)
}

func (c *simpleContainer) get(key common.Hash) *types.Transaction {
	item := c.data.Get(key)
	if nil == item {
		return nil
	}

	return item.(*types.Transaction)
}

func (c *simpleContainer) asSlice() []interface{} {
	return c.data.Values()
}

func (c *simpleContainer) push(tx *types.Transaction) {
	if c.data.Size() < c.limit {
		c.data.Set(tx.Hash, tx)
		c.txAnnualRingMap.Store(tx.Hash, uint64(0))
		c.flush()
	}
}

func (c *simpleContainer) remove(txHashList []interface{}) {
	c.data.Removes(txHashList)
	for _, item := range txHashList {
		c.txAnnualRingMap.Delete(item)
	}
	c.flush()
}

func (c *simpleContainer) flush() {
	txList := c.asSlice()
	data, err := json.Marshal(txList)
	if err != nil {
		txPoolLogger.Error("json marshal pending tx list error:%s", err.Error())
		return
	}
	err = c.db.Put(pendingTxListKey, data)
	if err != nil {
		txPoolLogger.Error("pending tx list db put error:%s", err.Error())
	}
}

func (c *simpleContainer) loadPendingTxList() {
	data, _ := c.db.Get(pendingTxListKey)
	if data == nil {
		return
	}

	var pendingTxList []*types.Transaction
	err := json.Unmarshal(data, &pendingTxList)
	if err != nil {
		txPoolLogger.Error("json unmarshal pending tx list error:%s", err.Error())
		return
	}
	for _, item := range pendingTxList {
		c.data.Set(item.Hash, item)
		c.txAnnualRingMap.Store(item.Hash, uint64(0))
		txPoolLogger.Debugf("load pending tx:%s", item.Hash.String())
	}
}

func (c *simpleContainer) loop() {
	for {
		select {
		case <-c.txCycleTicker.C:
			go c.growRing()
		}
	}
}

func (c *simpleContainer) growRing() {
	expiredTxHashList := make([]interface{}, 0)

	c.txAnnualRingMap.Range(func(key, value interface{}) bool {
		txHash := key.(common.Hash)
		txRing := value.(uint64)
		txRing = txRing + 1
		c.txAnnualRingMap.Store(txHash, txRing)
		if txRing >= expiredRing {
			expiredTxHashList = append(expiredTxHashList, txHash)
			txPoolLogger.Debugf("pending tx expired,drop it. hash:%s", txHash.String())
		}
		return true
	})

	if len(expiredTxHashList) > 0 {
		c.remove(expiredTxHashList)
	}
}
