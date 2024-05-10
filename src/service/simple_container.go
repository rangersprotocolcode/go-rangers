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
	"github.com/gogf/gf/container/gmap"
)

var (
	pendingTxListKey    = []byte("pending")
	pendingTxListPrefix = "tx_list_"
)

type simpleContainer struct {
	limit int
	data  *gmap.ListMap
	db    db.Database
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
	}
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

func (c *simpleContainer) isFull() bool {
	return c.data.Size() >= c.limit
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
	c.data.Set(tx.Hash, tx)
}

func (c *simpleContainer) batchRemove(txHashList []interface{}) {
	c.data.Removes(txHashList)
}

func (c *simpleContainer) remove(txHash common.Hash) {
	c.data.Remove(txHash)
}
