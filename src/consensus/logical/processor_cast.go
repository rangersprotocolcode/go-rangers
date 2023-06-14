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

package logical

import (
	"bytes"
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/base"
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"com.tuntun.rocket/node/src/consensus/model"
	"com.tuntun.rocket/node/src/consensus/net"
	"com.tuntun.rocket/node/src/utility"
	"math/big"

	"com.tuntun.rocket/node/src/middleware/types"
	"runtime/debug"
	"sync"
	"time"

	"com.tuntun.rocket/node/src/middleware"
)

type CastBlockContexts struct {
	blockCtxs    sync.Map //string -> *BlockContext
	reservedVctx sync.Map //blockHash -> *VerifyContext
}

func NewCastBlockContexts() *CastBlockContexts {
	return &CastBlockContexts{
		blockCtxs: sync.Map{},
	}
}

func (bctx *CastBlockContexts) addBlockContext(bc *BlockContext) (add bool) {
	_, load := bctx.blockCtxs.LoadOrStore(bc.MinerID.Gid.GetHexString(), bc)
	return !load
}

func (bctx *CastBlockContexts) getBlockContext(gid groupsig.ID) *BlockContext {
	if v, ok := bctx.blockCtxs.Load(gid.GetHexString()); ok {
		return v.(*BlockContext)
	}
	return nil
}

func (bctx *CastBlockContexts) blockContextSize() int32 {
	size := int32(0)
	bctx.blockCtxs.Range(func(key, value interface{}) bool {
		size++
		return true
	})
	return size
}

func (bctx *CastBlockContexts) removeBlockContexts(gids []groupsig.ID) {
	for _, id := range gids {
		stdLogger.Infof("removeBlockContexts %v", id.ShortS())
		bc := bctx.getBlockContext(id)
		if bc != nil {
			//bc.removeTicker()
			for _, vctx := range bc.SafeGetVerifyContexts() {
				bctx.removeReservedVctx(vctx.blockHash)
			}
			bctx.blockCtxs.Delete(id.GetHexString())
		}
	}
}

func (bctx *CastBlockContexts) forEachBlockContext(f func(bc *BlockContext) bool) {
	bctx.blockCtxs.Range(func(key, value interface{}) bool {
		v := value.(*BlockContext)
		return f(v)
	})
}

func (bctx *CastBlockContexts) removeReservedVctx(hash common.Hash) {
	bctx.reservedVctx.Delete(hash)
}

func (bctx *CastBlockContexts) addReservedVctx(vctx *VerifyContext) bool {
	_, load := bctx.reservedVctx.LoadOrStore(vctx.blockHash, vctx)
	return !load
}

func (bctx *CastBlockContexts) forEachReservedVctx(f func(vctx *VerifyContext) bool) {
	bctx.reservedVctx.Range(func(key, value interface{}) bool {
		v := value.(*VerifyContext)
		return f(v)
	})
}

func (p *Processor) AddBlockContext(bc *BlockContext) bool {
	var add = p.blockContexts.addBlockContext(bc)
	newBizLog("AddBlockContext").log("gid=%v, result=%v\n.", bc.MinerID.Gid.ShortS(), add)
	return add
}

func (p *Processor) GetBlockContext(gid groupsig.ID) *BlockContext {
	return p.blockContexts.getBlockContext(gid)
}

func (p *Processor) triggerCastCheck() {
	p.Ticker.StartAndTriggerRoutine(p.getCastCheckRoutineName())
}

func (p *Processor) CalcVerifyGroupFromCache(preBH *types.BlockHeader, castTime time.Time, height uint64) *groupsig.ID {
	var hash = CalcRandomHash(preBH, castTime)

	selectGroup, err := p.globalGroups.SelectVerifyGroupFromCache(hash, height)
	if err != nil {
		stdLogger.Errorf("SelectNextGroupFromCache height=%v, err: %v", height, err)
		return nil
	}
	return &selectGroup
}

func (p *Processor) CalcVerifyGroupFromChain(preBH *types.BlockHeader, castTime time.Time, height uint64) *groupsig.ID {
	var hash = CalcRandomHash(preBH, castTime)

	selectGroup, err := p.globalGroups.SelectVerifyGroupFromChain(hash, height)
	if err != nil {
		stdLogger.Errorf("SelectNextGroupFromChain height=%v, err:%v", height, err)
		return nil
	}
	return &selectGroup
}

func (p *Processor) spreadGroupBrief(bh *types.BlockHeader, castTime time.Time, height uint64) *net.GroupBrief {
	nextId := p.CalcVerifyGroupFromCache(bh, castTime, height)
	if nextId == nil {
		return nil
	}
	group := p.GetGroup(*nextId)
	g := &net.GroupBrief{
		Gid:    *nextId,
		MemIds: group.GetGroupMembers(),
	}
	return g
}

func (p *Processor) reserveBlock(vctx *VerifyContext) {
	slot := vctx.slot
	bh := slot.BH
	blog := newBizLog("reserveBLock")
	blog.log("height=%v, totalQN=%v, hash=%v, slotStatus=%v", bh.Height, bh.TotalQN, bh.Hash.ShortS(), slot.GetSlotStatus())
	if slot.IsRecovered() {
		vctx.markCastSuccess()
		p.blockContexts.addReservedVctx(vctx)
		if !p.tryBroadcastBlock(vctx) {
			blog.log("reserved, hash=%v", vctx.blockHash)
		}

	}
}

func (p *Processor) tryBroadcastBlock(vctx *VerifyContext) bool {
	if sc := vctx.checkBroadcast(); sc != nil {
		bh := sc.BH
		tlog := newHashTraceLog("tryBroadcastBlock", bh.Hash, p.GetMinerID())
		tlog.log("try broadcast, height=%v, totalQN=%v, 耗时%v", bh.Height, bh.TotalQN, utility.GetTime().Sub(bh.CurTime))

		go p.successNewBlock(vctx, sc)

		p.blockContexts.removeReservedVctx(vctx.blockHash)
		return true
	}
	return false
}

func (p *Processor) successNewBlock(vctx *VerifyContext, slot *SlotContext) {
	defer func() {
		if r := recover(); r != nil {
			common.DefaultLogger.Errorf("error：%v\n", r)
			s := debug.Stack()
			common.DefaultLogger.Errorf(string(s))
		}
	}()

	bh := slot.BH

	blog := newBizLog("successNewBlock")

	if slot.IsFailed() {
		blog.log("slot is failed")
		return
	}
	if vctx.broadCasted() {
		blog.log("block broadCasted!")
		return
	}

	if p.blockOnChain(bh.Hash) {
		blog.log("block already onchain!")
		return
	}

	block := p.MainChain.GenerateBlock(*bh)
	if block == nil {
		blog.log("core.GenerateBlock is nil! won't broadcast block!")
		return
	}

	group := p.GetGroup(groupsig.DeserializeID(bh.GroupId))
	gpk := group.GetGroupPubKey()
	if !slot.VerifyGroupSigns(gpk, vctx.prevBH.Random) {
		blog.log("group pub key local check failed, gpk=%v, hash in slot=%v, hash in bh=%v status=%v.",
			gpk.ShortS(), slot.BH.Hash.ShortS(), bh.Hash.ShortS(), slot.GetSlotStatus())
		return
	}

	r := p.doAddOnChain(block)
	if r != int8(types.AddBlockSucc) {
		slot.setSlotStatus(SS_FAILED)
		return
	}

	tlog := newHashTraceLog("successNewBlock", bh.Hash, p.GetMinerID())
	tlog.log("height=%v, status=%v", bh.Height, vctx.consensusStatus)

	seed := big.NewInt(0).SetBytes(bh.Hash.Bytes()).Uint64()
	index := seed % uint64(group.GetMemberCount())
	id := group.GetMemberID(int(index)).GetBigInt()
	if id.Cmp(p.mi.ID.GetBigInt()) == 0 {
		cbm := &model.ConsensusBlockMessage{
			Block: *block,
		}
		p.NetServer.BroadcastNewBlock(cbm)
		tlog.log("broadcasted height=%v, cost: %v, seed: %d, index: %d", bh.Height, utility.GetTime().Sub(bh.CurTime), seed, index)
	}

	vctx.broadcastSlot = slot
	vctx.markBroadcast()
	slot.setSlotStatus(SS_SUCCESS)

	blog.log("After BroadcastNewBlock hash=%v:%v", bh.Hash.ShortS(), utility.GetTime().Format(TIMESTAMP_LAYOUT))
	return
}

func (p *Processor) sampleBlockHeight(heightLimit uint64, rand []byte, id groupsig.ID) uint64 {
	if heightLimit > 2*model.Param.Epoch {
		heightLimit -= 2 * model.Param.Epoch
	}
	return base.RandFromBytes(rand).DerivedRand(id.Serialize()).ModuloUint64(heightLimit)
}

func (p *Processor) GenProveHashs(heightLimit uint64, rand []byte, ids []groupsig.ID) (proves []common.Hash, root common.Hash) {
	hashs := make([]common.Hash, len(ids))

	blog := newBizLog("GenProveHashs")
	for idx, id := range ids {
		h := p.sampleBlockHeight(heightLimit, rand, id)
		b := p.getNearestBlockByHeight(h)
		hashs[idx] = p.GenVerifyHash(b, id)
		blog.log("sampleHeight for %v is %v, real height is %v, proveHash is %v", id.GetHexString(), h, b.Header.Height, hashs[idx].String())
	}
	proves = hashs

	buf := bytes.Buffer{}
	for _, hash := range hashs {
		buf.Write(hash.Bytes())
	}
	root = base.Data2CommonHash(buf.Bytes())
	buf.Reset()
	return
}

func (p *Processor) blockProposal() {
	worker := p.GetVrfWorker()
	if nil == worker {
		return
	}

	blog := newBizLog("blockProposal")
	top := p.MainChain.TopBlock()
	if worker.getBaseBH().Hash != top.Hash {
		blog.log("vrf baseBH differ from top!")
		return
	}

	if worker.isProposed() || worker.isSuccess() {
		blog.log("vrf worker proposed/success, status %v", worker.getStatus())
		return
	}
	height := worker.castHeight

	totalStake := p.minerReader.GetTotalStake(worker.baseBH.Height, worker.baseBH.StateTree)
	blog.log("totalStake height=%v, stake=%v", height, totalStake)
	start := utility.GetTime()
	pi, qn, err := worker.genProve(start, totalStake)
	if err != nil {
		blog.log("vrf prove not ok! %v", err)
		return
	}

	if worker.timeout() {
		blog.log("vrf worker timeout")
		return
	}
	middleware.PerfLogger.Debugf("after genProve, last: %v, height: %v", utility.GetTime().Sub(start), height)

	gb := p.spreadGroupBrief(top, start, height)
	if gb == nil {
		blog.log("spreadGroupBrief nil, bh=%v, height=%v", top.Hash.ShortS(), height)
		return
	}
	gid := gb.Gid
	middleware.PerfLogger.Debugf("after spreadGroupBrief, last: %v, height: %v", utility.GetTime().Sub(start), height)

	//proveHash, root := p.GenProveHashs(height, worker.getBaseBH().Random, gb.MemIds)

	middleware.PerfLogger.Infof("start cast block, last: %v, height: %v", utility.GetTime().Sub(start), height)
	block := p.MainChain.CastBlock(start, uint64(height), pi.Big(), common.Hash{}, qn, p.GetMinerID().Serialize(), gid.Serialize())
	if block == nil {
		blog.log("MainChain::CastingBlock failed, height=%v", height)
		return
	}
	bh := block.Header
	middleware.PerfLogger.Infof("fin cast block, last: %v, hash: %v, height: %v", utility.GetTime().Sub(start), bh.Hash.String(), bh.Height)

	tlog := newHashTraceLog("CASTBLOCK", bh.Hash, p.GetMinerID())
	blog.log("begin proposal, hash=%v, height=%v, qn=%v,, verifyGroup=%v, pi=%v...", bh.Hash.ShortS(), height, qn, gid.ShortS(), pi.ShortS())
	tlog.logStart("height=%v,qn=%v, preHash=%v, verifyGroup=%v", bh.Height, qn, bh.PreHash.ShortS(), gid.ShortS())

	if bh.Height > 0 && bh.Height == height && bh.PreHash == worker.baseBH.Hash {
		skey := p.mi.SecKey

		var ccm model.ConsensusCastMessage
		ccm.BH = *bh
		ccm.ProveHash = []common.Hash{}

		if signInfo, ok := model.NewSignInfo(p.mi.SecKey, p.mi.ID, &ccm); !ok {
			blog.log("sign fail, id=%v, sk=%v", p.GetMinerID().ShortS(), skey.ShortS())
			return
		} else {
			ccm.SignInfo = signInfo
		}
		tlog.log("cast successfully, SendVerifiedCast, cost: %v, castor=%v, hash=%v, genHash=%v", bh.CurTime.Sub(bh.PreTime).Seconds(), ccm.SignInfo.GetSignerID().ShortS(), bh.Hash.ShortS(), ccm.SignInfo.GetDataHash().ShortS())
		p.NetServer.SendCandidate(&ccm, gb, block.Transactions)

		worker.markProposed()

		middleware.PerfLogger.Infof("fin block, last: %v, hash: %v, height: %v", utility.GetTime().Sub(start), bh.Hash.String(), bh.Height)
	} else {
		blog.log("bh/prehash Error or sign Error, bh=%v, real height=%v. bc.prehash=%v, bh.prehash=%v", height, bh.Height, worker.baseBH.Hash, bh.PreHash)
	}

}
