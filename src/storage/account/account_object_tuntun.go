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
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package account

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/utility"
	"math/big"
)

var (
	dummyAppId = utility.StrToBytes("zero")
)

func (self *accountObject) checkAndCreate() {
	if self.empty() {
		self.touch()
	}
}
func (c *accountObject) getAllRefund(db AccountDatabase) map[common.Address]*big.Int {
	c.cachedLock.Lock()
	defer c.cachedLock.Unlock()

	result := make(map[common.Address]*big.Int)
	for key, value := range c.cachedStorage {
		result[common.BytesToAddress(utility.StrToBytes(key))] = new(big.Int).SetBytes(value)
	}

	iterator := c.DataIterator(db, []byte{})
	for iterator.Next() {
		cachedKey := utility.BytesToStr(iterator.Key)
		_, contains := c.cachedStorage[cachedKey]
		if !contains {
			c.cachedStorage[cachedKey] = iterator.Value
			result[common.BytesToAddress(iterator.Key)] = new(big.Int).SetBytes(iterator.Value)
		}

	}

	return result
}
