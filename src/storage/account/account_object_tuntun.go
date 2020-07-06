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
	"encoding/json"
	"math/big"
)

func (self *accountObject) checkAndCreate(gameId string) {
	if self.empty() {
		self.touch()
	}

	if self.data.GameData.GetNFTMaps(gameId) == nil {
		self.data.GameData.SetNFTMaps(gameId, &types.NFTMap{})
		self.db.transitions = append(self.db.transitions, createGameDataChange{
			account: &self.address,
			gameId:  gameId,
		})
	}
}

func (self *accountObject) getFT(ftList []*types.FT, name string) *types.FT {
	for _, ft := range ftList {
		if ft.ID == name {
			return ft
		}
	}

	return nil
}

func (c *accountObject) AddFT(amount *big.Int, name string) bool {
	if amount.Sign() == 0 {
		if c.empty() {
			c.touch()
		}

		return true
	}
	raw := c.getFT(c.data.Ft, name)
	if nil == raw {
		return c.SetFT(new(big.Int).Set(amount), name)
	} else {
		return c.SetFT(new(big.Int).Add(raw.Balance, amount), name)
	}

}

func (c *accountObject) SubFT(amount *big.Int, name string) (*big.Int, bool) {
	if amount.Sign() == 0 {
		raw := c.getFT(c.data.Ft, name)
		if nil == raw {
			return big.NewInt(0), true
		}
		return raw.Balance, true
	}

	raw := c.getFT(c.data.Ft, name)
	// 余额不足就滚粗
	if nil == raw || raw.Balance.Cmp(amount) == -1 {
		return nil, false
	}

	left := new(big.Int).Sub(raw.Balance, amount)
	c.SetFT(left, name)

	return left, true
}

func (self *accountObject) SetFT(amount *big.Int, name string) bool {
	raw := self.getFT(self.data.Ft, name)
	change := tuntunFTChange{
		account: &self.address,
		name:    name,
	}
	if raw != nil {
		change.prev = new(big.Int).Set(raw.Balance)
	} else {
		change.prev = nil
	}

	self.db.transitions = append(self.db.transitions, change)
	return self.setFT(amount, name)
}

func (self *accountObject) setFT(amount *big.Int, name string) bool {
	if nil == amount || amount.Sign() == 0 {
		index := -1
		for i, ft := range self.data.Ft {
			if ft.ID == name {
				index = i
				break
			}
		}
		if -1 != index {
			self.data.Ft = append(self.data.Ft[:index], self.data.Ft[index+1:]...)
		}
	} else {
		ftObject := self.getFT(self.data.Ft, name)
		if ftObject == nil {
			ftObject = &types.FT{}
			ftObject.ID = name

			self.data.Ft = append(self.data.Ft, ftObject)
		}
		ftObject.Balance = new(big.Int).Set(amount)

		// 溢出判断
		if ftObject.Balance.Sign() == -1 {
			return false
		}
	}

	self.callback()
	return true
}

// 新增一个nft实例
func (self *accountObject) AddNFTByGameId(gameId string, nft *types.NFT) bool {
	if nil == nft {
		return false
	}

	self.checkAndCreate(gameId)
	nftMaps := self.data.GameData.GetNFTMaps(gameId)
	if nftMaps.GetNFT(nft.SetID, nft.ID) != nil {
		return false
	}

	change := tuntunAddNFTChange{
		account: &self.address,
		gameId:  gameId,
		id:      nft.ID,
		setId:   nft.SetID,
	}
	self.db.transitions = append(self.db.transitions, change)
	ok := nftMaps.SetNFT(nft)
	if ok {
		self.callback()
	}

	return ok
}

func (self *accountObject) ApproveNFT(gameId, setId, id, renter string) bool {
	nft := self.getNFT(gameId, setId, id)
	if nft == nil || nft.Status != 0 {
		return false
	}

	change := tuntunNFTApproveChange{
		account: &self.address,
		appId:   gameId,
		id:      nft.ID,
		setId:   nft.SetID,
		prev:    nft.Renter,
	}
	self.db.transitions = append(self.db.transitions, change)
	nft.Renter = renter

	self.callback()
	return true
}

func (self *accountObject) RemoveNFT(gameId, setId, id string) bool {
	common.DefaultLogger.Debugf("Remove nft.gameId:%s,setId:%s,id:%s", gameId, setId, id)
	data := self.data.GameData.GetNFTMaps(gameId)
	if nil == data {
		common.DefaultLogger.Debugf("Remove nft.get nil data")
		return false
	}
	jsonBefore, _ := json.Marshal(data)
	common.DefaultLogger.Debugf("before delete.data:%s", jsonBefore)

	nft := data.Delete(setId, id)
	jsonAfter, _ := json.Marshal(data)
	common.DefaultLogger.Debugf("after delete.data:%s", jsonAfter)
	common.DefaultLogger.Debugf("after delete.deleted nft:%v", nft)

	change := tuntunRemoveNFTChange{
		nft:     nft,
		account: &self.address,
		gameId:  gameId,
	}
	self.db.transitions = append(self.db.transitions, change)
	self.callback()
	return true
}

// 更新nft属性值
func (self *accountObject) SetNFTValueByGameId(gameId, setId, id, value string) bool {
	data := self.data.GameData.GetNFTMaps(gameId)
	if nil == data || data.Empty() {
		return false
	}

	change := tuntunNFTChange{
		account: &self.address,
		gameId:  gameId,
		setId:   setId,
		id:      id,
	}
	nft := data.GetNFT(setId, id)
	if nil != nft {
		nftValue := nft.GetData(gameId)
		if 0 != len(nftValue) {
			change.prev = nftValue
		}
	}
	self.db.transitions = append(self.db.transitions, change)

	return self.setNFTByGameId(gameId, setId, id, value)
}

func (self *accountObject) setNFTByGameId(gameId, setId, id, value string) bool {
	data := self.data.GameData.GetNFTMaps(gameId)
	if nil == data {
		return false
	}

	nftData := data.GetNFT(setId, id)
	if nil != nftData {
		nftData.SetData(value, gameId)
		self.callback()
	}

	return true
}

func (self *accountObject) ChangeNFTStatus(gameId, setId, id string, status byte) bool {
	nft := self.getNFT(gameId, setId, id)
	if nft == nil {
		return false
	}

	change := tuntunNFTStatusChange{
		account: &self.address,
		appId:   gameId,
		id:      nft.ID,
		setId:   nft.SetID,
		prev:    nft.Status,
	}
	self.db.transitions = append(self.db.transitions, change)
	nft.Status = status
	return true
}

func (self *accountObject) ChangeNFTStatusById(setId, id string, status byte) bool {
	nft := self.getNFTById(setId, id)
	if nft == nil {
		return false
	}

	change := tuntunNFTStatusChange{
		account: &self.address,
		appId:   "",
		id:      nft.ID,
		setId:   nft.SetID,
		prev:    nft.Status,
	}
	self.db.transitions = append(self.db.transitions, change)
	nft.Status = status
	return true
}

func (self *accountObject) getNFT(appId, setId, id string) *types.NFT {
	data := self.data.GameData.GetNFTMaps(appId)
	if nil == data {
		return nil
	}

	return data.GetNFT(setId, id)
}

func (self *accountObject) getNFTById(setId, id string) *types.NFT {
	data := self.data.GameData.NFTMaps
	if nil == data || 0 == len(data) {
		return nil
	}

	for _, nftMap := range data {
		nft := nftMap.GetNFT(setId, id)
		if nil != nft {
			return nft
		}
	}

	return nil
}

func (self *accountObject) callback() {
	if self.onDirty != nil {
		self.onDirty(self.Address())
		self.onDirty = nil
	}
}
