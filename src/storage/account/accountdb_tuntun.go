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
	"com.tuntun.rocket/node/src/utility"
	"math/big"
)

func (self *AccountDB) GetBNT(addr common.Address, bntName string) *big.Int {
	accountObject := self.getOrNewAccountObject(addr)
	raw := accountObject.getFT(self.db, common.GenerateBNTName(bntName))
	if raw == nil {
		return big.NewInt(0)
	}
	return raw
}

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

func (self *AccountDB) GetAllBNT(addr common.Address) map[string]*big.Int {
	accountObject := self.getOrNewAccountObject(addr)
	return accountObject.getAllFT(self.db, true)
}

func (self *AccountDB) GetAllFT(addr common.Address) map[string]*big.Int {
	result := make(map[string]*big.Int)
	return result
}

func (self *AccountDB) SetBNT(addr common.Address, bntName string, balance *big.Int) {
	if nil == balance {
		return
	}
	account := self.getOrNewAccountObject(addr)
	account.SetFT(self.db, balance, common.GenerateBNTName(bntName))
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

func (self *AccountDB) AddBNT(addr common.Address, bntName string, balance *big.Int) bool {
	if nil == balance {
		return true
	}
	account := self.getOrNewAccountObject(addr)

	return account.AddFT(self.db, balance, common.GenerateBNTName(bntName))
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
		account.setData(key, remain.Bytes())

		return true
	}

	account := self.getOrNewAccountObject(addr)
	return account.AddFT(self.db, balance, ftName)
}

func (self *AccountDB) SubBNT(addr common.Address, bntName string, balance *big.Int) (*big.Int, bool) {
	if nil == balance {
		return nil, false
	}
	account := self.getOrNewAccountObject(addr)
	return account.SubBNT(self.db, balance, bntName)
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
		account.setData(key, remain.Bytes())
		return utility.FormatDecimalForRocket(remain, int64(decimal)), true
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

	// 这里出错，意味着这个nft不属于owner
	if !ownerObject.RemoveNFTLink(self.db, setId, id) {
		accountLog.Errorf("error: not found nft(setId: %s, id: %s) for owner: %s", setId, id, owner.String())
		return false
	}

	newOwnerObject := self.getOrNewAccountObject(newOwner)
	if !newOwnerObject.AddNFTLink(self.db, appId, setId, id) {
		accountLog.Errorf("error: already have nft(setId: %s, id: %s) for owner: %s", setId, id, newOwner.String())
		return false
	}

	nft := self.getAccountObject(common.GenerateNFTAddress(setId, id), false)
	if nil == nft {
		accountLog.Errorf("error: nil nft(setId: %s, id: %s)", setId, id)
		return false
	}
	nftStatus := nft.getNFTStatus(self.db)
	if 0 != nftStatus {
		accountLog.Errorf("error: wrong nft(setId: %s, id: %s), status: %d", setId, id, nftStatus)
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

func (self *AccountDB) SetNFTValueByProperty(appId, setId, id, property, value string) bool {
	nftAddress := common.GenerateNFTAddress(setId, id)
	stateObject := self.getAccountObject(nftAddress, false)
	if nil == stateObject {
		return false
	}

	return stateObject.SetNFTProperty(self.db, appId, property, value)
}

func (self *AccountDB) RemoveNFTByGameId(addr common.Address, setId, id string) bool {
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

func (self *AccountDB) ApproveNFT(owner common.Address, appId, setId, id, renter string) bool {
	nftAddress := common.GenerateNFTAddress(setId, id)
	stateObject := self.getAccountObject(nftAddress, false)
	if nil == stateObject {
		accountLog.Errorf("fail to find nft: %s %s, approve failed", setId, id)
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

func (adb *AccountDB) CheckNFTSetOwner(setId string, owner string) bool {
	accountObject := adb.getAccountObject(common.GenerateNFTSetAddress(setId), false)
	if nil == accountObject {
		return false
	}

	return accountObject.CheckNFTSetOwner(adb.db, owner)
}

func (adb *AccountDB) GetNFTSet(setId string) *types.NFTSet {
	accountObject := adb.getAccountObject(common.GenerateNFTSetAddress(setId), false)
	if nil == accountObject {
		return nil
	}

	return accountObject.GetNFTSet(adb.db)
}

func (adb *AccountDB) GetNFTSetDefinition(setId string) *types.NFTSet {
	accountObject := adb.getAccountObject(common.GenerateNFTSetAddress(setId), false)
	if nil == accountObject {
		return nil
	}

	return accountObject.GetNFTSetDefinition(adb.db)
}