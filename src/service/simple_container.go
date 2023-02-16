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
	"github.com/gogf/gf/container/gmap"
)

type simpleContainer struct {
	limit int
	data  *gmap.ListMap
}

func newSimpleContainer(l int) *simpleContainer {
	c := &simpleContainer{
		data:  gmap.NewListMap(true),
		limit: l,
	}

	return c
}

func (c *simpleContainer) Len() int {
	return c.data.Size()
}

func (c *simpleContainer) contains(key common.Hash) bool {
	return c.data.Contains(key)
}

func (c *simpleContainer) get(key common.Hash) *types.Transaction {
	item := c.data.Get(key)
	return item.(*types.Transaction)
}

func (c *simpleContainer) asSlice() []interface{} {
	return c.data.Values()
}

func (c *simpleContainer) push(tx *types.Transaction) {
	if c.data.Size() < c.limit {
		c.data.Set(tx.Hash, tx)
	}
}

func (c *simpleContainer) remove(txHashList []interface{}) {
	c.data.Removes(txHashList)
}
