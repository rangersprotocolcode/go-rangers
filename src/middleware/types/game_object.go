package types

import (
	"time"
	"math/big"
)

type NFTSet struct {
	ID          string
	Name        string
	Symbol      string
	TotalSupply uint
	Occupied    uint
}

type NFT struct {
	//
	SetID  string
	Name   string
	Symbol string

	// 1. 通用数据
	ID         string    // NFT自身ID，创建时指定。创建后不可修改
	Creator    string    // 初次创建者，一般为gameId
	CreateTime time.Time // 创建时间

	// 2. 状态数据
	// 2.1 物权
	Owner  string // 当前所有权拥有者。如果为空，则表示由创建者所有。只有owner有权transfer。一个NFT只有一个owner
	Renter string // 当前使用权拥有者。由owner指定。owner默认有使用权。同一时间内，一个NFT只有一个renter
	// 2.2 锁定状态
	Status    byte // 状态位（默认0） 0：正常，1：锁定（数据与状态不可变更，例如：提现待确认）
	Condition byte // 解锁条件 1：锁定直到状态机解锁 2：锁定直到用户解锁
	// 2.3 使用权回收条件（待定）
	//ReturnCondition byte // 使用权结束条件 0：到期自动结束 1：所有者触发结束 2：使用者触发结束
	//ReturnTime      byte // 到指定块高后使用权回收

	// 3. NFT业务数据
	GameId string // 当前游戏id

	// 4. NFT在游戏中的数据
	DataValue []string //key为gameId，
	DataKey   []string
}

func (self *NFT) GetData(gameId string) string {
	index := -1
	for i, key := range self.DataKey {
		if key == gameId {
			index = i
			break
		}
	}

	if -1 == index {
		return ""
	}

	return self.DataValue[index]
}

func (self *NFT) SetData(data string, gameId string) {
	index := -1
	for i, key := range self.DataKey {
		if key == gameId {
			index = i
			break
		}
	}

	if -1 == index {
		self.DataKey = append(self.DataKey, gameId)
		self.DataValue = append(self.DataValue, data)
	} else {
		self.DataValue[index] = data
	}

}

type NFTMap struct {
	NFTList   []*NFT
	NFTIDList []string
}

func (self *NFTMap) Delete(id string) {
	index := -1
	for i, key := range self.NFTIDList {
		if key == id {
			index = i
			break
		}
	}

	if -1 == index {
		return
	}

	self.NFTList = append(self.NFTList[:index], self.NFTList[index+1:]...)
	self.NFTIDList = append(self.NFTIDList[:index], self.NFTIDList[index+1:]...)
}

func (self *NFTMap) GetNFT(id string) *NFT {
	index := -1
	for i, key := range self.NFTIDList {
		if key == id {
			index = i
			break
		}
	}

	if -1 == index {
		return nil
	}
	return self.NFTList[index]
}

func (self *NFTMap) SetNFT(id string, nft *NFT) bool {
	index := -1
	for i, key := range self.NFTIDList {
		if key == id {
			index = i
			break
		}
	}

	if -1 == index {
		self.NFTIDList = append(self.NFTIDList, id)
		self.NFTList = append(self.NFTList, nft)
		return true
	}

	self.NFTList[index] = nft
	return false
}

func (self *NFTMap) GetAllNFT(gameId string) map[string]string {
	result := make(map[string]string, len(self.NFTIDList))
	for i, value := range self.NFTIDList {
		result[value] = self.NFTList[i].GetData(gameId)
	}
	return result
}

type GameData struct {
	GameIDList []string
	NFTMaps    []*NFTMap
}

func (self *GameData) GetNFTMaps(id string) *NFTMap {
	index := -1
	for i, key := range self.GameIDList {
		if key == id {
			index = i
			break
		}
	}

	if -1 == index {
		return nil
	}
	return self.NFTMaps[index]
}

func (self *GameData) SetNFT(id string, nft *NFTMap) bool {
	index := -1
	for i, key := range self.GameIDList {
		if key == id {
			index = i
			break
		}
	}

	if -1 == index {
		self.GameIDList = append(self.GameIDList, id)
		self.NFTMaps = append(self.NFTMaps, nft)
		return true
	}

	self.NFTMaps[index] = nft
	return false
}

func (self *GameData) Delete(gameId string) bool {
	index := -1
	for i, key := range self.GameIDList {
		if key == gameId {
			index = i
			break
		}
	}

	if -1 == index {
		return false
	}

	self.GameIDList = append(self.GameIDList[:index], self.GameIDList[index+1:]...)
	self.NFTMaps = append(self.NFTMaps[:index], self.NFTMaps[index+1:]...)

	return true
}

// FT发行配置
type FTSet struct {
	ID     string // 代币ID，在发行时由layer2生成。生成规则时appId-symbol。例如0x12ef3-NOX。特别的，对于公链币，layer2会自动发行，例如official-ETH
	Name   string // 代币名字，例如以太坊
	Symbol string // 代币代号，例如ETH
	AppId  string // 发行方

	TotalSupply int64 //发行总数， -1表示无限量（对于公链币，也是如此）
	Remain      int64 // 还剩下多少，-1表示无限（对于公链币，也是如此）
	Type        byte  // 类型，0代表公链币，1代表游戏发行的FT
}

// 用户ft数据结构
type FT struct {
	Balance *big.Int // 余额，注意这里会存储实际余额乘以10的9次方，用于表达浮点数。例如，用户拥有12.45币，这里的数值就是12450000000
	ID      string   // 代币ID，在发行时由layer2生成。生成规则时appId-symbol。例如0x12ef3-NOX
}
