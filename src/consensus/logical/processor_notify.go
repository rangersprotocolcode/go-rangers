package logical

import (
	"x/src/common"
	"x/src/consensus/groupsig"
	"x/src/consensus/model"
	"x/src/middleware/notify"
	"x/src/middleware/types"
)

func (p *Processor) triggerFutureVerifyMsg(hash common.Hash) {
	futures := p.getFutureVerifyMsgs(hash)
	if futures == nil || len(futures) == 0 {
		return
	}
	p.removeFutureVerifyMsgs(hash)
	mtype := "FUTURE_VERIFY"
	for _, msg := range futures {
		tlog := newHashTraceLog(mtype, msg.BH.Hash, msg.SI.GetID())
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

	tlog := newMsgTraceLog("OnBlockAddSuccess", bh.Hash.ShortS(), "")
	tlog.log("preHash=%v, height=%v", bh.PreHash.ShortS(), bh.Height)

	gid := groupsig.DeserializeId(bh.GroupId)
	if p.IsMinerGroup(gid) {
		bc := p.GetBlockContext(gid)
		if bc != nil {
			bc.AddCastedHeight(bh.Height, bh.PreHash)
			vctx := bc.GetVerifyContextByHeight(bh.Height)
			if vctx != nil && vctx.prevBH.Hash == bh.PreHash {
				//如果本地没有广播准备，说明是其他节点广播过来的块，则标记为已广播
				vctx.markBroadcast()
			}
		}
		p.removeVerifyMsgCache(bh.Hash)
	}

	vrf := p.GetVrfWorker()
	if vrf != nil && vrf.baseBH.Hash == bh.PreHash && vrf.castHeight == bh.Height {
		vrf.markSuccess()
	}

	//p.triggerFutureBlockMsg(bh)
	p.triggerFutureVerifyMsg(bh.Hash)

	p.groupManager.CreateNextGroupRoutine()

	p.cleanVerifyContext(bh.Height)

	if p.isTriggerCastImmediately() {
		p.triggerCastCheck()
	}
}

// todo: 触发条件可以更丰富，更动态
func (p *Processor) isTriggerCastImmediately() bool {
	return p.MainChain.GetTransactionPool().TxNum() > 500
}

func (p *Processor) onGroupAddSuccess(message notify.Message) {
	group := message.GetData().(types.Group)
	stdLogger.Infof("groupAddEventHandler receive message, groupId=%v, workheight=%v\n", groupsig.DeserializeId(group.Id).GetHexString(), group.Header.WorkHeight)
	if group.Id == nil || len(group.Id) == 0 {
		return
	}
	sgi := NewSGIFromCoreGroup(&group)
	p.acceptGroup(sgi)

	p.groupManager.onGroupAddSuccess(sgi)
	p.joiningGroups.Clean(sgi.GInfo.GroupHash())
	p.globalGroups.removeInitedGroup(sgi.GInfo.GroupHash())

}

func (p *Processor) onNewBlockReceive(message notify.Message) {
	if !p.Ready() {
		return
	}
	msg := &model.ConsensusBlockMessage{
		Block: message.GetData().(types.Block),
	}
	p.OnMessageBlock(msg)
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
