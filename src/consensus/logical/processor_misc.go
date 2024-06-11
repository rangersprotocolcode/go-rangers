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
	"com.tuntun.rangers/node/src/consensus/base"
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/middleware/types"
	"fmt"
)

func (p *Processor) Start() bool {
	p.prepareMiner()

	p.Ticker.RegisterRoutine(p.getCastCheckRoutineName(), p.checkSelfCastRoutine, common.CastingCheckInterval)

	p.Ticker.RegisterRoutine(p.getUpdateGlobalGroupsRoutineName(), p.updateGlobalGroups, 60*1000)
	p.Ticker.StartTickerRoutine(p.getUpdateGlobalGroupsRoutineName(), false)

	p.Ticker.RegisterRoutine(p.getReleaseRoutineName(), p.releaseRoutine, 600)
	p.Ticker.StartTickerRoutine(p.getReleaseRoutineName(), false)

	p.triggerCastCheck()

	p.ready = true
	return true
}

func (p *Processor) Stop() {
	return
}

func (p *Processor) prepareMiner() {
	topHeight := p.MainChain.TopBlock().Height

	stdLogger.Infof("prepareMiner get groups from groupchain")
	iterator := p.GroupChain.Iterator()
	groups := make([]*model.GroupInfo, 0)
	for coreGroup := iterator.Current(); coreGroup != nil; coreGroup = iterator.MovePre() {
		stdLogger.Infof("get group from core, id=%+v", coreGroup.Header)
		if coreGroup.Id == nil || len(coreGroup.Id) == 0 {
			continue
		}

		sgi := model.ConvertToGroupInfo(coreGroup)
		if sgi.NeedDismiss(topHeight) {
			continue
		}

		groups = append(groups, sgi)
		stdLogger.Infof("load group=%v, beginHeight=%v, topHeight=%v\n", sgi.GroupID.ShortS(), sgi.GetGroupHeader().WorkHeight, topHeight)
		if sgi.MemExist(p.GetMinerID()) {
			jg := p.belongGroups.GetJoinedGroupInfo(sgi.GroupID)
			if jg == nil {
				stdLogger.Infof("prepareMiner get join group fail, gid=%v\n", sgi.GroupID.ShortS())
			} else {
				p.belongGroups.JoinGroup(jg, p.mi.ID)
			}
		}
	}

	stdLogger.Infof("prepare %d groups", len(groups))
	for i := len(groups) - 1; i >= 0; i-- {
		p.acceptGroup(groups[i])
	}
	stdLogger.Infof("prepare finished")
}

func (p *Processor) Ready() bool {
	return p.ready
}

func (p *Processor) GetCastQualifiedGroups(height uint64) []*model.GroupInfo {
	return p.globalGroups.GetEffectiveGroups(height)
}

func (p *Processor) Finalize() {
	if p.belongGroups != nil {
		p.belongGroups.Close()
	}
}

func (p *Processor) GetVrfWorker() *vrfWorker {
	if v := p.vrf.Load(); v != nil {
		return v.(*vrfWorker)
	}
	return nil
}

func (p *Processor) setVrfWorker(vrf *vrfWorker) {
	p.vrf.Store(vrf)
}

func (p *Processor) GetSelfMinerDO(pre *types.BlockHeader) *model.SelfMinerInfo {
	md := p.minerReader.GetProposeMiner(p.GetMinerID(), pre.StateTree)
	if md != nil {
		p.mi.MinerInfo = *md
	}
	return p.mi
}

func (p *Processor) canProposalAt(pre *types.BlockHeader) bool {
	miner := p.minerReader.GetProposeMiner(p.GetMinerID(), pre.StateTree)
	if miner == nil {
		//		stdLogger.Errorf("get nil proposeMiner:%s", p.GetMinerID().String())
		return false
	}

	return miner.CanCastAt(pre.Height + 1)
}

func (p *Processor) GetJoinedWorkGroupNums() (work, avail int) {
	h := p.MainChain.TopBlock().Height
	groups := p.globalGroups.GetAvailableGroups(h)
	for _, g := range groups {
		if !g.MemExist(p.GetMinerID()) {
			continue
		}
		if g.IsEffective(h) {
			work++
		}
		avail++
	}
	return
}

func (p *Processor) GenVerifyHash(b *types.Block, id groupsig.ID) common.Hash {
	buf, err := types.MarshalBlock(b)
	if err != nil {
		panic(fmt.Sprintf("marshal block error, hash=%v, err=%v", b.Header.Hash.ShortS(), err))
	}
	//header := &b.Header
	//log.Printf("GenVerifyHash aaa bufHash=%v, buf %v", base.Data2CommonHash(buf).ShortS(), buf)
	//log.Printf("GenVerifyHash aaa headerHash=%v, genHash=%v", b.Header.Hash.ShortS(), b.Header.GenHash().ShortS())

	//headBuf, _ := msgpack.Marshal(header)
	//log.Printf("GenVerifyHash aaa headerBufHash=%v, headerBuf=%v", base.Data2CommonHash(headBuf).ShortS(), headBuf)

	//log.Printf("GenVerifyHash height:%v,id:%v,%v, bbbbbuf %v", b.Header.Height,id.ShortS(), b.Transactions == nil, buf)
	//log.Printf("GenVerifyHash height:%v,id:%v,bbbbbuf ids %v", b.Header.Height,id.ShortS(),id.Serialize())
	buf = append(buf, id.Serialize()...)
	//log.Printf("GenVerifyHash height:%v,id:%v,bbbbbuf after %v", b.Header.Height,id.ShortS(),buf)
	h := base.Data2CommonHash(buf)
	//log.Printf("GenVerifyHash height:%v,id:%v,bh:%v,vh:%v", b.Header.Height,id.ShortS(),b.Header.Hash.ShortS(), h.ShortS())
	return h
}

func (p *Processor) GetJoinGroupInfo(gid string) *model.JoinedGroupInfo {
	var id groupsig.ID
	id.SetHexString(gid)
	jg := p.belongGroups.GetJoinedGroupInfo(id)
	return jg
}

func (p *Processor) GetCastQualifiedGroupsFromChain(height uint64) []*types.Group {
	return p.globalGroups.GetCastQualifiedGroupFromChains(height)
}
