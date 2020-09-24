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
	"math/big"
)

type transitionEntry interface {
	undo(*AccountDB)
}

type transition []transitionEntry

type (
	createObjectChange struct {
		account *common.Address
	}
	resetObjectChange struct {
		prev *accountObject
	}
	suicideChange struct {
		account     *common.Address
		prev        bool
		prevbalance *big.Int
	}

	balanceChange struct {
		account *common.Address
		prev    *big.Int
	}
	nonceChange struct {
		account *common.Address
		prev    uint64
	}
	blobChange struct {
		account *common.Address
		prev    []byte
	}
	storageChange struct {
		account  *common.Address
		key      []byte
		prevalue []byte
	}

	refundChange struct {
		prev uint64
	}
	addLogChange struct {
		txhash common.Hash
	}
	touchChange struct {
		account   *common.Address
		prev      bool
		prevDirty bool
	}
)

func (ch createObjectChange) undo(s *AccountDB) {
	s.accountObjects.Delete(*ch.account)
	delete(s.accountObjectsDirty, *ch.account)
}

func (ch resetObjectChange) undo(s *AccountDB) {
	s.setAccountObject(ch.prev)
}

func (ch suicideChange) undo(s *AccountDB) {
	obj := s.getAccountObject(*ch.account, false)
	if obj != nil {
		obj.suicided = ch.prev
		obj.setBalance(ch.prevbalance)
	}
}

var ripemd = common.StringToAddress("0000000000000000000000000000000000000003")

func (ch touchChange) undo(s *AccountDB) {
	if !ch.prev && *ch.account != ripemd {
		s.getAccountObject(*ch.account, false).touched = ch.prev
		if !ch.prevDirty {
			delete(s.accountObjectsDirty, *ch.account)
		}
	}
}

func (ch balanceChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account, false).setBalance(ch.prev)
}

func (ch nonceChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account, false).setNonce(ch.prev)
}

func (ch blobChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account, false).setBlob(ch.prev)
}

func (ch storageChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account, false).setData(ch.key, ch.prevalue)
}

func (ch refundChange) undo(s *AccountDB) {
	s.refund = ch.prev
}
