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
	"fmt"
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

func (self *AccountDB) GetAllBNT(addr common.Address) map[string]*big.Int {
	accountObject := self.getOrNewAccountObject(addr)
	return accountObject.getAllFT(self.db, true)
}

func (self *AccountDB) GetAllFT(addr common.Address) map[string]*big.Int {
	accountObject := self.getOrNewAccountObject(addr)
	return accountObject.getAllFT(self.db, false)
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
		remain.Sub(remain, utility.FormatDecimalForERC20(balance, int64(decimal)))
		account.setData(key, remain.Bytes())
		return remain, true
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
		return false
	}

	newOwnerObject := self.getOrNewAccountObject(newOwner)
	if !newOwnerObject.AddNFTLink(self.db, appId, setId, id) {
		return false
	}

	nft := self.getAccountObject(common.GenerateNFTAddress(setId, id), false)
	if nil == nft || 0 != nft.getNFTStatus(self.db) {
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

// source 用户 锁定 resource 到target
func (adb *AccountDB) LockResource(sourceAddr, targetAddr common.Address, resource types.LockResource) bool {
	db := adb.db
	source := adb.getOrNewAccountObject(sourceAddr)
	target := adb.getOrNewAccountObject(targetAddr)
	if target.IsNFT() {
		return false
	}

	// balance
	balanceString := resource.Balance
	if 0 != len(balanceString) {
		balance, err := utility.StrToBigInt(balanceString)
		if nil != err || source.Balance().Cmp(balance) < 0 {
			return false
		}

		source.SubBalance(balance)
		target.lockBalance(db, sourceAddr, balance)
	}

	// bnt
	bntMap := resource.Coin
	if 0 != len(bntMap) {
		for bnt, value := range bntMap {
			amount, err := utility.StrToBigInt(value)
			if err != nil {
				return false
			}
			if 0 == amount.Sign() {
				continue
			}

			_, ok := source.SubBNT(db, amount, bnt)
			if !ok {
				return false
			}
			target.lockBNT(db, sourceAddr, bnt, amount)
		}
	}

	// ft
	ftMap := resource.FT
	if 0 != len(ftMap) {
		for ft, value := range ftMap {
			amount, err := utility.StrToBigInt(value)
			if err != nil {
				return false
			}
			if 0 == amount.Sign() {
				continue
			}
			_, ok := source.SubFT(db, amount, ft)
			if !ok {
				return false
			}
			target.lockFT(db, sourceAddr, ft, amount)
		}
	}

	// nft
	nftList := resource.NFT
	if 0 != len(nftList) {
		for _, nft := range nftList {
			setId := nft.SetId
			id := nft.Id

			nft := adb.getAccountObject(common.GenerateNFTAddress(setId, id), false)
			if nil == nft || !nft.lockNFTSelf(db, sourceAddr, targetAddr) {
				return false
			}

			target.lockNFT(db, sourceAddr, setId, id)
		}
	}
	return true
}

// source 用户 解锁 存放在target(通常是nftSet)中 resource
func (adb *AccountDB) UnLockResource(sourceAddr, targetAddr common.Address, demand types.LockResource) (bool, string) {
	flag, msg := adb.processResource(sourceAddr, targetAddr, "", "", demand, 1)
	if !flag {
		accountLog.Errorf(msg)
	} else {
		accountLog.Debugf(msg)
	}
	return flag, msg
}

// target(nftSet) mintNFT时，销毁
// source 用户 锁定在target中 resource
func (adb *AccountDB) DestroyResource(sourceAddr, targetAddr common.Address, demand types.LockResource) (bool, string) {
	flag, msg := adb.processResource(sourceAddr, targetAddr, "", "", demand, 2)
	if !flag {
		accountLog.Errorf(msg)
	} else {
		accountLog.Debugf(msg)
	}
	return flag, msg
}

// target(nftSet账户) comboNFT时，转移
// source(资源提供方) 用户 锁定在target中的 resource到ComboResource
func (adb *AccountDB) ComboResource(sourceAddr, targetAddr common.Address, setId, id string, demand types.LockResource) (bool, string) {
	flag, msg := adb.processResource(sourceAddr, targetAddr, setId, id, demand, 3)
	if !flag {
		accountLog.Errorf(msg)
	} else {
		accountLog.Debugf(msg)
	}
	return flag, msg
}

// 处理target中锁定的资源
// 处理类型有：
// 1 解锁（返回给sourceAddr）
// 2 mintNFT时，销毁对应的资源
// 3 组合NFT时转移至nft的ComboResource中
func (adb *AccountDB) processResource(sourceAddr, targetAddr common.Address, setId, id string, demand types.LockResource, kind byte) (bool, string) {
	var targetNFT *accountObject
	if 3 == kind {
		targetNFT = adb.getAccountObject(common.GenerateNFTAddress(setId, id), false)
		if nil == targetNFT {
			return false, fmt.Sprintf("setId: %s id: %s not existed", setId, id)
		}
		if !targetNFT.checkOwner(adb.db, sourceAddr) {
			return false, fmt.Sprintf("setId: %s id: %s owner wrong: %s", setId, id, sourceAddr.GetHexString())
		}
		status := targetNFT.getNFTStatus(adb.db)
		if 0 != status {
			return false, fmt.Sprintf("setId: %s id: %s status error: %d", setId, id, status)
		}
	}

	source := adb.getOrNewAccountObject(sourceAddr)
	target := adb.getOrNewAccountObject(targetAddr)
	db := adb.db

	// balance
	balanceString := demand.Balance
	if 0 != len(balanceString) {
		balance, err := utility.StrToBigInt(balanceString)
		if nil != err || !target.unlockBalance(db, sourceAddr, balance) {
			return false, fmt.Sprintf("setId: %s id: %s balance error: %s", setId, id, balanceString)
		}

		switch kind {
		case 1:
			source.AddBalance(balance)
			break
		case 2:
			// do nothing
			break
		case 3:
			targetNFT.setComboBalance(db, balance)
			break
		}

	}

	// bnt
	bntMap := demand.Coin
	if 0 != len(bntMap) {
		for bnt, value := range bntMap {
			amount, err := utility.StrToBigInt(value)
			if err != nil || !target.unlockBNT(db, sourceAddr, bnt, amount) {
				return false, fmt.Sprintf("setId: %s id: %s bnt error: %s, %s", setId, id, bnt, value)
			}

			switch kind {
			case 1:
				source.AddBNT(db, amount, bnt)
				break
			case 2:
				// do nothing
				break
			case 3:
				targetNFT.setComboBNT(db, amount, bnt)
				break
			}

		}
	}

	// ft
	ftMap := demand.FT
	if 0 != len(ftMap) {
		for ft, value := range ftMap {
			amount, err := utility.StrToBigInt(value)
			if err != nil || !target.unlockFT(db, sourceAddr, ft, amount) {
				return false, fmt.Sprintf("setId: %s id: %s ft error: %s, %s", setId, id, ft, value)
			}

			switch kind {
			case 1:
				source.AddFT(db, amount, ft)
				break
			case 2:
				// do nothing
				break
			case 3:
				targetNFT.setComboFT(db, amount, ft)
				break
			}

		}
	}

	// nft
	nftList := demand.NFT
	if 0 != len(nftList) {
		for _, nft := range nftList {
			setId := nft.SetId
			id := nft.Id

			nft := adb.getAccountObject(common.GenerateNFTAddress(setId, id), false)
			if nil == nft {
				return false, fmt.Sprintf("setId: %s id: %s not existed", setId, id)
			}

			nft.unlockNFTSelf(db)
			target.unlockNFT(db, sourceAddr, setId, id)

			switch kind {
			case 1:
				// do nothing
				break
			case 2:
				// destory nft
				adb.RemoveNFTByGameId(sourceAddr, setId, id)
				break
			case 3:
				targetNFT.setComboNFT(db, setId, id)
				break
			}
		}
	}

	return true, ""
}

// 查询target中所有的锁定的资源情况
func (adb *AccountDB) GetLockedResource(targetAddr common.Address) map[string]*types.LockResource {
	target := adb.getAccountObject(targetAddr, false)
	if nil == target {
		return nil
	}

	return target.getAllLockedResource(adb.db)
}

// 查询target中 filter 地址 锁定的资源情况
func (adb *AccountDB) GetLockedResourceByAddress(targetAddr, filter common.Address) *types.LockResource {
	target := adb.getAccountObject(targetAddr, false)
	if nil == target {
		return nil
	}

	return target.getLockedResource(adb.db, filter)
}
