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
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/consensus/logical/group_create"
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/utility"
	"time"
)

func (p *Processor) getCastCheckRoutineName() string {
	return "self_cast_check_" + p.getPrefix()
}

func (p *Processor) getReleaseRoutineName() string {
	return "release_routine_" + p.getPrefix()
}

func (p *Processor) checkSelfCastRoutine() bool {
	if !p.Ready() {
		return false
	}

	blog := newBizLog("checkSelfCastRoutine")
	top := p.MainChain.TopBlock()

	delta := utility.GetTime().Sub(top.CurTime)
	if delta.Seconds() < float64(common.GetCastingInterval()/1000) {
		return false
	}

	castHeight := top.Height + 1
	if !p.canProposalAt(top) {
		return false
	}
	blog.log("proposal at %d", castHeight)

	p.lock.Lock()
	defer p.lock.Unlock()

	worker := p.GetVrfWorker()
	if worker == nil {
		blog.log("castHeight=%v, worker nil ", castHeight)
	} else if worker.workingOn(top, castHeight) {
		blog.log("already working on that block height=%v, status=%v", castHeight, worker.getStatus())
		return false
	} else {
		blog.log("castHeight=%v, worker not nil, worker cast height: %d, expired: %s, baseHash: %s  ", castHeight, worker.castHeight, worker.expire.String(), worker.baseBH.Hash.String())

	}

	var expireTime time.Time
	if castHeight == 1 {
		expireTime = utility.GetTime().Add(24 * time.Hour)
	} else {
		if worker == nil {
			expireTime = utility.GetTime().Add(time.Second * time.Duration(uint64(model.Param.MaxGroupCastTime)))
		} else {
			expireTime = worker.expire.Add(time.Second * time.Duration(uint64(model.Param.MaxGroupCastTime)))
		}
	}

	blog.log("topHeight=%v, topHash=%v, topCurTime=%v, castHeight=%v, expireTime=%v,current time:%v", top.Height, top.Hash.ShortS(), top.CurTime, castHeight, expireTime, utility.GetTime())
	worker = newVRFWorker(p.GetSelfMinerDO(top), top, castHeight, expireTime)
	p.setVrfWorker(worker)
	p.blockProposal()
	return true
}

func (p *Processor) getUpdateGlobalGroupsRoutineName() string {
	return "update_global_groups_routine_" + p.getPrefix()
}

func (p *Processor) updateGlobalGroups() bool {
	top := p.MainChain.Height()
	iter := p.GroupChain.Iterator()
	for g := iter.Current(); g != nil && !IsGroupDissmisedAt(g.Header, top); g = iter.MovePre() {
		gid := groupsig.DeserializeID(g.Id)
		if g, _ := p.globalGroups.GetGroupFromCache(gid); g != nil {
			continue
		}
		sgi := model.ConvertToGroupInfo(g)
		stdLogger.Debugf("updateGlobalGroups:gid=%v, workHeight=%v, topHeight=%v", gid.ShortS(), g.Header.WorkHeight, top)
		p.acceptGroup(sgi)
	}
	return true
}

func (p *Processor) releaseRoutine() bool {
	topHeight := p.MainChain.TopBlock().Height
	if topHeight <= model.Param.CreateGroupInterval {
		return true
	}

	group_create.GroupCreateProcessor.ReleaseGroups(topHeight)

	return true
}
