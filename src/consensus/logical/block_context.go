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
	"bytes"
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/middleware/types"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
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

func (bc *BlockContext) GetVerifyContextByHash(hash common.Hash) *VerifyContext {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	return bc.getVctxByHash(hash)
}

func (bc *BlockContext) getVctxByHash(hash common.Hash) *VerifyContext {
	if v, ok := bc.vctxs[hash]; ok {
		return v
	}
	return nil
}

func (bc *BlockContext) replaceVerifyCtx(hash common.Hash, expireTime time.Time, preBH *types.BlockHeader) *VerifyContext {
	vctx := newVerifyContext(bc, expireTime, preBH, hash)
	bc.vctxs[hash] = vctx
	return vctx
}

func (bc *BlockContext) getOrNewVctx(expireTime time.Time, bh, preBH *types.BlockHeader) *VerifyContext {
	var vctx *VerifyContext
	blog := newBizLog("getOrNewVctx")
	hash := bh.Hash

	if vctx = bc.getVctxByHash(hash); vctx == nil {
		vctx = newVerifyContext(bc, expireTime, preBH, hash)
		bc.vctxs[hash] = vctx
		blog.log("add vctx expire %v,hash:%s", expireTime, hash.String())
	} else {
		if vctx.prevBH.Hash != preBH.Hash {
			blog.log("vctx pre hash diff, hash=%v, existHash=%v, commingHash=%v", hash, vctx.prevBH.Hash.ShortS(), preBH.Hash.ShortS())
			preOld := bc.Proc.getBlockHeaderByHash(vctx.prevBH.Hash)

			if preOld == nil {
				vctx = bc.replaceVerifyCtx(hash, expireTime, preBH)
				return vctx
			}
			preNew := bc.Proc.getBlockHeaderByHash(preBH.Hash)
			if preNew == nil {
				return nil
			}

			if preOld.Height < preNew.Height {
				vctx = bc.replaceVerifyCtx(hash, expireTime, preNew)
			}
		} else {
			if bh.Height == 1 && expireTime.After(vctx.expireTime) {
				vctx.expireTime = expireTime
			}
			blog.log("get exist vctx hash %s, expire %v", hash.String(), vctx.expireTime)
		}
	}
	return vctx
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

func (bc *BlockContext) GetOrNewVerifyContext(bh *types.BlockHeader, preBH *types.BlockHeader) *VerifyContext {
	deltaHeightByTime := DeltaHeightByTime(bh, preBH)

	expireTime := GetCastExpireTime(preBH.CurTime, deltaHeightByTime, bh.Height)

	bc.lock.Lock()
	defer bc.lock.Unlock()

	vctx := bc.getOrNewVctx(expireTime, bh, preBH)
	return vctx
}

func (bc *BlockContext) CleanVerifyContext(height uint64) {
	newCtxs := make(map[common.Hash]*VerifyContext, 0)
	for _, ctx := range bc.SafeGetVerifyContexts() {
		bRemove := ctx.shouldRemove(height)
		if !bRemove {
			newCtxs[ctx.blockHash] = ctx
		} else {
			//ctx.Clear()
			bc.Proc.blockContexts.removeReservedVctx(ctx.blockHash)
			stdLogger.Debugf("CleanVerifyContext: ctx.castHash=%v, ctx.prevHash=%v, signedNum=%v\n", ctx.blockHash, ctx.prevBH.Hash.ShortS(), ctx.signedNum)
		}
	}

	bc.lock.Lock()
	defer bc.lock.Unlock()
	bc.vctxs = newCtxs
}

func (bc *BlockContext) IsHashCasted(hash, pre common.Hash) (cb *castedBlock, casted bool) {
	bc.lock.RLock()
	defer bc.lock.RUnlock()
	for i, h := range bc.recentCasted {
		if h != nil && bytes.Compare(h.hash.Bytes(), hash.Bytes()) == 0 {
			cb = bc.recentCasted[i]
			casted = h.preHash == pre
			return
		}
	}
	return
}

func (bc *BlockContext) AddCastedHeight(hash, pre common.Hash) {
	if cb, same := bc.IsHashCasted(hash, pre); same {
		return
	} else {
		bc.lock.Lock()
		defer bc.lock.Unlock()

		if cb != nil {
			cb.preHash = pre
		} else {
			bc.recentCasted[bc.curr] = &castedBlock{hash: hash, preHash: pre}
			bc.curr = (bc.curr + 1) % len(bc.recentCasted)
		}
	}
}

func (bc *BlockContext) CheckQN(bh *types.BlockHeader) error {
	if bc.hasSignedBiggerQN(bh.TotalQN) {
		return fmt.Errorf("已签过更高qn块%v,本块qn%v", bc.getSignedMaxQN(), bh.TotalQN)
	}

	return nil
}

func (bc *BlockContext) getSignedMaxQN() uint64 {
	return atomic.LoadUint64(&bc.signedMaxQN)
}

func (bc *BlockContext) hasSignedBiggerQN(totalQN uint64) bool {
	return bc.getSignedMaxQN() > totalQN
}

func (bc *BlockContext) updateSignedMaxQN(totalQN uint64) bool {
	if bc.getSignedMaxQN() < totalQN {
		atomic.StoreUint64(&bc.signedMaxQN, totalQN)
		return true
	}
	return false
}
