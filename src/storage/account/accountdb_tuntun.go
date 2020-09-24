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

package account

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"fmt"
	"math/big"
)

func (self *AccountDB) GetFT(addr common.Address, ftName string) *big.Int {
	accountObject := self.getOrNewAccountObject(addr)
	data := accountObject.data
	if 0 == len(data.Ft) {
		return big.NewInt(0)
	}

	raw := accountObject.getFT(accountObject.data.Ft, ftName)
	if raw == nil {
		return big.NewInt(0)
	}
	return raw.Balance
}

func (self *AccountDB) GetAllFT(addr common.Address) map[string]*big.Int {
	accountObject := self.getOrNewAccountObject(addr)
	data := accountObject.data
	if 0 == len(data.Ft) {
		return nil
	}

	result := make(map[string]*big.Int, len(data.Ft))
	for _, value := range data.Ft {
		result[value.ID] = value.Balance
	}
	return result
}

func (self *AccountDB) SetFT(addr common.Address, ftName string, balance *big.Int) {
	if nil == balance {
		return
	}
	account := self.getOrNewAccountObject(addr)
	account.SetFT(balance, ftName)
}

func (self *AccountDB) AddFT(addr common.Address, ftName string, balance *big.Int) bool {
	if nil == balance {
		return true
	}
	account := self.getOrNewAccountObject(addr)

	return account.AddFT(balance, ftName)
}

func (self *AccountDB) SubFT(addr common.Address, ftName string, balance *big.Int) (*big.Int, bool) {
	if nil == balance {
		return nil, false
	}
	account := self.getOrNewAccountObject(addr)
	return account.SubFT(balance, ftName)

}

// 根据setId/id 查找NFT
func (self *AccountDB) GetNFTById(addr common.Address, setId, id string) *types.NFT {
	accountObject := self.getOrNewAccountObject(addr)
	return accountObject.getNFTById(self.db, setId, id)
}

func (self *AccountDB) GetAllNFTByGameId(addr common.Address, appId string) []*types.NFT {
	accountObject := self.getOrNewAccountObject(addr)
	return accountObject.getAllNFT(self.db, appId)
}

func (self *AccountDB) GetAllNFT(addr common.Address) []*types.NFT {
	accountObject := self.getOrNewAccountObject(addr)
	return accountObject.getAllNFT(self.db, "")
}

func (self *AccountDB) AddNFTByGameId(addr common.Address, appId string, nft *types.NFT) bool {
	stateObject := self.getOrNewAccountObject(addr)
	return stateObject.AddNFTByGameId(self.db, appId, nft)
}

func (self *AccountDB) SetNFTValueByGameId(addr common.Address, appId, setId, id, value string) bool {
	stateObject := self.getOrNewAccountObject(addr)
	return stateObject.SetNFTValueByGameId(self.db, appId, setId, id, value)
}

func (self *AccountDB) RemoveNFTByGameId(addr common.Address, appId, setId, id string) bool {
	stateObject := self.getOrNewAccountObject(addr)
	return stateObject.RemoveNFT(self.db, appId, setId, id)
}

func (self *AccountDB) RemoveNFT(addr common.Address, nft *types.NFT) bool {
	stateObject := self.getOrNewAccountObject(addr)
	return stateObject.RemoveNFT(self.db, nft.AppId, nft.SetID, nft.ID)
}

func (self *AccountDB) ApproveNFT(owner common.Address, appId, setId, id, renter string) bool {
	stateObject := self.getOrNewAccountObject(owner)
	return stateObject.ApproveNFT(self.db, appId, setId, id, renter)
}

func (self *AccountDB) ChangeNFTStatus(owner common.Address, appId, setId, id string, status byte) bool {
	stateObject := self.getOrNewAccountObject(owner)

	if 0 == len(appId) {
		return stateObject.ChangeNFTStatusById(self.db, setId, id, status)
	}
	return stateObject.ChangeNFTStatus(self.db, appId, setId, id, status)
}

func (adb *AccountDB) GetNFTSet(setId string) *types.NFTSet {
	accountObject := adb.getAccountObject(adb.generateNFTSetAddress(setId), false)
	if nil == accountObject {
		return nil
	}

	return accountObject.GetNFTSet(adb.db)
}

func (adb *AccountDB) generateNFTSetAddress(setId string) common.Address {
	address := fmt.Sprintf("n-%s", setId)
	return common.StringToAddress(address)
}
