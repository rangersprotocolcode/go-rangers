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
	"com.tuntun.rangers/node/src/consensus/base"
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/consensus/net"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/utility"
	"time"

	"com.tuntun.rangers/node/src/middleware"
)

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
	bh, ok := p.MainChain.CastBlock(start, height, pi.Big(), common.Hash{}, qn, p.GetMinerID().Serialize(), gid.Serialize())
	if !ok {
		blog.log("MainChain::CastingBlock failed, height=%v", height)
		return
	}
	middleware.PerfLogger.Infof("fin cast block, last: %v, hash: %v, height: %v", utility.GetTime().Sub(start), bh.Hash.String(), bh.Height)

	tlog := newHashTraceLog("CASTBLOCK", bh.Hash, p.GetMinerID())
	blog.log("begin proposal, hash=%v, height=%v, qn=%v,, verifyGroup=%v, pi=%v...", bh.Hash.ShortS(), height, qn, gid.ShortS(), pi.ShortS())
	tlog.logStart("height=%v,qn=%v, preHash=%v, verifyGroup=%v", bh.Height, qn, bh.PreHash.ShortS(), gid.ShortS())

	if bh.Height > 0 && bh.Height == height && bh.PreHash == worker.baseBH.Hash {
		skey := p.mi.SecKey

		var ccm model.ConsensusCastMessage
		ccm.BH = bh
		ccm.ProveHash = []common.Hash{}

		if signInfo, ok := model.NewSignInfo(p.mi.SecKey, p.mi.ID, &ccm); !ok {
			blog.log("sign fail, id=%v, sk=%v", p.GetMinerID().ShortS(), skey.ShortS())
			return
		} else {
			ccm.SignInfo = signInfo
		}
		tlog.log("cast successfully, SendVerifiedCast, cost: %v, castor=%v, hash=%v, genHash=%v", bh.CurTime.Sub(bh.PreTime).Seconds(), ccm.SignInfo.GetSignerID().ShortS(), bh.Hash.ShortS(), ccm.SignInfo.GetDataHash().ShortS())
		p.NetServer.SendCandidate(&ccm)

		worker.markProposed()
		middleware.PerfLogger.Warnf("fin new signInfo, %s - %s", ccm.SignInfo.GetDataHash().String(), ccm.BH.Hash.String())
		middleware.PerfLogger.Infof("fin block, last: %v, hash: %v, height: %v", utility.GetTime().Sub(start), bh.Hash.String(), bh.Height)
	} else {
		blog.log("bh/prehash Error or sign Error, bh=%v, real height=%v. bc.prehash=%v, bh.prehash=%v", height, bh.Height, worker.baseBH.Hash, bh.PreHash)
	}

}
