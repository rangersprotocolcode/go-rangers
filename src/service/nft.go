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
	"com.tuntun.rocket/node/src/network"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"encoding/json"
	"fmt"
	"strconv"
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

func (self *NFTManager) generateNFTSetAddress(setId string) common.Address {
	address := fmt.Sprintf("n-%s", setId)
	return common.StringToAddress(address)
}

// 检查setId是否存在
func (self *NFTManager) contains(setId string, accountDB *account.AccountDB) bool {
	return accountDB.Exist(self.generateNFTSetAddress(setId))
}

// 从nftSet中删除某个nft
func (self *NFTManager) deleteNFTFromNFTSet(setId, id string, db *account.AccountDB) {
	nftSetAddress := self.generateNFTSetAddress(setId)
	db.RemoveData(nftSetAddress, utility.StrToBytes(id))
}

// 刷新nftset数据
func (self *NFTManager) updateOwnerFromNFTSet(setId, id string, owner common.Address, accountDB *account.AccountDB) {
	nftSetAddress := self.generateNFTSetAddress(setId)
	accountDB.SetData(nftSetAddress, utility.StrToBytes(id), owner.Bytes())
}

// 修改nftset数据
func (self *NFTManager) insertNewNFTFromNFTSet(setId, id string, owner common.Address, accountDB *account.AccountDB) {
	nftSetAddress := self.generateNFTSetAddress(setId)
	accountDB.SetData(nftSetAddress, utility.StrToBytes(id), owner.Bytes())
	accountDB.IncreaseNonce(nftSetAddress)
}

func (self *NFTManager) insertNewNFTSet(nftSet *types.NFTSet, db *account.AccountDB) {
	if nil == nftSet {
		return
	}

	nftSetAddress := self.generateNFTSetAddress(nftSet.SetID)
	db.SetNFTSetDefinition(nftSetAddress, nftSet.ToBlob())
}

// 获取NFTSet信息
// 状态机&客户端(钱包)调用
func (self *NFTManager) GetNFTSet(setId string, accountDB *account.AccountDB) *types.NFTSet {
	self.lock.RLock()
	defer self.lock.RUnlock()

	return accountDB.GetNFTSet(setId)
}

// 从layer2 层面删除
func (self *NFTManager) DeleteNFT(owner common.Address, setId, id string, accountDB *account.AccountDB) *types.NFT {
	self.lock.RLock()
	defer self.lock.RUnlock()

	nft := accountDB.GetNFTById(owner, setId, id)
	if nil == nft {
		return nil
	}

	//删除要提现的NFT
	accountDB.RemoveNFT(owner, nft)

	// 更新nftSet
	self.deleteNFTFromNFTSet(setId, id, accountDB)

	return nft
}

func (self *NFTManager) GenerateNFTSet(setId, name, symbol, creator, owner string, maxSupply uint64, createTime string) *types.NFTSet {
	// 创建NFTSet
	nftSet := &types.NFTSet{
		SetID:      setId,
		Name:       name,
		Symbol:     symbol,
		Creator:    creator,
		Owner:      owner,
		MaxSupply:  maxSupply,
		CreateTime: createTime,
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

	if self.contains(nftSet.SetID, accountDB) {
		return fmt.Sprintf("setId: %s, existed", nftSet.SetID), false
	}

	self.insertNewNFTSet(nftSet, accountDB)
	return fmt.Sprintf("nft publish successful, setId: %s", nftSet.SetID), true
}

// L2创建NFT
// 状态机调用
func (self *NFTManager) MintNFT(appId, setId, id, data, createTime string, owner common.Address, accountDB *account.AccountDB) (string, bool) {
	txLogger.Debugf("Mint NFT! appId: %s, setId: %s, id: %s, data: %s, createTime: %s, owner: %s", appId, setId, id, data, createTime, owner.String())
	self.lock.Lock()
	defer self.lock.Unlock()

	if 0 == len(setId) || 0 == len(id) {
		txLogger.Tracef("Mint nft! setId and id cannot be null")
		return "setId and id cannot be null", false
	}

	// 检查setId是否存在
	nftSet := accountDB.GetNFTSet(setId)
	if nil == nftSet || nftSet.Owner != appId {
		txLogger.Debugf("Mint nft! wrong setId or not setOwner! appId%s,setId:%s,id:%s,data:%s,createTime:%s,owner:%s", appId, setId, id, data, createTime, owner.String())
		return "wrong setId or not setOwner", false
	}

	if nftSet.MaxSupply != 0 && uint64(len(nftSet.OccupiedID)) == nftSet.MaxSupply {
		return "not enough nftSet", false
	}

	return self.GenerateNFT(nftSet, appId, setId, id, data, appId, createTime, "", owner, nil, accountDB)
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
	}
	nft.DataKey = make([]string, 0)
	nft.DataValue = make([]string, 0)
	if 0 != len(data) {
		nft.SetData(data, appId)
	} else if nil != fullData && 0 != len(fullData) {
		for key, value := range fullData {
			nft.DataKey = append(nft.DataKey, key)
			nft.DataValue = append(nft.DataValue, value)
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

// 获取NFT信息
// 状态机&客户端(钱包)调用
func (self *NFTManager) GetNFT(setId string, id string, accountDB *account.AccountDB) *types.NFT {
	// 检查setId是否存在
	nftSet := accountDB.GetNFTSet(setId)
	if nil == nftSet {
		return nil
	}

	address, ok := nftSet.OccupiedID[id]
	if !ok {
		return nil
	}

	return accountDB.GetNFTById(address, setId, id)
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
func (self *NFTManager) UpdateNFT(addr common.Address, appId, setId, id, data string, accountDB *account.AccountDB) bool {
	return accountDB.SetNFTValueByGameId(addr, appId, setId, id, data)
}

// 批量更新用户当前游戏的NFT数据属性
// 状态机调用
func (self *NFTManager) BatchUpdateNFT(addr common.Address, appId, setId string, idList, data []string, accountDB *account.AccountDB) (string, bool) {
	if 0 == len(idList) || 0 == len(data) || len(idList) != len(data) {
		return "wrong idList/data", false
	}
	for i := range idList {
		self.UpdateNFT(addr, appId, setId, idList[i], data[i], accountDB)
	}
	return "batchUpdate successful", true
}

// NFT 迁移
// 状态机&玩家(钱包)调用
func (self *NFTManager) Transfer(setId, id string, owner, newOwner common.Address, accountDB *account.AccountDB) (string, bool) {
	txLogger.Tracef("Transfer nft.setId:%s,id:%s,owner:%s,newOwner:%s", setId, id, owner.String(), newOwner.String())
	// 根据setId+id 查找nft
	nft := accountDB.GetNFTById(owner, setId, id)
	if nil == nft {
		return fmt.Sprintf("nft is not existed. setId: %s, id: %s, owner: %s", setId, id, owner.String()), false
	}
	txLogger.Tracef("Transfer nft.Got nft:%v", nft)

	// 判断nft是否可以被transfer
	if nft.Status != 0 {
		return fmt.Sprintf("nft cannot be transferred. setId: %s, id: %s", setId, id), false
	}

	// 修改数据
	newOwnerString := newOwner.GetHexString()
	nft.Owner = newOwnerString
	nft.Renter = newOwnerString
	if accountDB.AddNFTByGameId(newOwner, nft.AppId, nft) && accountDB.RemoveNFTByGameId(owner, nft.AppId, nft.SetID, nft.ID) {
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
	addr := common.HexToAddress(nft.Owner)
	oldAppId := nft.AppId
	accountDB.RemoveNFTByGameId(addr, oldAppId, setId, id)
	nft.AppId = newAppId
	accountDB.AddNFTByGameId(addr, newAppId, nft)

	// 通知当前状态机
	// 通知接收状态机
	return "nft shuttle successful", true
}

func (self *NFTManager) SendPublishNFTSetToConnector(nftSet *types.NFTSet) {
	data := make(map[string]string, 8)
	data["setId"] = nftSet.SetID
	data["name"] = nftSet.Name
	data["symbol"] = nftSet.Symbol
	data["maxSupply"] = strconv.FormatUint(nftSet.MaxSupply, 10)
	data["creator"] = nftSet.Creator
	data["owner"] = nftSet.Owner
	data["createTime"] = nftSet.CreateTime
	data["contract"] = "" // 标记为源生layer2的数据

	self.publishNFTSetToConnector(data, nftSet.Creator, nftSet.CreateTime)
}

func (self *NFTManager) ImportNFTSet(setId, contract, chainType string) {
	data := make(map[string]string)
	data["setId"] = setId
	data["maxSupply"] = "0"
	data["contract"] = contract // 标记为外部导入的数据
	data["chainType"] = chainType

	self.publishNFTSetToConnector(data, "", "")
}

func (self *NFTManager) publishNFTSetToConnector(data map[string]string, source, time string) {
	b, err := json.Marshal(data)
	if err != nil {
		txLogger.Error("json marshal err, err:%s", err.Error())
		return
	}

	t := types.Transaction{Source: source, Target: "", Data: string(b), Type: types.TransactionTypePublishNFTSet, Time: time}
	t.Hash = t.GenHash()

	msg, err := json.Marshal(t.ToTxJson())
	if err != nil {
		txLogger.Debugf("Json marshal tx json error:%s", err.Error())
		return
	}

	txLogger.Tracef("After publish nft.Send msg to coiner:%s", t.ToTxJson().ToString())
	go network.GetNetInstance().SendToCoinConnector(msg)
}
