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
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package logical

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus/access"
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/consensus/net"
	"com.tuntun.rangers/node/src/core"
	"com.tuntun.rangers/node/src/middleware/log"
	"com.tuntun.rangers/node/src/middleware/types"
	"fmt"
	"sync"
)

type Error struct {
	cause    error
	task     string
	round    int
	victim   string
	culprits []string
}

func NewError(err error, task string, round int, victim string, culprits []string) *Error {
	return &Error{cause: err, task: task, round: round, victim: victim, culprits: culprits}
}

func (err *Error) Error() string {
	if err == nil || err.cause == nil {
		return "Error is nil"
	}
	if err.culprits != nil && len(err.culprits) > 0 {
		return fmt.Sprintf("task %s, party %v, round %d, culprits %s: %s",
			err.task, err.victim, err.round, err.culprits, err.cause.Error())
	}
	return fmt.Sprintf("task %s, party %v, round %d: %s",
		err.task, err.victim, err.round, err.cause.Error())
}

func (err *Error) Unwrap() error { return err.cause }

func (err *Error) Cause() error { return err.cause }

type Round interface {
	Start() *Error
	Close()
	Update(msg model.ConsensusMessage) *Error
	RoundNumber() int
	CanAccept(msg model.ConsensusMessage) int
	CanProceed() bool
	NextRound() Round
}

type (
	baseRound struct {
		processed      map[string]byte
		futureMessages map[string]model.ConsensusMessage

		number       int
		canProcessed bool

		errChan chan error
		lock    sync.Mutex
		started bool

		logger log.Logger

		partyId string
	}

	round1 struct {
		*baseRound
		mi           groupsig.ID
		belongGroups *access.JoinedGroupStorage
		globalGroups *access.GroupAccessor
		minerReader  *access.MinerPoolReader
		blockchain   core.BlockChain
		netServer    net.NetworkServer

		changedId chan string

		lostTxs   map[common.Hashes]byte
		preBH, bh *types.BlockHeader
		ccm       *model.ConsensusCastMessage
		group     *model.GroupInfo

		// if send block by myself
		isSend       bool
		blockExisted bool
	}
	round2 struct {
		*round1

		gSignGenerator *model.GroupSignGenerator
		rSignGenerator *model.GroupSignGenerator
	}
	round3 struct {
		*round2
	}
)

func (round *baseRound) RoundNumber() int {
	return round.number
}

func (round *baseRound) CanProceed() bool {
	return round.canProcessed
}
