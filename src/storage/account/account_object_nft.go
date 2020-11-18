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
	"com.tuntun.rocket/node/src/utility"
	"fmt"
	"math/big"
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
	lockKey       = utility.StrToBytes("l")

	dataPrefix  = "d:"
	comboPrefix = "cm:"
	bntPrefix   = fmt.Sprintf("%s%s", comboPrefix, "bn:")
	ftPrefix    = fmt.Sprintf("%s%s", comboPrefix, "f:")
	nftPrefix   = fmt.Sprintf("%s%s", comboPrefix, "n:")
)

func (self *accountObject) generateNFTDataKey(key string) string {
	return fmt.Sprintf("%s%s", dataPrefix, key)
}

func (self *accountObject) checkOwner(db AccountDatabase, addr common.Address) bool {
	ownerAddressBytes := self.GetData(db, ownerKey)
	return 0 == bytes.Compare(ownerAddressBytes, addr.Bytes())
}

func (self *accountObject) SetOwner(db AccountDatabase, owner string) {
	self.SetData(db, ownerKey, common.FromHex(owner))
}

func (self *accountObject) SetAppId(db AccountDatabase, appId string) {
	self.SetData(db, appIdKey, utility.StrToBytes(appId))
}

func (self *accountObject) SetLock(db AccountDatabase, lock []byte) {
	self.SetData(db, lockKey, lock)
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
	self.SetData(db, creatorKey, common.FromHex(nft.Creator))
	self.SetData(db, createTimeKey, utility.StrToBytes(nft.CreateTime))
	self.SetData(db, ownerKey, common.FromHex(nft.Owner))
	if 0 != len(nft.Renter) {
		self.SetData(db, renterKey, common.FromHex(nft.Renter))
	}
	if 0 != nft.Status {
		self.SetData(db, statusKey, []byte{nft.Status})
	}
	if 0 != nft.Condition {
		self.SetData(db, conditionKey, []byte{nft.Condition})
	}
	self.SetData(db, appIdKey, utility.StrToBytes(nft.AppId))
	if 0 != len(nft.Imported) {
		self.SetData(db, importedKey, utility.StrToBytes(nft.Imported))
	}

	for key, value := range nft.Data {
		self.SetData(db, utility.StrToBytes(self.generateNFTDataKey(key)), utility.StrToBytes(value))
	}

	self.data.kind = NFT_TYPE
	return true
}

func (self *accountObject) GetNFT(db AccountDatabase) *types.NFT {
	nft := &types.NFT{
		SetID:      utility.BytesToStr(self.GetData(db, setIDKey)),
		Name:       utility.BytesToStr(self.GetData(db, nameKey)),
		Symbol:     utility.BytesToStr(self.GetData(db, symbolKey)),
		ID:         utility.BytesToStr(self.GetData(db, idKey)),
		Creator:    common.ToHex(self.GetData(db, creatorKey)),
		CreateTime: utility.BytesToStr(self.GetData(db, createTimeKey)),
		Owner:      common.ToHex(self.GetData(db, ownerKey)),
		Renter:     common.ToHex(self.GetData(db, renterKey)),
		AppId:      utility.BytesToStr(self.GetData(db, appIdKey)),
		Imported:   utility.BytesToStr(self.GetData(db, importedKey)),
		Lock:       common.Bytes2Hex(self.GetData(db, lockKey)),
		Data:       make(map[string]string),
	}

	nft.Status = self.getNFTStatus(db)

	contidion := self.GetData(db, conditionKey)
	if nil != contidion && 1 == len(contidion) {
		nft.Condition = contidion[0]
	}

	// getLockedBalance
	key := utility.StrToBytes(fmt.Sprintf("%s%s", comboPrefix, "b"))
	value := self.GetData(db, key)

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

	nft.ComboResource = self.getCombo(db)
	if nil == value || utility.IsEmptyByteSlice(value) {
		nft.ComboResource.Balance = "0"
	} else {
		now := new(big.Int).SetBytes(value)
		nft.ComboResource.Balance = utility.BigIntToStr(now)
	}
	return nft
}
func (self *accountObject) ApproveNFT(db AccountDatabase, owner common.Address, renter string) bool {
	if !self.checkOwner(db, owner) {
		self.log.Errorf("check owner error: %s, approve failed", owner.String())
		return false
	}

	status := self.getNFTStatus(db)
	if status != 0 {
		self.log.Errorf("check status error. status: %d, owner: %s, approve failed", status, owner.String())
		return false
	}

	self.SetData(db, renterKey, common.FromHex(renter))
	return true
}

// 更新nft属性值
func (self *accountObject) SetNFTProperty(db AccountDatabase, appId, propertyName, value string) bool {
	if 0 != strings.Compare(appId, utility.BytesToStr(self.GetData(db, appIdKey))) {
		return false
	}

	self.SetData(db, utility.StrToBytes(self.generateNFTDataKey(common.GenerateAppIdProperty(appId, propertyName))), utility.StrToBytes(value))
	return true
}

// 更新nft属性值
func (self *accountObject) SetNFTValueByGameId(db AccountDatabase, appId, value string) bool {
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

func (self *accountObject) getNFTStatus(db AccountDatabase) byte {
	status := self.GetData(db, statusKey)
	if nil == status || 1 != len(status) {
		return 0
	}

	return status[0]
}

// nft记录锁定的target
func (ao *accountObject) lockNFTSelf(db AccountDatabase, owner, target common.Address) bool {
	if !ao.checkOwner(db, owner) {
		return false
	}

	if 0 != ao.getNFTStatus(db) {
		return false
	}

	ao.SetLock(db, target.Bytes())
	ao.SetData(db, statusKey, []byte{3})
	return true
}

func (ao *accountObject) unlockNFTSelf(db AccountDatabase) {
	ao.SetLock(db, nil)
	ao.SetData(db, statusKey, []byte{0})
}

func (ao *accountObject) setComboCoin(db AccountDatabase, key []byte, amount *big.Int) {
	if nil == amount || 0 == amount.Sign() {
		return
	}

	value := ao.GetData(db, key)
	if nil == value || utility.IsEmptyByteSlice(value) {
		ao.SetData(db, key, amount.Bytes())
	} else {
		now := new(big.Int).SetBytes(value)
		now = now.Add(now, amount)
		ao.SetData(db, key, now.Bytes())
	}
}

func (ao *accountObject) setComboBalance(db AccountDatabase, amount *big.Int) {
	ao.setComboCoin(db, utility.StrToBytes(fmt.Sprintf("%s%s", comboPrefix, "b")), amount)
}

func (ao *accountObject) setComboBNT(db AccountDatabase, amount *big.Int, bnt string) {
	ao.setComboCoin(db, utility.StrToBytes(fmt.Sprintf("%s%s", bntPrefix, bnt)), amount)
}

func (ao *accountObject) setComboFT(db AccountDatabase, amount *big.Int, ft string) {
	ao.setComboCoin(db, utility.StrToBytes(fmt.Sprintf("%s%s", ftPrefix, ft)), amount)
}

func (ao *accountObject) setComboNFT(db AccountDatabase, setId, id string) {
	key := utility.StrToBytes(fmt.Sprintf("%s%s:%s", nftPrefix, setId, id))
	ao.SetData(db, key, []byte{0})
}

// 调用的地方已经加锁了，这里不用加锁了
func (ao *accountObject) getCombo(db AccountDatabase) types.ComboResource {
	result := types.ComboResource{
		Coin: make(map[string]string),
		FT:   make(map[string]string),
		NFT:  make([]types.NFTID, 0),
	}

	//	bnt/ft/nft
	for key, value := range ao.cachedStorage {
		if strings.HasPrefix(key, bntPrefix) {
			result.Coin[key[len(bntPrefix):]] = utility.BigIntBytesToStr(value)
		}
		if strings.HasPrefix(key, ftPrefix) {
			result.FT[key[len(ftPrefix):]] = utility.BigIntBytesToStr(value)
		}
		if strings.HasPrefix(key, nftPrefix) {
			ids := strings.Split(key[len(nftPrefix):], ":")
			if 2 == len(ids) {
				nft := types.NFTID{
					SetId: ids[0],
					Id:    ids[1],
				}
				result.NFT = append(result.NFT, nft)
			}
		}
	}

	iterator := ao.DataIterator(db, utility.StrToBytes(comboPrefix))
	for iterator.Next() {
		key := utility.BytesToStr(iterator.Key)

		_, contains := ao.cachedStorage[key]
		if contains {
			continue
		}

		ao.cachedStorage[key] = iterator.Value

		if strings.HasPrefix(key, bntPrefix) {
			result.Coin[key[len(bntPrefix):]] = utility.BigIntBytesToStr(iterator.Value)
		}
		if strings.HasPrefix(key, ftPrefix) {
			result.FT[key[len(ftPrefix):]] = utility.BigIntBytesToStr(iterator.Value)
		}
		if strings.HasPrefix(key, nftPrefix) {
			ids := strings.Split(key[len(nftPrefix):], ":")
			if 2 == len(ids) {
				nft := types.NFTID{
					SetId: ids[0],
					Id:    ids[1],
				}
				result.NFT = append(result.NFT, nft)
			}
		}
	}

	return result
}
