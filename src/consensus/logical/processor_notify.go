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
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"com.tuntun.rocket/node/src/consensus/logical/group_create"
	"com.tuntun.rocket/node/src/consensus/model"
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/notify"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/utility"
)

func (p *Processor) triggerFutureVerifyMsg(hash common.Hash) {
	futures := p.getFutureVerifyMsgs(hash)
	if futures == nil || len(futures) == 0 {
		return
	}
	p.removeFutureVerifyMsgs(hash)
	mtype := "FUTURE_VERIFY"
	for _, msg := range futures {
		tlog := newHashTraceLog(mtype, msg.BH.Hash, msg.SignInfo.GetSignerID())
		tlog.logStart("size %v", len(futures))
		slog := newSlowLog(mtype, 0.5)
		err := p.doVerify(mtype, msg, tlog, newBizLog(mtype), slog)
		if err != nil {
			tlog.logEnd("result=%v", err.Error())
		}
	}

}

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

	p.triggerFutureVerifyMsg(bh.Hash)

	group_create.GroupCreateProcessor.StartCreateGroupPolling()

	p.cleanVerifyContext(bh.Height)

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

func (p *Processor) onMissTxAddSucc(message notify.Message) {
	if !p.Ready() {
		return
	}
	tgam, ok := message.(*notify.TransactionGotAddSuccMessage)
	if !ok {
		stdLogger.Infof("minerTransactionHandler Message assert not ok!")
		return
	}
	transactions := tgam.Transactions
	var txHashes []common.Hashes
	for _, tx := range transactions {
		hashes := common.Hashes{}
		hashes[0] = tx.Hash
		hashes[1] = tx.SubHash
		txHashes = append(txHashes, hashes)

	}
	p.OnMessageNewTransactions(txHashes)
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
