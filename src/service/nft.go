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
	"bytes"
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

var NFTManagerInstance *NFTManager

func initNFTManager() {
	NFTManagerInstance = &NFTManager{}
	NFTManagerInstance.lock = sync.RWMutex{}
}

type NFTManager struct {
	lock sync.RWMutex
}

// 检查setId是否存在
func (self *NFTManager) Contains(setId string, accountDB *account.AccountDB) bool {
	return accountDB.Exist(common.GenerateNFTSetAddress(setId))
}

// 从nftSet中删除某个nft
func (self *NFTManager) deleteNFTFromNFTSet(setId, id string, db *account.AccountDB) {
	nftSetAddress := common.GenerateNFTSetAddress(setId)
	db.RemoveData(nftSetAddress, utility.StrToBytes(id))
}

// 刷新nftset数据
func (self *NFTManager) updateOwnerFromNFTSet(setId, id string, owner common.Address, accountDB *account.AccountDB) {
	nftSetAddress := common.GenerateNFTSetAddress(setId)
	accountDB.SetData(nftSetAddress, utility.StrToBytes(id), owner.Bytes())
}

// 修改nftset数据
func (self *NFTManager) insertNewNFTFromNFTSet(setId, id string, owner common.Address, accountDB *account.AccountDB) {
	nftSetAddress := common.GenerateNFTSetAddress(setId)
	accountDB.SetData(nftSetAddress, utility.StrToBytes(id), owner.Bytes())
	accountDB.IncreaseNonce(nftSetAddress)
}

func (self *NFTManager) insertNewNFTSet(nftSet *types.NFTSet, db *account.AccountDB) {
	if nil == nftSet {
		return
	}

	nftSetAddress := common.GenerateNFTSetAddress(nftSet.SetID)
	db.SetNFTSetDefinition(nftSetAddress, nftSet.ToBlob(), nftSet.Owner)
}

// 获取NFTSet信息
// 状态机&客户端(钱包)调用
func (self *NFTManager) GetNFTSet(setId string, accountDB *account.AccountDB) *types.NFTSet {
	self.lock.RLock()
	defer self.lock.RUnlock()

	return accountDB.GetNFTSet(setId)
}

func (self *NFTManager) GenerateNFTSet(setId, name, symbol, creator, owner string, conditions types.NFTConditions, maxSupply uint64, createTime string) *types.NFTSet {
	// 创建NFTSet
	nftSet := &types.NFTSet{
		SetID:      setId,
		Name:       name,
		Symbol:     symbol,
		Creator:    creator,
		Owner:      owner,
		MaxSupply:  maxSupply,
		CreateTime: createTime,
		Conditions: conditions,
	}

	return nftSet
}

// L2发行NFTSet
// 状态机调用
func (self *NFTManager) PublishNFTSet(nftSet *types.NFTSet, accountDB *account.AccountDB) (string, bool) {
	self.lock.Lock()
	defer self.lock.Unlock()

	if nil == nftSet {
		return "nil nftSet", false
	}

	// 检查setId是否存在
	if nftSet.MaxSupply < 0 || 0 == len(nftSet.SetID) {
		return fmt.Sprintf("setId or maxSupply wrong, setId: %s, maxSupply: %d", nftSet.SetID, nftSet.MaxSupply), false
	}

	if strings.Contains(nftSet.SetID, ":") || self.Contains(nftSet.SetID, accountDB) {
		return fmt.Sprintf("setId: %s, existed", nftSet.SetID), false
	}

	self.insertNewNFTSet(nftSet, accountDB)
	return fmt.Sprintf("nft publish successful, setId: %s", nftSet.SetID), true
}

// L2创建NFT
// 状态机调用
func (self *NFTManager) MintNFT(nftSetOwner, appId, setId, id, data, createTime string, owner common.Address, accountDB *account.AccountDB) (string, bool) {
	txLogger.Debugf("Mint NFT! appId: %s, setId: %s, id: %s, data: %s, createTime: %s, owner: %s", appId, setId, id, data, createTime, owner.String())
	self.lock.Lock()
	defer self.lock.Unlock()

	if 0 == len(setId) || 0 == len(id) || strings.Contains(id, ":") {
		txLogger.Tracef("Mint nft! setId and id cannot be null")
		return "setId and id cannot be null", false
	}

	// 检查setId是否存在
	nftSet := accountDB.GetNFTSet(setId)
	if nil == nftSet || 0 != strings.Compare(common.FormatHexString(nftSet.Owner), common.FormatHexString(nftSetOwner)) {
		txLogger.Errorf("Mint nft! wrong setId or not setOwner! appId%s,setId:%s,id:%s,data:%s,createTime:%s,owner:%s", appId, setId, id, data, createTime, owner.String())
		return "wrong setId or not setOwner", false
	}

	if nftSet.MaxSupply != 0 && uint64(len(nftSet.OccupiedID)) == nftSet.MaxSupply {
		txLogger.Errorf("not enough nftSet: " + setId)
		return fmt.Sprintf("not enough nftSet %s, MaxSupply: %d, Occupied: %d", setId, nftSet.MaxSupply, len(nftSet.OccupiedID)), false
	}

	if 0 == len(appId) {
		appId = nftSetOwner
	}

	// 定义了组合规则的
	if !nftSet.Conditions.IsEmpty() {
		msg, demand := self.checkNFTConditions(nftSet.Conditions, accountDB, setId, owner)
		if nil == demand {
			txLogger.Errorf(msg)
			return msg, false
		}

		flag, message := accountDB.DestroyResource(owner, common.GenerateNFTSetAddress(setId), *demand)
		if !flag {
			txLogger.Errorf(message)
			return message, false
		}
	}
	return self.GenerateNFT(nftSet, appId, setId, id, data, nftSetOwner, createTime, "", owner, nil, accountDB)
}

func (self *NFTManager) checkNFTConditions(nftConditions types.NFTConditions, accountDB *account.AccountDB, nftSetId string, owner common.Address) (string, *types.LockResource) {
	demand := types.LockResource{}
	if 0 != len(nftConditions.Balance) {
		demand.Balance = nftConditions.Balance
	}
	if 0 != len(nftConditions.FT) {
		demand.FT = nftConditions.FT
	}
	if 0 != len(nftConditions.Coin) {
		demand.Coin = nftConditions.Coin
	}
	if 0 != len(nftConditions.NFT) {
		demand.NFT = make([]types.NFTID, 0)
		lockedResource := accountDB.GetLockedResourceByAddress(common.GenerateNFTSetAddress(nftSetId), owner)
		if nil == lockedResource || 0 == len(lockedResource.NFT) {
			return fmt.Sprintf("owner: %s has no lockedResource in NFTSet %s for conditions: %s", owner.GetHexString(), nftSetId, nftConditions.String()), nil
		}

		for setId, nftCondition := range nftConditions.NFT {
			num := nftCondition.Num
			if 0 >= num {
				num = 1
			}

			found := 0
			for i := 0; i < len(lockedResource.NFT); i++ {
				lockedNFT := lockedResource.NFT[i]
				if lockedNFT.SetId != setId {
					continue
				}

				nft := accountDB.GetNFTById(setId, lockedNFT.Id)
				if nil == nft {
					continue
				}

				match := true
				if 0 != len(nftCondition.Attribute) {
					for property, conditionMap := range nftCondition.Attribute {
						var valueString string
						if strings.Contains(property, ":") {
							list := strings.Split(property, ":")
							if 2 != len(list) {
								return fmt.Sprintf("bad conditions: %s for property: %s in NFTSet %s", nftConditions.String(), property, nftSetId), nil
							}
							valueString = nft.GetProperty(list[0], list[1])
						} else {
							valueString = nft.GetProperty(nft.AppId, property)
						}

						value, err := strconv.Atoi(valueString)
						if err != nil {
							continue
						}

						targetValue := strings.TrimSpace(conditionMap["value"])
						switch strings.ToLower(conditionMap["operate"]) {
						case "eq":
							target, err := strconv.Atoi(targetValue)
							if err != nil || value != target {
								match = false
							}
							break
						case "ne":
							target, err := strconv.Atoi(targetValue)
							if err != nil || value == target {
								match = false
							}
						case "gt":
							target, err := strconv.Atoi(targetValue)
							if err != nil || value <= target {
								match = false
							}
							break
						case "lt":
							target, err := strconv.Atoi(targetValue)
							if err != nil || value >= target {
								match = false
							}
							break
						case "ge":
							target, err := strconv.Atoi(targetValue)
							if err != nil || value < target {
								match = false
							}
							break
						case "le":
							target, err := strconv.Atoi(targetValue)
							if err != nil || value > target {
								match = false
							}
							break
						case "between":
							borders := targetValue[1 : len(targetValue)-1]
							borderList := strings.Split(borders, ",")
							if 2 != len(borderList) {
								return fmt.Sprintf("bad conditions: %s for borders: %s in NFTSet %s", nftConditions.String(), borders, nftSetId), nil
							}
							border, err := strconv.Atoi(strings.TrimSpace(borderList[0]))
							if err != nil || value < border {
								match = false
							}
							border, err = strconv.Atoi(strings.TrimSpace(borderList[1]))
							if err != nil || value > border {
								match = false
							}
							break
						default:
							match = false
						}
						if !match {
							break
						}
					}
				}
				if match {
					demand.NFT = append(demand.NFT, types.NFTID{SetId: setId, Id: lockedNFT.Id})
					found++
					lockedResource.NFT = append(lockedResource.NFT[:i], lockedResource.NFT[i+1:]...)
					i--
				}
			}

			if num != found {
				return fmt.Sprintf("owner: %s bad num: %d Vs found: %d for nftSet: %s in conditons: %s", owner.GetHexString(), num, found, nftSetId, nftConditions.String()), nil
			}
		}

	}

	if 0 == len(demand.Balance) && 0 == len(demand.FT) && 0 == len(demand.Coin) && 0 == len(demand.NFT) {
		return fmt.Sprintf("owner: %s has no lockedResource in NFTSet %s for conditions: %s", owner.GetHexString(), nftSetId, nftConditions.String()), nil
	}
	return "", &demand
}

func (self *NFTManager) GenerateNFT(nftSet *types.NFTSet, appId, setId, id, data, creator, timeStamp, imported string, owner common.Address, fullData map[string]string, accountDB *account.AccountDB) (string, bool) {
	txLogger.Tracef("Generate NFT! appId%s,setId:%s,id:%s,data:%s,createTime:%s,owner:%s", appId, setId, id, data, timeStamp, owner.String())
	// 检查id是否存在
	if _, ok := nftSet.OccupiedID[id]; ok {
		msg := fmt.Sprintf("Generate NFT wrong id! appId%s,setId:%s,id:%s,data:%s,createTime:%s,owner:%s", appId, setId, id, data, timeStamp, owner.String())
		txLogger.Debugf(msg)
		return msg, false
	}
	ownerString := owner.GetHexString()

	// 创建NFT对象
	nft := &types.NFT{
		SetID:      setId,
		Name:       nftSet.Name,
		Symbol:     nftSet.Symbol,
		ID:         id,
		Creator:    creator,
		CreateTime: timeStamp,
		Owner:      ownerString,
		Renter:     ownerString,
		Status:     0,
		AppId:      appId,
		Imported:   imported,
		Data:       make(map[string]string),
	}

	if 0 != len(data) {
		nft.SetData(appId, data)
	} else if nil != fullData && 0 != len(fullData) {
		for key, value := range fullData {
			nft.SetData(key, value)
		}
	}

	//分配NFT
	if accountDB.AddNFTByGameId(owner, appId, nft) {
		// 修改NFTSet数据
		self.insertNewNFTFromNFTSet(setId, id, owner, accountDB)
		return fmt.Sprintf("nft mint successful. setId: %s,id: %s", setId, id), true
	} else {
		msg := fmt.Sprintf("nft mint failed. appId: %s,setId: %s,id: %s,data: %s,createTime: %s,owner: %s", appId, setId, id, data, timeStamp, owner.String())
		txLogger.Debugf(msg)
		return msg, false
	}
}

//mark nft been withdrawn
func (self *NFTManager) MarkNFTWithdrawn(owner common.Address, setId, id string, accountDB *account.AccountDB) *types.NFT {
	self.lock.RLock()
	defer self.lock.RUnlock()

	nft := accountDB.GetNFTById(setId, id)
	if nil == nft || nft.Status != 0 {
		return nil
	}

	//change nft status to be withdrawn
	accountDB.ChangeNFTStatus(owner, "", setId, id, 2)
	return nft
}

//deposit local withdrawn nft
//only owner renter appId data will be updated,other fields will not be updated
func (self *NFTManager) DepositWithdrawnNFT(owner, renter, appId string, fullData map[string]string, accountDB *account.AccountDB, originalNFT *types.NFT) (string, bool) {
	self.lock.RLock()
	defer self.lock.RUnlock()

	if nil == originalNFT {
		return fmt.Sprintf("nft is not existed. setId: %s, id: %s", originalNFT.SetID, originalNFT.ID), false
	}

	nft := &types.NFT{SetID: originalNFT.SetID, ID: originalNFT.ID,
		Name: originalNFT.Name, Symbol: originalNFT.Symbol,
		Creator: originalNFT.Creator, CreateTime: originalNFT.CreateTime,
		Condition: originalNFT.Condition, Imported: originalNFT.Imported}
	nft.Status = 0
	nft.Owner = owner
	nft.Renter = renter
	nft.AppId = appId
	nft.Data = make(map[string]string)

	if nil != fullData && 0 != len(fullData) {
		for key, value := range fullData {
			nft.SetData(key, value)
		}
	}

	if accountDB.RemoveNFTByGameId(common.HexStringToAddress(originalNFT.Owner), originalNFT.SetID, originalNFT.ID) && accountDB.AddNFTByGameId(common.HexStringToAddress(nft.Owner), nft.AppId, nft) {
		if nft.Owner != originalNFT.Owner {
			self.updateOwnerFromNFTSet(originalNFT.SetID, originalNFT.ID, common.HexStringToAddress(nft.Owner), accountDB)
		}
		return "Deposit withdrawn nft successful", true
	}
	return "Deposit withdrawn nft false", true
}

// 获取NFT信息
// 状态机&客户端(钱包)调用
func (self *NFTManager) GetNFT(setId string, id string, accountDB *account.AccountDB) *types.NFT {
	return accountDB.GetNFTById(setId, id)
}

// 批量获取NFT信息
// 状态机&玩家(钱包)调用
func (self *NFTManager) GetNFTs(setId string, idList []string, accountDB *account.AccountDB) []*types.NFT {
	if 0 == len(setId) || 0 == len(idList) {
		return nil
	}

	result := make([]*types.NFT, len(idList))
	for i, id := range idList {
		result[i] = self.GetNFT(setId, id, accountDB)
	}
	return result
}

// 获取用户地址下，某个游戏的所有NFT信息
// 状态机&玩家(钱包)调用
func (self *NFTManager) GetNFTListByAddress(address common.Address, appId string, accountDB *account.AccountDB) []*types.NFT {
	if len(appId) == 0 {
		return accountDB.GetAllNFT(address)
	}

	return accountDB.GetAllNFTByGameId(address, appId)
}

func (self *NFTManager) GetNFTOwner(setId, id string, accountDB *account.AccountDB) *common.Address {
	// 检查setId是否存在
	nftSet := self.GetNFTSet(setId, accountDB)
	if nil == nftSet || nil == nftSet.OccupiedID {
		return nil
	}

	address, ok := nftSet.OccupiedID[id]
	if ok {
		return &address
	} else {
		return nil
	}
}

// 更新用户当前游戏的NFT数据属性
// 状态机调用
func (self *NFTManager) UpdateNFT(appId, setId, id, data, propertyString string, accountDB *account.AccountDB) bool {
	result := accountDB.SetNFTValueByGameId(appId, setId, id, data)
	if !result {
		return false
	}

	property := make(map[string]string)
	if 0 != len(propertyString) {
		if err := json.Unmarshal(utility.StrToBytes(propertyString), &property); nil != err {
			return false
		}
	}
	if 0 != len(property) {
		for key, value := range property {
			if !self.updateNFTProperty(appId, setId, id, key, value, accountDB) {
				return false
			}
		}
	}

	return true
}

func (self *NFTManager) updateNFTProperty(appId, setId, id, property, data string, accountDB *account.AccountDB) bool {
	return accountDB.SetNFTValueByProperty(appId, setId, id, property, data)
}

// NFT 迁移
// 状态机&玩家(钱包)调用
func (self *NFTManager) Transfer(setId, id string, owner, newOwner common.Address, accountDB *account.AccountDB) (string, bool) {
	txLogger.Tracef("Transfer nft.setId:%s,id:%s,owner:%s,newOwner:%s", setId, id, owner.String(), newOwner.String())
	// 根据setId+id 查找nft
	nft := accountDB.GetNFTById(setId, id)
	if nil == nft {
		return fmt.Sprintf("nft is not existed. setId: %s, id: %s, owner: %s", setId, id, owner.String()), false
	}
	if 0 != bytes.Compare(owner.Bytes(), common.HexStringToAddress(nft.Owner).Bytes()) {
		return fmt.Sprintf("nft is not authed. setId: %s, id: %s, owner: %s", setId, id, owner.String()), false
	}
	txLogger.Tracef("Transfer nft.Got nft:%v", nft)

	// 判断nft是否可以被transfer
	if nft.Status != 0 {
		return fmt.Sprintf("nft cannot be transferred. setId: %s, id: %s", setId, id), false
	}

	// 修改数据
	if accountDB.ChangeNFTOwner(owner, newOwner, setId, id) {
		self.updateOwnerFromNFTSet(setId, id, newOwner, accountDB)

		// 通知本状态机
		return "nft transfer successful", true
	}

	// 通知本状态机
	return "nft transfer fail", false

}

// NFT 穿梭
// 状态机&玩家(钱包)调用
func (self *NFTManager) Shuttle(owner, setId, id, newAppId string, accountDB *account.AccountDB) (string, bool) {
	return self.shuttle(owner, setId, id, newAppId, accountDB, false)
}

// NFT 穿梭
// 玩家（钱包）调用
// appId若为空，则穿梭到默认appId（库存）
func (self *NFTManager) ForceShuttle(owner, setId, id, newAppId string, accountDB *account.AccountDB) (string, bool) {
	// 根据setId+id 查找nft
	// 修改数据
	// 通知当前状态机
	// 通知目标状态机（如果appId不为空）
	return self.shuttle(owner, setId, id, newAppId, accountDB, true)
}

func (self *NFTManager) shuttle(owner, setId, id, newAppId string, accountDB *account.AccountDB, isForce bool) (string, bool) {
	// 根据setId+id 查找nft
	nft := self.GetNFT(setId, id, accountDB)
	if nil == nft {
		return fmt.Sprintf("nft is not existed. setId: %s, id: %s", setId, id), false
	}

	// owner 判断
	if nft.Owner != owner {
		return fmt.Sprintf("nft cannot be shuttled by owner. setId: %s, id: %s, owner: %s", setId, id, owner), false
	}
	// 判断nft是否可以被shuttle
	if !isForce && (nft.Status != 0 || nft.AppId == newAppId) {
		return fmt.Sprintf("nft cannot be shuttled. setId: %s, id: %s", setId, id), false
	}

	// 修改数据
	accountDB.SetNFTAppId(common.HexStringToAddress(owner), setId, id, newAppId)

	// 通知当前状态机
	// 通知接收状态机
	return "nft shuttle successful", true
}
