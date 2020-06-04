package logical

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"com.tuntun.rocket/node/src/consensus/logical/group_create"
	"com.tuntun.rocket/node/src/consensus/model"
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/notify"
	"com.tuntun.rocket/node/src/middleware/types"
	"time"
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

//func (p *Processor) triggerFutureBlockMsg(preBH *types.BlockHeader) {
//	futureMsgs := p.getFutureBlockMsgs(preBH.Hash)
//	if futureMsgs == nil || len(futureMsgs) == 0 {
//		return
//	}
//	log.Printf("handle future blocks, size=%v\n", len(futureMsgs))
//	for _, msg := range futureMsgs {
//		tbh := msg.Block.Header
//		tlog := newHashTraceLog("OMB-FUTRUE", tbh.Hash, groupsig.DeserializeId(tbh.Castor))
//		tlog.log( "%v", "trigger cached future block")
//		p.receiveBlock(&msg.Block, preBH)
//	}
//	p.removeFutureBlockMsgs(preBH.Hash)
//}

func (p *Processor) onBlockAddSuccess(message notify.Message) {
	if !p.Ready() {
		return
	}
	block := message.GetData().(types.Block)
	bh := block.Header

	//tlog := newMsgTraceLog("OnBlockAddSuccess", bh.Hash.ShortS(), "")
	//tlog.log("preHash=%v, height=%v", bh.PreHash.ShortS(), bh.Height)

	gid := groupsig.DeserializeID(bh.GroupId)
	if p.belongGroups.BelongGroup(gid) {
		bc := p.GetBlockContext(gid)
		if bc != nil {
			bc.AddCastedHeight(bh.Hash, bh.PreHash)
			bc.updateSignedMaxQN(bh.TotalQN)
			vctx := bc.GetVerifyContextByHash(bh.Hash)
			if vctx != nil && vctx.prevBH.Hash == bh.PreHash {
				//如果本地没有广播准备，说明是其他节点广播过来的块，则标记为已广播
				vctx.markBroadcast()
			}
		}
		p.removeVerifyMsgCache(bh.Hash)
	}
	p.setVrfWorker(nil)

	//p.triggerFutureBlockMsg(bh)
	p.triggerFutureVerifyMsg(bh.Hash)

	group_create.GroupCreateProcessor.StartCreateGroupPolling()

	p.cleanVerifyContext(bh.Height)

	middleware.PerfLogger.Infof("OnBlockAddSuccess. cost: %v, Hash: %v, height: %v", time.Since(bh.CurTime), bh.Hash.String(), bh.Height)
	if p.isTriggerCastImmediately() {
		p.triggerCastCheck()
	}
}

// todo: 触发条件可以更丰富，更动态
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
