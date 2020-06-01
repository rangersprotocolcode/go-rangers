package logical

import (
	"time"
	"x/src/common"
	"x/src/consensus/groupsig"
	"x/src/consensus/logical/group_create"
	"x/src/consensus/model"
	"x/src/utility"
)

func (p *Processor) getCastCheckRoutineName() string {
	return "self_cast_check_" + p.getPrefix()
}

func (p *Processor) getBroadcastRoutineName() string {
	return "broadcast_" + p.getPrefix()
}

func (p *Processor) getReleaseRoutineName() string {
	return "release_routine_" + p.getPrefix()
}

//检查是否当前组铸块
func (p *Processor) checkSelfCastRoutine() bool {
	if !p.Ready() {
		return false
	}

	blog := newBizLog("checkSelfCastRoutine")
	top := p.MainChain.TopBlock()

	delta := utility.GetTime().Sub(top.CurTime)
	if delta.Seconds() < common.CastingInterval/1000 {
		blog.log("time cost %vs from chain casting last block,less than %vs,do not proposal.last block cast time:%v ", time.Since(top.CurTime).Seconds(), common.CastingInterval/1000, top.CurTime)
		return false
	}

	castHeight := top.Height + 1
	if !p.canProposalAt(top) {
		blog.log("can not proposal at %d", castHeight)
		return false
	}

	p.lock.Lock()
	defer p.lock.Unlock()

	worker := p.GetVrfWorker()
	if worker != nil && worker.workingOn(top, castHeight) {
		blog.log("already working on that block height=%v, status=%v", castHeight, worker.getStatus())
		return false
	}

	var expireTime time.Time
	if worker == nil {
		expireTime = utility.GetTime().Add(time.Second * time.Duration(uint64(model.Param.MaxGroupCastTime)))
	} else {
		expireTime = worker.expire.Add(time.Second * time.Duration(uint64(model.Param.MaxGroupCastTime)))
	}
	blog.log("topHeight=%v, topHash=%v, topCurTime=%v, castHeight=%v, expireTime=%v,current time:%v", top.Height, top.Hash.ShortS(), top.CurTime, castHeight, expireTime, utility.GetTime())
	worker = newVRFWorker(p.GetSelfMinerDO(top), top, castHeight, expireTime)
	p.setVrfWorker(worker)
	p.blockProposal()
	return true
}

func (p *Processor) broadcastRoutine() bool {
	p.blockContexts.forEachReservedVctx(func(vctx *VerifyContext) bool {
		p.tryBroadcastBlock(vctx)
		return true
	})
	return true
}

func (p *Processor) releaseRoutine() bool {
	topHeight := p.MainChain.TopBlock().Height
	if topHeight <= model.Param.CreateGroupInterval {
		return true
	}

	//删除verifyContext
	p.cleanVerifyContext(topHeight)
	blog := newBizLog("releaseRoutine")

	ids := group_create.GroupCreateProcessor.ReleaseGroups(topHeight)
	if len(ids) > 0 {
		p.blockContexts.removeBlockContexts(ids)
	}

	//释放futureVerifyMsg
	p.futureVerifyMsgs.forEach(func(key common.Hash, arr []interface{}) bool {
		for _, msg := range arr {
			b := msg.(*model.ConsensusCastMessage)
			if b.BH.Height+200 < topHeight {
				blog.debug("remove future verify msg, hash=%v", key.String())
				p.removeFutureVerifyMsgs(key)
				break
			}
		}
		return true
	})

	for _, h := range p.verifyMsgCaches.Keys() {
		hash := h.(common.Hash)
		cache := p.getVerifyMsgCache(hash)
		if cache != nil && cache.expired() {
			blog.debug("remove verify cache msg, hash=%v", hash.ShortS())
			p.removeVerifyMsgCache(hash)
		}
	}

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
