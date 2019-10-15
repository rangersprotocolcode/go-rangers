package account

import (
	"math/big"
	"x/src/common"
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
	storageChange struct {
		account  *common.Address
		key      []byte
		prevalue []byte
	}
	codeChange struct {
		account            *common.Address
		prevcode, prevhash []byte
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
	obj := s.getAccountObject(*ch.account,false)
	if obj != nil {
		obj.suicided = ch.prev
		obj.setBalance(ch.prevbalance)
	}
}

var ripemd = common.StringToAddress("0000000000000000000000000000000000000003")

func (ch touchChange) undo(s *AccountDB) {
	if !ch.prev && *ch.account != ripemd {
		s.getAccountObject(*ch.account,false).touched = ch.prev
		if !ch.prevDirty {
			delete(s.accountObjectsDirty, *ch.account)
		}
	}
}

func (ch balanceChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account,false).setBalance(ch.prev)
}

func (ch nonceChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account,false).setNonce(ch.prev)
}

func (ch codeChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account,false).setCode(common.BytesToHash(ch.prevhash), ch.prevcode)
}

func (ch storageChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account,false).setData(ch.key, ch.prevalue)
}

func (ch refundChange) undo(s *AccountDB) {
	s.refund = ch.prev
}