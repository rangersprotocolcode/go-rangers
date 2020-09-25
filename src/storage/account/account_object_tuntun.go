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
	"com.tuntun.rocket/node/src/storage/rlp"
	"com.tuntun.rocket/node/src/utility"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
)

func (self *accountObject) checkAndCreate() {
	if self.empty() {
		self.touch()
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

func (self *accountObject) setNFT(nft *types.NFT) {
	key := self.generateNFTKey(nft.SetID, nft.ID)
	self.cachedNFT.Lock()
	self.cachedNFTStorage[key] = nft
	self.cachedNFT.Unlock()
	self.dirtyNFTStorage[key] = nft

	self.callback()
}

func (self *accountObject) removeNFT(setId, id string) {
	key := self.generateNFTKey(setId, id)
	self.cachedNFT.Lock()
	self.cachedNFTStorage[key] = nil
	self.cachedNFT.Unlock()
	self.dirtyNFTStorage[key] = nil

	self.callback()
}

// 新增一个nft实例
func (self *accountObject) AddNFTByGameId(db AccountDatabase, appId string, nft *types.NFT) bool {
	if nil == nft || nil != self.getNFTById(db, nft.SetID, nft.ID) {
		return false
	}

	change := tuntunAddNFTChange{
		account: &self.address,
		id:      nft.ID,
		setId:   nft.SetID,
	}
	self.db.transitions = append(self.db.transitions, change)

	self.setNFT(nft)
	return true
}

func (self *accountObject) ApproveNFT(db AccountDatabase, gameId, setId, id, renter string) bool {
	nft := self.getNFT(db, gameId, setId, id)
	if nft == nil || nft.Status != 0 {
		return false
	}

	change := tuntunNFTApproveChange{
		account: &self.address,
		nft:     nft,
		prev:    nft.Renter,
	}
	self.db.transitions = append(self.db.transitions, change)

	nft.Renter = renter
	self.setNFT(nft)
	return true
}

func (self *accountObject) RemoveNFT(db AccountDatabase, appId, setId, id string) bool {
	common.DefaultLogger.Debugf("Remove nft.gameId:%s,setId:%s,id:%s", appId, setId, id)
	nft := self.getNFT(db, appId, setId, id)
	if nil == nft {
		common.DefaultLogger.Debugf("Remove nft. get nil nft")
		return false
	}

	change := tuntunRemoveNFTChange{
		nft:     nft,
		account: &self.address,
	}
	self.db.transitions = append(self.db.transitions, change)

	self.removeNFT(setId, id)
	return true
}

// 更新nft属性值
func (self *accountObject) SetNFTValueByGameId(db AccountDatabase, appId, setId, id, value string) bool {
	nft := self.getNFT(db, appId, setId, id)
	if nil == nft {
		common.DefaultLogger.Debugf("Remove nft. get nil nft")
		return false
	}

	change := tuntunNFTChange{
		account: &self.address,
		appId:   appId,
		nft:     nft,
	}

	nftValue := nft.GetData(appId)
	if 0 != len(nftValue) {
		change.prev = nftValue
	}

	self.db.transitions = append(self.db.transitions, change)
	nft.SetData(value, appId)
	self.setNFT(nft)
	return true
}

func (self *accountObject) ChangeNFTStatus(db AccountDatabase, appId, setId, id string, status byte) bool {
	nft := self.getNFT(db, appId, setId, id)
	if nft == nil {
		return false
	}

	change := tuntunNFTStatusChange{
		account: &self.address,
		nft:     nft,
		prev:    nft.Status,
	}
	self.db.transitions = append(self.db.transitions, change)
	nft.Status = status
	self.setNFT(nft)
	return true
}

func (self *accountObject) ChangeNFTStatusById(db AccountDatabase, setId, id string, status byte) bool {
	nft := self.getNFTById(db, setId, id)
	if nft == nil {
		return false
	}

	change := tuntunNFTStatusChange{
		account: &self.address,
		nft:     nft,
		prev:    nft.Status,
	}
	self.db.transitions = append(self.db.transitions, change)
	nft.Status = status
	self.setNFT(nft)
	return true
}

func (self *accountObject) getNFT(db AccountDatabase, appId, setId, id string) *types.NFT {
	nft := self.getNFTById(db, setId, id)
	if nil == nft || 0 != strings.Compare(appId, nft.AppId) {
		return nil
	}

	return nft
}

func (self *accountObject) getNFTById(db AccountDatabase, setId, id string) *types.NFT {
	key := self.generateNFTKey(setId, id)
	self.cachedNFT.RLock()
	nft, exists := self.cachedNFTStorage[key]
	self.cachedNFT.RUnlock()
	if exists {
		return nft
	}

	value, err := self.getTrie(db).TryGet(utility.StrToBytes(key))
	if err != nil {
		self.setError(err)
		return nil
	}

	if utility.IsEmptyByteSlice(value) {
		return nil
	}

	nft = &types.NFT{}
	err = rlp.DecodeBytes(value, nft)
	if nil != err {
		common.DefaultLogger.Errorf(err.Error())
		return nil
	}

	self.cachedNFT.RLock()
	self.cachedNFTStorage[key] = nft
	self.cachedNFT.RUnlock()
	return nft
}

func (self *accountObject) getAllNFT(db AccountDatabase, filter string) []*types.NFT {
	self.cachedNFT.Lock()
	defer self.cachedNFT.Unlock()

	filtered := 0 != len(filter)
	result := make([]*types.NFT, 0)
	for _, nft := range self.cachedNFTStorage {
		if nil == nft {
			continue
		}
		if filtered {
			if 0 == strings.Compare(nft.AppId, filter) {
				result = append(result, nft)
			}
		} else {
			result = append(result, nft)
		}
	}

	tr := self.getTrie(db)
	iterator := tr.NodeIterator(utility.StrToBytes("n-"))
	for iterator.Next(true) {
		if !iterator.Leaf() {
			continue
		}

		bytes := iterator.LeafBlob()
		nft := &types.NFT{}
		err := rlp.DecodeBytes(bytes, nft)
		if err != nil {
			continue
		}

		key := self.generateNFTKey(nft.SetID, nft.ID)
		_, contains := self.cachedNFTStorage[key]
		if contains {
			continue
		} else {
			self.cachedNFTStorage[key] = nft
		}

		if filtered {
			if 0 == strings.Compare(nft.AppId, filter) {
				result = append(result, nft)
			}
		} else {
			result = append(result, nft)
		}

	}

	return result
}

func (self *accountObject) callback() {
	if self.onDirty != nil {
		self.onDirty(self.Address())
		self.onDirty = nil
	}
}

func (self *accountObject) generateNFTKey(setId, id string) string {
	return fmt.Sprintf("n-%s-%s", setId, id)
}

func (self *accountObject) GetNFTSet(db AccountDatabase) *types.NFTSet {
	valueByte := self.nftSetDefinition(db)
	if nil == valueByte || 0 == len(valueByte) {
		return nil
	}

	var nftSet types.NFTSet
	err := json.Unmarshal(valueByte, &nftSet)
	if err != nil {
		return nil
	}

	self.cachedLock.RLock()
	defer self.cachedLock.RUnlock()

	nftSet.OccupiedID = make(map[string]common.Address)
	nftSet.TotalSupply = int(self.Nonce())

	iterator := self.DataIterator(db, []byte{})
	for iterator.Next() {
		nftSet.OccupiedID[utility.BytesToStr(iterator.Key)] = common.BytesToAddress(iterator.Value)
	}

	for id, addr := range self.cachedStorage {
		if addr == nil {
			delete(nftSet.OccupiedID, id)
			continue
		}
		nftSet.OccupiedID[id] = common.BytesToAddress(addr)
	}

	if 0 == len(nftSet.OccupiedID) {
		nftSet.OccupiedID = nil
	}

	return &nftSet
}
