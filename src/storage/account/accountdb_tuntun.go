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

func (self *AccountDB) GetFT(addr common.Address, ftName string) *big.Int {
	// check if erc-20
	found, contract, position, decimal := self.GetERC20Binding(ftName)
	if found {
		account := self.getOrNewAccountObject(contract)
		data := account.GetData(self.db, self.GetERC20Key(addr, position))
		result := new(big.Int).SetBytes(data)
		return utility.FormatDecimalForRocket(result, int64(decimal))
	}

	accountObject := self.getOrNewAccountObject(addr)
	raw := accountObject.getFT(self.db, ftName)
	if raw == nil {
		return big.NewInt(0)
	}
	return raw
}

func (self *AccountDB) GetAllRefund(addr common.Address) map[common.Address]*big.Int {
	accountObject := self.getOrNewAccountObject(addr)
	return accountObject.getAllRefund(self.db)
}

func (self *AccountDB) SetFT(addr common.Address, ftName string, balance *big.Int) {
	if nil == balance {
		return
	}
	// check if erc-20
	found, contract, position, decimal := self.GetERC20Binding(ftName)
	if found {
		account := self.getOrNewAccountObject(contract)
		account.SetData(self.db, self.GetERC20Key(addr, position), utility.FormatDecimalForERC20(balance, int64(decimal)).Bytes())
		return
	}

	account := self.getOrNewAccountObject(addr)
	account.SetFT(self.db, balance, ftName)
}

func (self *AccountDB) AddFT(addr common.Address, ftName string, balance *big.Int) bool {
	if nil == balance {
		return true
	}

	// check if erc-20
	found, contract, position, decimal := self.GetERC20Binding(ftName)
	if found {
		account := self.getOrNewAccountObject(contract)
		key := self.GetERC20Key(addr, position)
		remain := new(big.Int).SetBytes(account.GetData(self.db, key))
		remain.Add(remain, utility.FormatDecimalForERC20(balance, int64(decimal)))
		if common.IsProposal002() {
			account.SetData(self.db, key, remain.Bytes())
		} else {
			account.setData(key, remain.Bytes())
		}
		return true
	}

	account := self.getOrNewAccountObject(addr)
	return account.AddFT(self.db, balance, ftName)
}

func (self *AccountDB) SubFT(addr common.Address, ftName string, balance *big.Int) (*big.Int, bool) {
	if nil == balance {
		return nil, false
	}

	// check if erc-20
	found, contract, position, decimal := self.GetERC20Binding(ftName)
	if found {
		account := self.getOrNewAccountObject(contract)

		key := self.GetERC20Key(addr, position)
		remain := new(big.Int).SetBytes(account.GetData(self.db, key))

		value := utility.FormatDecimalForERC20(balance, int64(decimal))
		if remain.Cmp(value) < 0 {
			return remain, false
		}

		remain.Sub(remain, value)
		if common.IsProposal002() {
			account.SetData(self.db, key, remain.Bytes())
		} else {
			account.setData(key, remain.Bytes())
		}
		return utility.FormatDecimalForRocket(remain, int64(decimal)), true
	}

	account := self.getOrNewAccountObject(addr)
	return account.SubFT(self.db, balance, ftName)

}
