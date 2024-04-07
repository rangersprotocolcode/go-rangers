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
	"com.tuntun.rangers/node/src/consensus/model"
	"sync"
	"sync/atomic"
)

type castedBlock struct {
	hash    common.Hash
	preHash common.Hash
}

type BlockContext struct {
	Version      uint
	GroupMembers int
	Proc         *Processor
	MinerID      *model.GroupMinerID

	signedMaxQN uint64

	vctxs map[common.Hash]*VerifyContext //height -> *VerifyContext

	recentCasted [100]*castedBlock
	curr         int

	lock sync.RWMutex
}

func NewBlockContext(p *Processor, sgi *model.GroupInfo) *BlockContext {
	bc := &BlockContext{
		Proc:         p,
		MinerID:      model.NewGroupMinerID(sgi.GroupID, p.GetMinerID()),
		GroupMembers: sgi.GetMemberCount(),
		vctxs:        make(map[common.Hash]*VerifyContext),
		Version:      model.CONSENSUS_VERSION,
		curr:         0,
	}

	return bc
}

func (bc *BlockContext) threshold() int {
	return model.Param.GetGroupK(bc.GroupMembers)
}

func (bc *BlockContext) SafeGetVerifyContexts() []*VerifyContext {
	bc.lock.RLock()
	defer bc.lock.RUnlock()
	vctx := make([]*VerifyContext, len(bc.vctxs))
	i := 0
	for _, vc := range bc.vctxs {
		vctx[i] = vc
		i++
	}
	return vctx
}

func (bc *BlockContext) getSignedMaxQN() uint64 {
	return atomic.LoadUint64(&bc.signedMaxQN)
}
