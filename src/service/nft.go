package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"x/src/common"
	"x/src/middleware/types"
	"x/src/network"
	"x/src/storage/account"
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
	self.lock.RLock()
	defer self.lock.RUnlock()

	return self.getNFTSet(setId, accountDB)
}

func (self *NFTManager) getNFTSet(setId string, accountDB *account.AccountDB) *types.NFTSet {
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
	nftSet := self.getNFTSet(setId, accountDB)
	nftSet.RemoveOwner(id)
	self.updateNFTSet(nftSet, accountDB)

	return nft
}

func (self *NFTManager) GenerateNFTSet(setId, name, symbol, creator, owner string, maxSupply int, createTime string) *types.NFTSet {
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
	if nftSet.MaxSupply < 0 || 0 == len(nftSet.SetID) || self.contains(nftSet.SetID, accountDB) {
		return fmt.Sprintf("setId or maxSupply wrong, setId: %s, maxSupply: %d", nftSet.SetID, nftSet.MaxSupply), false
	}

	self.updateNFTSet(nftSet, accountDB)
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
	nftSet := self.getNFTSet(setId, accountDB)
	if nil == nftSet || nftSet.Owner != appId {
		txLogger.Debugf("Mint nft! wrong setId or not setOwner! appId%s,setId:%s,id:%s,data:%s,createTime:%s,owner:%s", appId, setId, id, data, createTime, owner.String())
		return "wrong setId or not setOwner", false
	}

	if nftSet.MaxSupply != 0 && len(nftSet.OccupiedID) == nftSet.MaxSupply {
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
		if nil == nftSet.OccupiedID {
			nftSet.OccupiedID = make(map[string]common.Address, 0)
		}

		nftSet.OccupiedID[id] = owner
		nftSet.TotalSupply++
		self.updateNFTSet(nftSet, accountDB)
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
	nftSet := self.getNFTSet(setId, accountDB)
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
		nftSet := self.GetNFTSet(setId, accountDB)
		nftSet.ChangeOwner(id, newOwner)
		self.updateNFTSet(nftSet, accountDB)

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
	data["maxSupply"] = strconv.Itoa(nftSet.MaxSupply)
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
