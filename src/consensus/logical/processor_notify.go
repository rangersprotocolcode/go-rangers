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
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/consensus/logical/group_create"
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/middleware"
	"com.tuntun.rangers/node/src/middleware/notify"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/utility"
)

func (p *Processor) onBlockAddSuccess(message notify.Message) {
	if !p.Ready() {
		return
	}
	block := message.GetData().(types.Block)
	bh := block.Header

	p.lock.Lock()
	defer p.lock.Unlock()

	gid := groupsig.DeserializeID(bh.GroupId)
	if p.belongGroups.BelongGroup(gid) {
		bc := p.GetBlockContext(gid)
		if bc != nil {
			bc.AddCastedHeight(bh.Hash, bh.PreHash)
			bc.updateSignedMaxQN(bh.TotalQN)
			vctx := bc.GetVerifyContextByHash(bh.Hash)
			if vctx != nil && vctx.prevBH.Hash == bh.PreHash {
				vctx.markBroadcast()
			}
		}
		p.removeVerifyMsgCache(bh.Hash)
	}

	worker := p.GetVrfWorker()
	if nil != worker && worker.castHeight <= bh.Height {
		p.setVrfWorker(nil)
	}

	group_create.GroupCreateProcessor.StartCreateGroupPolling()

	middleware.PerfLogger.Infof("OnBlockAddSuccess. cost: %v, Hash: %v, height: %v", utility.GetTime().Sub(bh.CurTime), bh.Hash.String(), bh.Height)
	if p.isTriggerCastImmediately() {
		p.triggerCastCheck()
	}
}

func (p *Processor) isTriggerCastImmediately() bool {
	return false
	//return p.MainChain.GetTransactionPool().TxNum() > 200
}

func (p *Processor) onGroupAddSuccess(message notify.Message) {
	group := message.GetData().(types.Group)
	stdLogger.Infof("groupAddEventHandler receive message, groupId=%v, workheight=%v\n", groupsig.DeserializeID(group.Id).GetHexString(), group.Header.WorkHeight)
	if group.Id == nil || len(group.Id) == 0 {
		return
	}
	sgi := model.ConvertToGroupInfo(&group)
	p.acceptGroup(sgi)

	group_create.GroupCreateProcessor.OnGroupAddSuccess(sgi)
}

func (p *Processor) onGroupAccept(message notify.Message) {
	group := message.GetData().(types.Group)
	stdLogger.Infof("groupAcceptHandler receive message, groupId=%v, workheight=%v\n", groupsig.DeserializeID(group.Id).GetHexString(), group.Header.WorkHeight)
	if group.Id == nil || len(group.Id) == 0 {
		return
	}
	sgi := model.ConvertToGroupInfo(&group)
	p.acceptGroup(sgi)
}

func (p *Processor) acceptGroup(staticGroup *model.GroupInfo) {
	add := p.globalGroups.AddGroupInfo(staticGroup)
	blog := newBizLog("acceptGroup")
	blog.debug("Add to Global static groups, result=%v, groups=%v.", add, p.globalGroups.GroupSize())
	if staticGroup.MemExist(p.GetMinerID()) {
		p.prepareForCast(staticGroup)
	}
}

func (p *Processor) prepareForCast(sgi *model.GroupInfo) {
	p.NetServer.JoinGroupNet(sgi.GroupID.GetHexString())

	bc := NewBlockContext(p, sgi)

	stdLogger.Debugf("prepareForCast current ID %v", p.GetMinerID().ShortS())

	b := p.AddBlockContext(bc)
	stdLogger.Infof("(proc:%v) prepareForCast Add BlockContext result = %v, bc_size=%v", p.getPrefix(), b, p.blockContexts.blockContextSize())
}
