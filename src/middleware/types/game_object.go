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

package types

import (
	"com.tuntun.rocket/node/src/common"
	"encoding/json"
	"math/big"
	"strconv"
)

// NFTSet 数据结构综述
type NFTSet struct {
	SetID       string `json:"setId,omitempty"`
	Name        string `json:"name,omitempty"`
	Symbol      string `json:"symbol,omitempty"`
	MaxSupply   int    `json:"maxSupply,omitempty"`   // 最大发行量，等于0则表示无限量
	TotalSupply int    `json:"totalSupply,omitempty"` // 历史上发行量
	Creator     string `json:"creator,omitempty"`
	Owner       string `json:"owner,omitempty"`
	CreateTime  string `json:"createTime,omitempty"`

	// 已经发行的NFTID及其拥有者
	OccupiedID map[string]common.Address `json:"occupied,omitempty"` // 当前在layer2里的nft
}

func (self *NFTSet) ToBlob() []byte {
	nftSetMap := make(map[string]interface{}, 12)
	nftSetMap["setId"] = self.SetID
	nftSetMap["name"] = self.Name
	nftSetMap["symbol"] = self.Symbol
	nftSetMap["maxSupply"] = self.MaxSupply
	nftSetMap["creator"] = self.Creator
	nftSetMap["owner"] = self.Owner
	nftSetMap["createTime"] = self.CreateTime
	bytes, _ := json.Marshal(nftSetMap)
	return bytes
}

func (self *NFTSet) ToJSONString() string {
	nftSetMap := make(map[string]interface{}, 12)
	nftSetMap["setId"] = self.SetID
	nftSetMap["name"] = self.Name
	nftSetMap["symbol"] = self.Symbol
	nftSetMap["maxSupply"] = self.MaxSupply
	nftSetMap["totalSupply"] = self.TotalSupply
	nftSetMap["currentSupply"] = strconv.Itoa(len(self.OccupiedID))
	nftSetMap["creator"] = self.Creator
	nftSetMap["owner"] = self.Owner
	nftSetMap["createTime"] = self.CreateTime
	nftSetMap["occupied"] = self.OccupiedID
	bytes, _ := json.Marshal(nftSetMap)
	return string(bytes)
}

func (self *NFTSet) ToJSON() map[string]interface{} {
	nftSetMap := make(map[string]interface{}, 12)
	nftSetMap["setId"] = self.SetID
	nftSetMap["name"] = self.Name
	nftSetMap["symbol"] = self.Symbol
	nftSetMap["maxSupply"] = self.MaxSupply
	nftSetMap["totalSupply"] = self.TotalSupply
	nftSetMap["currentSupply"] = strconv.Itoa(len(self.OccupiedID))
	nftSetMap["creator"] = self.Creator
	nftSetMap["owner"] = self.Owner
	nftSetMap["createTime"] = self.CreateTime
	nftSetMap["occupied"] = self.OccupiedID

	return nftSetMap
}

type NFT struct {
	//
	SetID  string `json:"setId,omitempty"`
	Name   string `json:"name,omitempty"`
	Symbol string `json:"symbol,omitempty"`

	// 1. 通用数据
	ID         string `json:"id,omitempty"`         // NFT自身ID，创建时指定。创建后不可修改
	Creator    string `json:"creator,omitempty"`    // 初次创建者，一般为appId
	CreateTime string `json:"createTime,omitempty"` // 创建时间

	// 2. 状态数据
	// 2.1 物权
	Owner  string `json:"owner,omitempty"`  // 当前所有权拥有者。如果为空，则表示由创建者所有。只有owner有权transfer。一个NFT只有一个owner
	Renter string `json:"renter,omitempty"` // 当前使用权拥有者。由owner指定。owner默认有使用权。同一时间内，一个NFT只有一个renter
	// 2.2 锁定状态
	Status    byte `json:"status,omitempty"`    // 状态位（默认0） 0：正常，1：锁定（数据与状态不可变更，例如：提现待确认）
	Condition byte `json:"condition,omitempty"` // 解锁条件 1：锁定直到状态机解锁 2：锁定直到用户解锁
	// 2.3 使用权回收条件（待定）
	//ReturnCondition byte // 使用权结束条件 0：到期自动结束 1：所有者触发结束 2：使用者触发结束
	//ReturnTime      byte // 到指定块高后使用权回收

	// 3. NFT业务数据
	AppId string `json:"appId,omitempty"` // 当前游戏id

	// 4. NFT在游戏中的数据
	DataValue []string `json:"dataValue,omitempty"` //key为appId，
	DataKey   []string `json:"dataKey,omitempty"`

	// 5. 从外部导入的相关信息
	Imported string `json:"imported,omitempty"`
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

func (self *NFT) ToJSONString() string {
	bytes, _ := json.Marshal(self.ToMap())
	return string(bytes)
}

func (self *NFT) ToMap() map[string]interface{} {
	nftMap := make(map[string]interface{}, 12)
	nftMap["setId"] = self.SetID
	nftMap["name"] = self.Name
	nftMap["symbol"] = self.Symbol
	nftMap["id"] = self.ID
	nftMap["creator"] = self.Creator
	nftMap["createTime"] = self.CreateTime
	nftMap["owner"] = self.Owner
	nftMap["renter"] = self.Renter
	nftMap["status"] = self.Status
	nftMap["condition"] = self.Condition
	nftMap["appId"] = self.AppId
	nftMap["imported"] = self.Imported

	data := make(map[string]string, 0)
	for i := range self.DataKey {
		data[self.DataKey[i]] = self.DataValue[i]
	}
	nftMap["data"] = data
	return nftMap
}

// FT发行配置
type FTSet struct {
	ID         string // 代币ID，在发行时由layer2生成。生成规则时appId-symbol。例如0x12ef3-NOX。特别的，对于公链币，layer2会自动发行，例如official-ETH
	Name       string // 代币名字，例如以太坊
	Symbol     string // 代币代号，例如ETH
	AppId      string // 发行方
	Owner      string // 所有者
	CreateTime string // 发行时间

	MaxSupply   *big.Int // 发行总数， 0表示无限量（对于公链币，也是如此）
	TotalSupply *big.Int // 已经发行了多少
	Type        byte     // 类型，0代表公链币，1代表游戏发行的FT
}

// 用户ft数据结构
type FT struct {
	Balance *big.Int // 余额，注意这里会存储实际余额乘以10的9次方，用于表达浮点数。例如，用户拥有12.45币，这里的数值就是12450000000
	ID      string   // 代币ID，在发行时由layer2生成。生成规则时appId-symbol。例如0x12ef3-NOX
}
