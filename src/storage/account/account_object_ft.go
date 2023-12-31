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

package account

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/utility"
	"math/big"
)

func (c *accountObject) getFT(db AccountDatabase, name string) *big.Int {
	value := c.GetData(db, utility.StrToBytes(common.GenerateFTKey(name)))
	if nil == value || 0 == len(value) {
		return nil
	}
	return new(big.Int).SetBytes(value)
}

func (c *accountObject) AddFT(db AccountDatabase, amount *big.Int, name string) bool {
	if amount.Sign() == 0 {
		if c.empty() {
			c.touch()
		}

		return true
	}

	raw := c.getFT(db, name)
	if nil == raw {
		return c.SetFT(db, new(big.Int).Set(amount), name)
	} else {
		return c.SetFT(db, new(big.Int).Add(raw, amount), name)
	}

}

func (c *accountObject) SubFT(db AccountDatabase, amount *big.Int, name string) (*big.Int, bool) {
	if amount.Sign() == 0 {
		raw := c.getFT(db, name)
		if nil == raw {
			return big.NewInt(0), true
		}
		return raw, true
	}

	raw := c.getFT(db, name)

	// 余额不足就滚粗
	if nil == raw || raw.Cmp(amount) == -1 {
		return nil, false
	}

	left := new(big.Int).Sub(raw, amount)
	c.SetFT(db, left, name)

	return left, true
}

func (self *accountObject) SetFT(db AccountDatabase, amount *big.Int, name string) bool {
	if nil == amount {
		return false
	}

	self.SetData(db, utility.StrToBytes(common.GenerateFTKey(name)), amount.Bytes())
	return true
}
