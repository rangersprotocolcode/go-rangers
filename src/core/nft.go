package core

import (
	"x/src/common"
	"x/src/middleware/types"
	"sync"
	"x/src/storage/account"
	"encoding/json"
	"fmt"
	"time"
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
func (self *NFTManager) contains(setId string, accountDB *account.AccountDB) bool {
	valueByte := accountDB.GetData(common.NFTSetAddress, []byte(setId))
	if nil == valueByte || 0 == len(valueByte) {
		return false
	}

	return true
}

// 刷新nftset数据
func (self *NFTManager) updateNFTSet(nftSet *types.NFTSet, accountDB *account.AccountDB) {
	data, _ := json.Marshal(nftSet)
	accountDB.SetData(common.NFTSetAddress, []byte(nftSet.SetID), data)
}

// 获取NFTSet信息
// 状态机&客户端(钱包)调用
func (self *NFTManager) GetNFTSet(setId string, accountDB *account.AccountDB) *types.NFTSet {
	valueByte := accountDB.GetData(common.NFTSetAddress, []byte(setId))
	if nil == valueByte || 0 == len(valueByte) {
		return nil
	}

	var nftSet types.NFTSet
	err := json.Unmarshal(valueByte, &nftSet)
	if err != nil {
		logger.Error("fail to get nftSet: %s, %s", setId, err.Error())
		return nil
	}
	return &nftSet
}

// L2发行NFTSet
// 状态机调用
func (self *NFTManager) PublishNFTSet(setId string, name string, symbol string, totalSupply uint, accountDB *account.AccountDB) (string, bool, *types.NFTSet) {
	self.lock.Lock()
	defer self.lock.Unlock()

	// 检查setId是否存在
	if 0 == len(setId) || self.contains(setId, accountDB) {
		return "setId wrong", false, nil
	}

	// 创建NFTSet
	nftSet := &types.NFTSet{
		SetID:       setId,
		Name:        name,
		Symbol:      symbol,
		TotalSupply: totalSupply,
	}
	nftSet.OccupiedID = make(map[string]common.Address, 0)

	self.updateNFTSet(nftSet, accountDB)
	return "nft publish successful", true, nftSet
}

// L2创建NFT
// 状态机调用
func (self *NFTManager) MintNFT(appId, setId, id, data string, owner common.Address, accountDB *account.AccountDB) (string, bool) {
	if 0 == len(setId) || 0 == len(id) {
		return "setId and id cannot be null", false
	}

	// 检查setId是否存在
	nftSet := self.GetNFTSet(setId, accountDB)
	if nil == nftSet {
		return "wrong setId", false
	}

	return self.GenerateNFT(nftSet, appId, setId, id, data, appId, time.Now(), owner, accountDB)

}

func (self *NFTManager) GenerateNFT(nftSet *types.NFTSet, appId, setId, id, data, creator string, timeStamp time.Time, owner common.Address, accountDB *account.AccountDB) (string, bool) {
	// 检查id是否存在
	if _, ok := nftSet.OccupiedID[id]; ok {
		return "wrong id", false
	}

	// 创建NFT对象
	nft := &types.NFT{
		SetID:      setId,
		Name:       nftSet.Name,
		Symbol:     nftSet.Symbol,
		ID:         id,
		Creator:    creator,
		CreateTime: timeStamp,
		Owner:      owner.GetHexString(),
		Status:     0,
		AppId:      appId,
	}
	nft.DataKey = make([]string, 0)
	nft.DataValue = make([]string, 0)
	if 0 != len(data) {
		nft.SetData(data, appId)
	}

	//分配NFT
	if accountDB.AddNFTByGameId(owner, appId, nft) {
		// 修改NFTSet数据
		nftSet.OccupiedID[id] = owner
		self.updateNFTSet(nftSet, accountDB)
		return "nft mint successful", true
	} else {
		return "fail to nft mint", false
	}
}

// 获取NFT信息
// 状态机&客户端(钱包)调用
func (self *NFTManager) GetNFT(setId string, id string, accountDB *account.AccountDB) *types.NFT {
	// 检查setId是否存在
	nftSet := self.GetNFTSet(setId, accountDB)
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
	if nil == nftSet {
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
func (self *NFTManager) Transfer(appId, setId, id string, owner, newOwner common.Address, accountDB *account.AccountDB) (string, bool) {
	// 根据setId+id 查找nft
	nft := accountDB.GetNFTById(owner, setId, id)
	if nil == nft {
		return fmt.Sprintf("nft is not existed. setId: %s, id: %s, owner: %s", setId, id, owner.String()), false
	}

	// 判断nft是否可以被transfer
	if nft.Status != 0 {
		return fmt.Sprintf("nft cannot be transferred. setId: %s, id: %s", setId, id), false
	}

	// 修改数据
	nft.Owner = newOwner.GetHexString()
	nftSet := self.GetNFTSet(setId, accountDB)
	nftSet.ChangeOwner(id, newOwner)
	self.updateNFTSet(nftSet, accountDB)

	// 通知本状态机
	return "nft transfer successful", true
}

// NFT 穿梭
// 状态机&玩家(钱包)调用
func (self *NFTManager) Shuttle(setId, id, newAppId string, accountDB *account.AccountDB) (string, bool) {
	return self.shuttle(setId, id, newAppId, accountDB, false)
}

// NFT 穿梭
// 玩家（钱包）调用
// appId若为空，则穿梭到默认appId（库存）
func (self *NFTManager) ForceShuttle(setId, id, newAppId string, accountDB *account.AccountDB) (string, bool) {
	// 根据setId+id 查找nft
	// 修改数据
	// 通知当前状态机
	// 通知目标状态机（如果appId不为空）
	return self.shuttle(setId, id, newAppId, accountDB, true)
}

func (self *NFTManager) shuttle(setId, id, newAppId string, accountDB *account.AccountDB, isForce bool) (string, bool) {
	// 根据setId+id 查找nft
	nft := self.GetNFT(setId, id, accountDB)
	if nil == nft {
		return fmt.Sprintf("nft is not existed. setId: %s, id: %s", setId, id), false
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
