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
	"strings"
)

var (
	dummyAppId = utility.StrToBytes("zero")
)

func (self *accountObject) checkAndCreate() {
	if self.empty() {
		self.touch()
	}
}

func (self *accountObject) GetNFTAppId(db AccountDatabase, setId, id string) string {
	nftKey := utility.StrToBytes(common.GenerateNFTKey(setId, id))
	return utility.BytesToStr(self.GetData(db, nftKey))
}

func (self *accountObject) RemoveNFTLink(db AccountDatabase, setId, id string) bool {
	nftKey := utility.StrToBytes(common.GenerateNFTKey(setId, id))
	if utility.IsEmptyByteSlice(self.GetData(db, nftKey)) {
		return false
	}

	self.RemoveData(db, nftKey)
	return true
}

func (self *accountObject) AddNFTLink(db AccountDatabase, appId, setId, id string) bool {
	nftKey := utility.StrToBytes(common.GenerateNFTKey(setId, id))
	if !utility.IsEmptyByteSlice(self.GetData(db, nftKey)) {
		return false
	}

	if 0 == len(appId) {
		self.SetData(db, nftKey, dummyAppId)
	} else {
		self.SetData(db, nftKey, utility.StrToBytes(appId))
	}

	return true
}

func (self *accountObject) UpdateNFTLink(db AccountDatabase, appId, setId, id string) {
	nftKey := utility.StrToBytes(common.GenerateNFTKey(setId, id))
	if 0 == len(appId) {
		self.SetData(db, nftKey, dummyAppId)
	} else {
		self.SetData(db, nftKey, utility.StrToBytes(appId))
	}
}

func (self *accountObject) getAllNFT(db AccountDatabase, filter string) []*types.NFT {
	self.cachedLock.Lock()
	defer self.cachedLock.Unlock()

	filtered := 0 != len(filter)
	result := make([]*types.NFT, 0)

	for id, appIdBytes := range self.cachedStorage {
		if 0 == len(appIdBytes) {
			continue
		}
		setId, nftId := common.SplitNFTKey(id)
		if 0 == len(setId) {
			continue
		}

		appId := ""
		if 0 != bytes.Compare(appIdBytes, dummyAppId) {
			appId = utility.BytesToStr(appIdBytes)
		}
		if filtered {
			if 0 == strings.Compare(appId, filter) {
				nft := &types.NFT{SetID: setId, ID: nftId, AppId: appId}
				result = append(result, nft)
			}
		} else {
			nft := &types.NFT{SetID: setId, ID: nftId, AppId: appId}
			result = append(result, nft)
		}
	}

	tr := self.getTrie(db)
	iterator := tr.NodeIterator(utility.StrToBytes(common.NFTPrefix))
	for iterator.Next(true) {
		if !iterator.Leaf() {
			continue
		}

		setId, id := common.SplitNFTKey(utility.BytesToStr(iterator.LeafKey()))
		if 0 == len(setId) {
			continue
		}

		appIdBytes := iterator.LeafBlob()
		appId := ""
		if 0 != bytes.Compare(appIdBytes, dummyAppId) {
			appId = utility.BytesToStr(appIdBytes)
		}

		key := common.GenerateNFTKey(setId, id)
		_, contains := self.cachedStorage[key]
		if contains {
			continue
		} else {
			self.cachedStorage[key] = appIdBytes
		}

		if filtered {
			if 0 == strings.Compare(appId, filter) {
				nft := &types.NFT{SetID: setId, ID: id, AppId: appId}
				result = append(result, nft)
			}
		} else {
			nft := &types.NFT{SetID: setId, ID: id, AppId: appId}
			result = append(result, nft)
		}

	}

	if 0 == len(result) {
		return nil
	}
	return result
}
