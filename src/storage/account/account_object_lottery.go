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

import "com.tuntun.rocket/node/src/common"

func (self *accountObject) GetLotteryDefinition(db AccountDatabase) []byte {
	valueByte := self.nftSetDefinition(db)
	return valueByte
}

func (ao *accountObject) SetLotteryOwner(db AccountDatabase, owner string) {
	ao.SetData(db, nftSetOwnerKey, common.FromHex(owner))
}

func (ao *accountObject) GetLotteryOwner(db AccountDatabase) string {
	return common.ToHex(ao.GetData(db, nftSetOwnerKey))
}

func (ao *accountObject) SetLotteryDefinition(hash common.Hash, code []byte) {
	prevCode := ao.nftSetDefinition(ao.db.db)
	ao.db.transitions = append(ao.db.transitions, nftSetDefinitionChange{
		account:  &ao.address,
		prevhash: ao.NFTSetDefinitionHash(),
		prev:     prevCode,
	})
	ao.setLotteryDefinition(hash, code)
}

func (ao *accountObject) setLotteryDefinition(hash common.Hash, code []byte) {
	ao.data.kind = NFTSET_TYPE
	ao.nftSet = code
	ao.data.NFTSetDefinitionHash = hash[:]
	ao.dirtyNFTSet = true
	if ao.onDirty != nil {
		ao.onDirty(ao.Address())
		ao.onDirty = nil
	}
}