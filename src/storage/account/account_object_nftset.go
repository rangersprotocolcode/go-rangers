// Copyright 2020 The RangersProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RangersProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RangersProtocol library is distributed in the hope that it will be useful,
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
	"com.tuntun.rocket/node/src/utility"
	"fmt"
)

var (
	NFTSetOwnerString = "nso"
	nftSetOwnerKey    = utility.StrToBytes(NFTSetOwnerString)
)

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
