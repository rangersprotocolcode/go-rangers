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
	"math/big"
)

type (
	tuntunFTChange struct {
		account *common.Address
		prev    *big.Int
		name    string
	}
	tuntunNFTChange struct {
		account *common.Address
		prev    string
		nft     *types.NFT
		appId   string
	}
	tuntunNFTApproveChange struct {
		account *common.Address
		prev    string
		nft     *types.NFT
	}
	tuntunNFTStatusChange struct {
		account *common.Address
		prev    byte
		nft     *types.NFT
	}
	tuntunAddNFTChange struct {
		account *common.Address
		id      string
		setId   string
	}
	tuntunRemoveNFTChange struct {
		account *common.Address
		nft     *types.NFT
	}
)

func (ch tuntunFTChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account, false).setFT(ch.prev, ch.name)
}
func (ch tuntunNFTChange) undo(s *AccountDB) {
	ch.nft.SetData(ch.prev, ch.appId)
	s.getAccountObject(*ch.account, false).setNFT(ch.nft)
}

func (ch tuntunAddNFTChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account, false).removeNFT(ch.setId, ch.id)
}

func (ch tuntunRemoveNFTChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account, false).setNFT(ch.nft)
}

func (ch tuntunNFTApproveChange) undo(s *AccountDB) {
	object := s.getAccountObject(*ch.account, false)
	ch.nft.Renter = ch.prev
	object.setNFT(ch.nft)

}

func (ch tuntunNFTStatusChange) undo(s *AccountDB) {
	object := s.getAccountObject(*ch.account, false)
	ch.nft.Status = ch.prev
	object.setNFT(ch.nft)
}
