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
	NFTSetOwnerString = "nso"
	nftSetOwnerKey    = utility.StrToBytes(NFTSetOwnerString)
)

func (self *accountObject) CheckNFTSetOwner(db AccountDatabase, owner string) bool {
	return 0 == strings.Compare(strings.ToLower(self.GetNFTSetOwner(db)), strings.ToLower(owner))
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

	nftSet := definition.ToNFTSet()
	nftSet.Owner = self.GetNFTSetOwner(db)

	self.cachedLock.RLock()
	defer self.cachedLock.RUnlock()

	nftSet.OccupiedID = make(map[string]common.Address)
	nftSet.TotalSupply = self.Nonce()

	iterator := self.DataIterator(db, []byte{})
	for iterator.Next() {
		key := utility.BytesToStr(iterator.Key)
		if 0 == strings.Compare(key, NFTSetOwnerString) || strings.HasPrefix(key, common.LockPrefix) {
			continue
		}
		nftSet.OccupiedID[key] = common.BytesToAddress(iterator.Value)
	}

	for id, addr := range self.cachedStorage {
		if 0 == strings.Compare(id, NFTSetOwnerString) || strings.HasPrefix(id, common.LockPrefix) {
			continue
		}

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

func (ao *accountObject) SetNFTSetOwner(db AccountDatabase, owner string) {
	ao.SetData(db, nftSetOwnerKey, common.FromHex(owner))
}

func (ao *accountObject) GetNFTSetOwner(db AccountDatabase) string {
	return common.ToHex(ao.GetData(db, nftSetOwnerKey))
}

func (ao *accountObject) SetNFTSetDefinition(hash common.Hash, code []byte) {
	prevCode := ao.nftSetDefinition(ao.db.db)
	ao.db.transitions = append(ao.db.transitions, nftSetDefinitionChange{
		account:  &ao.address,
		prevhash: ao.NFTSetDefinitionHash(),
		prev:     prevCode,
	})
	ao.setNFTSetDefinition(hash, code)
}

func (ao *accountObject) setNFTSetDefinition(hash common.Hash, code []byte) {
	ao.data.kind = NFTSET_TYPE
	ao.nftSet = code
	ao.data.NFTSetDefinitionHash = hash[:]
	ao.dirtyNFTSet = true
	if ao.onDirty != nil {
		ao.onDirty(ao.Address())
		ao.onDirty = nil
	}
}

func (ao *accountObject) nftSetDefinition(db AccountDatabase) []byte {
	if ao.nftSet != nil {
		return ao.nftSet
	}
	if bytes.Equal(ao.NFTSetDefinitionHash(), emptyCodeHash[:]) {
		return nil
	}
	code, err := db.ContractCode(ao.addrHash, common.BytesToHash(ao.NFTSetDefinitionHash()))
	if err != nil {
		ao.setError(fmt.Errorf("can't load code hash %x: %v", ao.NFTSetDefinitionHash(), err))
	}
	ao.nftSet = code
	return code
}