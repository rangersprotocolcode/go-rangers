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
	"bytes"
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/rlp"
	"com.tuntun.rocket/node/src/utility"
	"fmt"
	"strings"
)

var (
	setIDKey      = utility.StrToBytes("s")
	nameKey       = utility.StrToBytes("n")
	symbolKey     = utility.StrToBytes("y")
	idKey         = utility.StrToBytes("i")
	creatorKey    = utility.StrToBytes("c")
	createTimeKey = utility.StrToBytes("ct")
	ownerKey      = utility.StrToBytes("o")
	renterKey     = utility.StrToBytes("r")
	statusKey     = utility.StrToBytes("t")
	conditionKey  = utility.StrToBytes("co")
	appIdKey      = utility.StrToBytes("a")
	importedKey   = utility.StrToBytes("im")
	dataPrefix    = "d:"
)

func (self *accountObject) generateNFTDataKey(key string) string {
	return fmt.Sprintf("%s%s", dataPrefix, key)
}

func (self *accountObject) checkOwner(db AccountDatabase, addr common.Address) bool {
	ownerAddressBytes := self.GetData(db, ownerKey)
	ownerAddress := common.HexStringToAddress(utility.BytesToStr(ownerAddressBytes))
	return 0 == bytes.Compare(ownerAddress.Bytes(), addr.Bytes())
}

// 新增一个nft实例
func (self *accountObject) AddNFT(db AccountDatabase, nft *types.NFT) bool {
	if nil == nft {
		return false
	}

	self.SetData(db, setIDKey, utility.StrToBytes(nft.SetID))
	self.SetData(db, nameKey, utility.StrToBytes(nft.Name))
	self.SetData(db, symbolKey, utility.StrToBytes(nft.Symbol))
	self.SetData(db, idKey, utility.StrToBytes(nft.ID))
	self.SetData(db, creatorKey, utility.StrToBytes(nft.Creator))
	self.SetData(db, createTimeKey, utility.StrToBytes(nft.CreateTime))
	self.SetData(db, ownerKey, utility.StrToBytes(nft.Owner))
	self.SetData(db, renterKey, utility.StrToBytes(nft.Renter))
	self.SetData(db, statusKey, []byte{nft.Status})
	self.SetData(db, conditionKey, []byte{nft.Condition})
	self.SetData(db, appIdKey, utility.StrToBytes(nft.AppId))
	self.SetData(db, importedKey, utility.StrToBytes(nft.Imported))

	for key, value := range nft.Data {
		self.SetData(db, utility.StrToBytes(self.generateNFTDataKey(key)), utility.StrToBytes(value))
	}
	return true
}

func (self *accountObject) GetNFT(db AccountDatabase) *types.NFT {
	nft := &types.NFT{
		SetID:      utility.BytesToStr(self.GetData(db, setIDKey)),
		Name:       utility.BytesToStr(self.GetData(db, nameKey)),
		Symbol:     utility.BytesToStr(self.GetData(db, symbolKey)),
		ID:         utility.BytesToStr(self.GetData(db, idKey)),
		Creator:    utility.BytesToStr(self.GetData(db, creatorKey)),
		CreateTime: utility.BytesToStr(self.GetData(db, createTimeKey)),
		Owner:      utility.BytesToStr(self.GetData(db, ownerKey)),
		Renter:     utility.BytesToStr(self.GetData(db, renterKey)),
		AppId:      utility.BytesToStr(self.GetData(db, appIdKey)),
		Imported:   utility.BytesToStr(self.GetData(db, importedKey)),
		Data:       make(map[string]string),
	}

	status := self.GetData(db, statusKey)
	if nil != status && 1 == len(status) {
		nft.Status = status[0]
	}

	contidion := self.GetData(db, conditionKey)
	if nil != contidion && 1 == len(contidion) {
		nft.Condition = contidion[0]
	}

	self.cachedLock.RLock()
	defer self.cachedLock.RUnlock()
	for key, value := range self.cachedStorage {
		if strings.HasPrefix(key, dataPrefix) {
			nft.Data[key[2:]] = utility.BytesToStr(value)
		}
	}
	iterator := self.DataIterator(db, utility.StrToBytes(dataPrefix))
	for iterator.Next() {
		key := utility.BytesToStr(iterator.Key)
		_, contains := self.cachedStorage[key]

		if contains {
			continue
		}
		self.cachedStorage[key] = iterator.Value
		nft.Data[key[2:]] = utility.BytesToStr(iterator.Value)
	}

	return nft
}
func (self *accountObject) ApproveNFT(db AccountDatabase, owner common.Address, renter string) bool {
	if !self.checkOwner(db, owner) {
		return false
	}

	status := self.GetData(db, statusKey)
	if nil == status || 1 != len(status) || status[0] != 0 {
		return false
	}

	self.SetData(db, renterKey, utility.StrToBytes(renter))
	return true
}

// 更新nft属性值
func (self *accountObject) SetNFTValue(db AccountDatabase, addr common.Address, appId, propertyName, value string) bool {
	if 0 != strings.Compare(appId, utility.BytesToStr(self.GetData(db, appIdKey))) {
		return false
	}

	self.SetData(db, utility.StrToBytes(self.generateNFTDataKey(common.GenerateAppIdProperty(appId, propertyName))), utility.StrToBytes(value))
	return true
}

// 更新nft属性值
func (self *accountObject) SetNFTValueByGameId(db AccountDatabase, addr common.Address, appId, value string) bool {
	if 0 != strings.Compare(appId, utility.BytesToStr(self.GetData(db, appIdKey))) {
		return false
	}

	self.SetData(db, utility.StrToBytes(self.generateNFTDataKey(appId)), utility.StrToBytes(value))
	return true
}

func (self *accountObject) ChangeNFTStatus(db AccountDatabase, addr common.Address, status byte) bool {
	if !self.checkOwner(db, addr) {
		return false
	}

	self.SetData(db, statusKey, []byte{status})
	return true
}

func (self *accountObject) GetNFTSet(db AccountDatabase) *types.NFTSet {
	valueByte := self.nftSetDefinition(db)
	if nil == valueByte || 0 == len(valueByte) {
		return nil
	}

	var definition types.NftSetDefinition
	err := rlp.DecodeBytes(valueByte, &definition)
	if err != nil {
		return nil
	}

	self.cachedLock.RLock()
	defer self.cachedLock.RUnlock()

	nftSet := definition.ToNFTSet()
	nftSet.OccupiedID = make(map[string]common.Address)
	nftSet.TotalSupply = self.Nonce()

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
