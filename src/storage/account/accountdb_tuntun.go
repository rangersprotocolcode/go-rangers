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
	"math/big"
)

func (self *AccountDB) GetFT(addr common.Address, ftName string) *big.Int {
	accountObject := self.getOrNewAccountObject(addr)
	raw := accountObject.getFT(self.db, ftName)
	if raw == nil {
		return big.NewInt(0)
	}
	return raw
}

func (self *AccountDB) GetAllFT(addr common.Address) map[string]*big.Int {
	accountObject := self.getOrNewAccountObject(addr)
	return accountObject.getAllFT(self.db)
}

func (self *AccountDB) SetFT(addr common.Address, ftName string, balance *big.Int) {
	if nil == balance {
		return
	}
	account := self.getOrNewAccountObject(addr)
	account.SetFT(self.db, balance, ftName)
}

func (self *AccountDB) AddFT(addr common.Address, ftName string, balance *big.Int) bool {
	if nil == balance {
		return true
	}
	account := self.getOrNewAccountObject(addr)

	return account.AddFT(self.db, balance, ftName)
}

func (self *AccountDB) SubFT(addr common.Address, ftName string, balance *big.Int) (*big.Int, bool) {
	if nil == balance {
		return nil, false
	}
	account := self.getOrNewAccountObject(addr)
	return account.SubFT(self.db, balance, ftName)

}

// 根据setId/id 查找NFT
func (self *AccountDB) GetNFTById(setId, id string) *types.NFT {
	addr := common.GenerateNFTAddress(setId, id)
	accountObject := self.getAccountObject(addr, false)

	if nil == accountObject {
		return nil
	}

	return accountObject.GetNFT(self.db)
}

func (self *AccountDB) GetAllNFTByGameId(addr common.Address, appId string) []*types.NFT {
	accountObject := self.getOrNewAccountObject(addr)
	idList := accountObject.getAllNFT(self.db, appId)
	if nil == idList || 0 == len(idList) {
		return make([]*types.NFT, len(idList))
	}

	result := make([]*types.NFT, len(idList))
	for i, id := range idList {
		result[i] = self.GetNFTById(id.SetID, id.ID)
	}
	return result
}

func (self *AccountDB) GetAllNFT(addr common.Address) []*types.NFT {
	return self.GetAllNFTByGameId(addr, "")
}

func (self *AccountDB) ChangeNFTOwner(owner, newOwner common.Address, setId, id string) bool {
	ownerObject := self.getOrNewAccountObject(owner)
	appId := ownerObject.GetNFTAppId(self.db, setId, id)
	if !ownerObject.RemoveNFTLink(self.db, setId, id) {
		return false
	}

	newOwnerObject := self.getOrNewAccountObject(newOwner)
	if !newOwnerObject.AddNFTLink(self.db, appId, setId, id) {
		return false
	}

	nft := self.getAccountObject(common.GenerateNFTAddress(setId, id), false)
	if nil == nft {
		return false
	}
	nft.SetOwner(self.db, newOwner.GetHexString())
	return true
}

func (self *AccountDB) AddNFTByGameId(addr common.Address, appId string, nft *types.NFT) bool {
	if nil == nft {
		return false
	}

	// save nft
	nftObject := self.getOrNewAccountObject(common.GenerateNFTAddress(nft.SetID, nft.ID))
	nftObject.AddNFT(self.db, nft)

	// link to user
	stateObject := self.getOrNewAccountObject(addr)
	return stateObject.AddNFTLink(self.db, appId, nft.SetID, nft.ID)
}

func (self *AccountDB) SetNFTValueByGameId(appId, setId, id, value string) bool {
	nftAddress := common.GenerateNFTAddress(setId, id)
	stateObject := self.getAccountObject(nftAddress, false)
	if nil == stateObject {
		return false
	}

	return stateObject.SetNFTValueByGameId(self.db, appId, value)
}

func (self *AccountDB) SetNFTValueByProperty(addr common.Address, appId, property, setId, id, value string) bool {
	nftAddress := common.GenerateNFTAddress(setId, id)
	stateObject := self.getAccountObject(nftAddress, false)
	if nil == stateObject {
		return false
	}

	return stateObject.SetNFTProperty(self.db, appId, property, value)
}

func (self *AccountDB) RemoveNFTByGameId(addr common.Address, appId, setId, id string) bool {
	nftAddress := common.GenerateNFTAddress(setId, id)
	nftObject := self.getAccountObject(nftAddress, false)
	if nil == nftObject {
		return false
	}

	stateObject := self.getOrNewAccountObject(addr)
	if !stateObject.RemoveNFTLink(self.db, setId, id) {
		return false
	}

	nftObject.markSuicided()
	return true
}

//func (self *AccountDB) RemoveNFT(addr common.Address, nft *types.NFT) bool {
//	stateObject := self.getOrNewAccountObject(addr)
//	return stateObject.RemoveNFT(self.db, nft.AppId, nft.SetID, nft.ID)
//}

func (self *AccountDB) ApproveNFT(owner common.Address, appId, setId, id, renter string) bool {
	nftAddress := common.GenerateNFTAddress(setId, id)
	stateObject := self.getAccountObject(nftAddress, false)
	if nil == stateObject {
		return false
	}
	return stateObject.ApproveNFT(self.db, owner, renter)
}

func (self *AccountDB) SetNFTAppId(owner common.Address, setId, id, appId string) bool {
	// change nft
	nftAddress := common.GenerateNFTAddress(setId, id)
	nftObject := self.getAccountObject(nftAddress, false)
	if nil == nftObject {
		return false
	}
	nftObject.SetAppId(self.db, appId)

	// change link
	stateObject := self.getAccountObject(owner, false)
	if nil == stateObject {
		return false
	}
	stateObject.UpdateNFTLink(self.db, setId, id, appId)
	return true
}

func (self *AccountDB) ChangeNFTStatus(owner common.Address, appId, setId, id string, status byte) bool {
	nftAddress := common.GenerateNFTAddress(setId, id)
	stateObject := self.getAccountObject(nftAddress, false)
	if nil == stateObject {
		return false
	}
	return stateObject.ChangeNFTStatus(self.db, owner, status)
}

func (adb *AccountDB) GetNFTSet(setId string) *types.NFTSet {
	accountObject := adb.getAccountObject(common.GenerateNFTSetAddress(setId), false)
	if nil == accountObject {
		return nil
	}

	return accountObject.GetNFTSet(adb.db)
}
