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
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/utility"
	"sync"
	"sync/atomic"
	"time"
)

const (
	CBCS_IDLE int32 = iota
	CBCS_CASTING
	CBCS_BLOCKED
	CBCS_BROADCAST
	CBCS_TIMEOUT
)

type CAST_BLOCK_MESSAGE_RESULT int8

const (
	CBMR_PIECE_NORMAL CAST_BLOCK_MESSAGE_RESULT = iota
	CBMR_PIECE_LOSINGTRANS
	CBMR_THRESHOLD_SUCCESS
	CBMR_THRESHOLD_FAILED
	CBMR_IGNORE_REPEAT

	CBMR_BH_HASH_DIFF
)

type VerifyContext struct {
	prevBH          *types.BlockHeader
	blockHash       common.Hash
	createTime      time.Time
	expireTime      time.Time
	consensusStatus int32
	slot            *SlotContext
	broadcastSlot   *SlotContext

	blockCtx  *BlockContext
	signedNum int32
	lock      sync.RWMutex
}

func (vc *VerifyContext) castSuccess() bool {
	return atomic.LoadInt32(&vc.consensusStatus) == CBCS_BLOCKED
}
func (vc *VerifyContext) broadCasted() bool {
	return atomic.LoadInt32(&vc.consensusStatus) == CBCS_BROADCAST
}

func (vc *VerifyContext) markBroadcast() bool {
	return atomic.CompareAndSwapInt32(&vc.consensusStatus, CBCS_BLOCKED, CBCS_BROADCAST)
}

func (vc *VerifyContext) checkBroadcast() *SlotContext {
	if !vc.castSuccess() {
		//blog.log("not success st=%v", vc.consensusStatus)
		return nil
	}

	now := utility.GetTime()
	if now.Sub(vc.createTime).Seconds() < float64(model.Param.MaxWaitBlockTime) {
		//blog.log("not the time, creatTime %v, now %v, since %v", vc.createTime, utility.GetTime(), time.Since(vc.createTime).String())
		return nil
	}

	return vc.slot
}
