package logical

import (
	"x/src/common"
	"x/src/consensus/groupsig"
	"x/src/consensus/model"
	"fmt"
	"x/src/middleware/types"
	"time"

	"x/src/middleware"
)

func (p *Processor) genCastGroupSummary(bh *types.BlockHeader) *model.CastGroupSummary {
	var gid groupsig.ID
	if err := gid.Deserialize(bh.GroupId); err != nil {
		return nil
	}
	var castor groupsig.ID
	if err := castor.Deserialize(bh.Castor); err != nil {
		return nil
	}
	cgs := &model.CastGroupSummary{
		PreHash:     bh.Hash,
		PreTime:     bh.PreTime,
		BlockHeight: bh.Height,
		GroupID:     gid,
		Castor:      castor,
	}
	cgs.CastorPos = p.getMinerPos(cgs.GroupID, cgs.Castor)
	return cgs
}

func (p *Processor) thresholdPieceVerify(mtype string, sender string, gid groupsig.ID, vctx *VerifyContext, slot *SlotContext, traceLog *msgTraceLog) {
	blog := newBizLog("thresholdPieceVerify")
	bh := slot.BH
	if vctx.castSuccess() {
		blog.debug("already cast success, height=%v", bh.Height)
		return
	}

	p.reserveBlock(vctx, slot)

}

func (p *Processor) normalPieceVerify(mtype string, sender string, gid groupsig.ID, vctx *VerifyContext, slot *SlotContext, traceLog *msgTraceLog)  {
	bh := slot.BH
	castor := groupsig.DeserializeId(bh.Castor)
	if slot.StatusTransform(SS_WAITING, SS_SIGNED) && !castor.IsEqual(p.GetMinerID()) {
		vctx.updateSignedMaxQN(bh.TotalQN)
		vctx.increaseSignedNum()
		skey := p.getSignKey(gid)
		var cvm model.ConsensusVerifyMessage
		cvm.BlockHash = bh.Hash
		//cvm.GroupID = gId
		blog := newBizLog("normalPieceVerify")
		if cvm.GenSign(model.NewSecKeyInfo(p.GetMinerID(), skey), &cvm) {
			cvm.GenRandomSign(skey, vctx.prevBH.Random)
			blog.debug("call network service SendVerifiedCast hash=%v, height=%v", bh.Hash.ShortS(), bh.Height)
			traceLog.log("SendVerifiedCast height=%v, castor=%v", bh.Height, slot.castor.ShortS())
			//验证消息需要给自己也发一份，否则自己的分片中将不包含自己的签名，导致分红没有
			p.NetServer.SendVerifiedCast(&cvm, gid)
		} else {
			blog.log("genSign fail, id=%v, sk=%v %v", p.GetMinerID().ShortS(), skey.ShortS(), p.IsMinerGroup(gid))
		}
	}
}

func (p *Processor) doVerify(mtype string, msg *model.ConsensusCastMessage, traceLog *msgTraceLog, blog *bizLog, slog *slowLog) (err error) {
	bh := &msg.BH
	si := &msg.SI

	sender := si.SignMember.ShortS()

	gid := groupsig.DeserializeId(bh.GroupId)
	castor := groupsig.DeserializeId(bh.Castor)

	slog.addStage("checkOnChain")
	if p.blockOnChain(bh.Hash) {
		slog.endStage()
		err = fmt.Errorf("block onchain already")
		return
	}
	slog.endStage()

	slog.addStage("checkPreBlock")
	preBH := p.getBlockHeaderByHash(bh.PreHash)
	slog.endStage()

	slog.addStage("baseCheck")
	if preBH == nil {
		p.addFutureVerifyMsg(msg)
		return fmt.Errorf("父块未到达")
	}
	if expireTime, expire := VerifyBHExpire(bh, preBH); expire {
		return fmt.Errorf("cast verify expire, gid=%v, preTime %v, expire %v", gid.ShortS(), preBH.CurTime, expireTime)
	} else if bh.Height > 1 {
		//设置为2倍的最大时间，防止由于时间不同步导致的跳块
		beginTime := expireTime.Add(-2*time.Second*time.Duration(model.Param.MaxGroupCastTime))
		if !time.Now().After(beginTime) {
			return fmt.Errorf("cast begin time illegal, expectBegin at %v, expire at %v", beginTime, expireTime)
		}

	}
	if !p.IsMinerGroup(gid) {
		return fmt.Errorf("%v is not in group %v", p.GetMinerID().ShortS(), gid.ShortS())
	}
	bc := p.GetBlockContext(gid)
	if bc == nil {
		err = fmt.Errorf("未获取到blockcontext, gid=" + gid.ShortS())
		return
	}

	if _, same := bc.IsHeightCasted(bh.Height, bh.PreHash); same {
		err = fmt.Errorf("该高度已铸过 %v", bh.Height)
		return
	}

	vctx := bc.GetOrNewVerifyContext(bh, preBH)
	if vctx == nil {
		err = fmt.Errorf("获取vctx为空，可能preBH已经被删除")
		return
	}
	var slot *SlotContext
	slot, err = vctx.baseCheck(bh, si.GetID())
	if err != nil {
		return
	}
	slog.endStage()

	var pk groupsig.Pubkey
	isProposal := castor.IsEqual(si.GetID())

	slog.addStage("getPK")
	if isProposal { //提案者
		castorDO := p.minerReader.getProposeMiner(castor)
		if castorDO == nil {
			err = fmt.Errorf("castorDO nil id=%v", castor.ShortS())
			return
		}
		pk = castorDO.PK

	} else {
		memPK, ok := p.GetMemberSignPubKey(model.NewGroupMinerID(gid, si.GetID()))
		if !ok {
			blog.log("GetMemberSignPubKey not ok, ask id %v", si.GetID().ShortS())
			return
		}
		pk = memPK
	}
	slog.endStage()
	if !pk.IsValid() {
		err = fmt.Errorf("get pk fail")
		return
	}

	//未签过名的情况下，需要校验铸块合法性和全量账本检查
	if slot == nil || slot.IsWaiting() {
		slog.addStage("checkLegal")
		ok, _, err2 := p.isCastLegal(bh, preBH)
		slog.endStage()
		if !ok {
			err = err2
			return
		}

		//校验提案者是否有全量账本
		//slog.addStage("sampleCheck")
		//sampleHeight := p.sampleBlockHeight(bh.Height, preBH.Random, p.GetMinerID())
		//realHeight, existHash := p.getNearestVerifyHashByHeight(sampleHeight)
		//if realHeight > 0 {
		//	if !existHash.IsValid() {
		//		err = fmt.Errorf("MainChain GetCheckValue error, height=%v, err=%v", sampleHeight, err)
		//		return
		//	}
		//	vHash := msg.ProveHash[p.getMinerPos(gid, p.GetMinerID())]
		//	if vHash != existHash {
		//		err = fmt.Errorf("check p rove hash fail, sampleHeight=%v, realHeight=%v, receive hash=%v, exist hash=%v", sampleHeight, realHeight, vHash.String(), existHash.String())
		//		return
		//	}
		//}
		slog.endStage()
	}

	slog.addStage("UVCheck")
	blog.debug("%v start UserVerified, height=%v, hash=%v", mtype, bh.Height, bh.Hash.ShortS())
	verifyResult, err := vctx.UserVerified(bh, si, pk, slog)
	slog.endStage()
	blog.log("proc(%v) UserVerified height=%v, hash=%v, result=%v.%v", p.getPrefix(), bh.Height, bh.Hash.ShortS(), CBMR_RESULT_DESC(verifyResult), err)
	if err != nil {
		return
	}

	slot = vctx.GetSlotByHash(bh.Hash)
	if slot == nil {
		err = fmt.Errorf("找不到合适的验证槽, 放弃验证")
		return
	}

	err = fmt.Errorf("%v, %v, %v", CBMR_RESULT_DESC(verifyResult), slot.gSignGenerator.Brief(), slot.TransBrief())

	switch verifyResult {
	case CBMR_THRESHOLD_SUCCESS:
		if !slot.HasTransLost() {
			p.thresholdPieceVerify(mtype, sender, gid, vctx, slot, traceLog)
		}

	case CBMR_PIECE_NORMAL:
		slog.addStage("normPiece")
		p.normalPieceVerify(mtype, sender, gid, vctx, slot, traceLog)
		slog.endStage()

	case CBMR_PIECE_LOSINGTRANS: //交易缺失
	}
	return
}

func (p *Processor) verifyCastMessage(mtype string, msg *model.ConsensusCastMessage) {
	bh := &msg.BH
	si := &msg.SI
	blog := newBizLog(mtype)
	traceLog := newHashTraceLog(mtype, bh.Hash, si.GetID())
	castor := groupsig.DeserializeId(bh.Castor)
	groupId := groupsig.DeserializeId(bh.GroupId)

	slog := newSlowLog(mtype, 0.5)

	traceLog.logStart("height=%v, castor=%v", bh.Height, castor.ShortS())
	blog.debug("proc(%v) begin hash=%v, height=%v, sender=%v, castor=%v, groupId=%v", p.getPrefix(), bh.Hash.ShortS(), bh.Height, si.GetID().ShortS(), castor.ShortS(), groupId.ShortS())

	result := ""

	defer func() {
		traceLog.logEnd("height=%v, hash=%v, preHash=%v,groupId=%v, result=%v", bh.Height, bh.Hash.ShortS(), bh.PreHash.ShortS(),groupId.ShortS(), result)
		blog.debug("height=%v, hash=%v, preHash=%v, groupId=%v, result=%v", bh.Height, bh.Hash.ShortS(), bh.PreHash.ShortS(), groupId.ShortS(), result)
		slog.log("sender=%v, hash=%v, gid=%v, height=%v", si.GetID().ShortS(), bh.Hash.ShortS(), groupId.ShortS(), bh.Height)
	}()

	if !p.IsMinerGroup(groupId) { //检测当前节点是否在该铸块组
		result = fmt.Sprintf("don't belong to group, gid=%v, hash=%v, id=%v", groupId.ShortS(), bh.Hash.ShortS(), p.GetMinerID().ShortS())
		return
	}

	//castor要忽略自己的消息
	if castor.IsEqual(p.GetMinerID()) && si.GetID().IsEqual(p.GetMinerID()) {
		result = "ignore self message"
		return
	}

	if msg.GenHash() != si.DataHash {
		blog.debug("msg proveHash=%v", msg.ProveHash)
		result = fmt.Sprintf("msg genHash %v diff from si.DataHash %v", msg.GenHash().ShortS(), si.DataHash.ShortS())
		return
	}
	bc := p.GetBlockContext(groupId)
	if bc == nil {
		result = fmt.Sprintf("未获取到blockcontext, gid=" + groupId.ShortS())
		return
	}
	vctx := bc.GetVerifyContextByHeight(bh.Height)
	if vctx != nil {
		_, err := vctx.baseCheck(bh, si.GetID())
		if err != nil {
			result = err.Error()
			return
		}
	}
	middleware.PerfLogger.Infof("verify msg2, cost: %v, height: %v, hash: %v", time.Since(bh.CurTime), bh.Height, bh.Hash.String())

	err := p.doVerify(mtype, msg, traceLog, blog, slog)
	if err != nil {
		result = err.Error()
	}

	middleware.PerfLogger.Infof("verify msg3, cost: %v, height: %v, hash: %v", time.Since(bh.CurTime), bh.Height, bh.Hash.String())
	return
}

func (p *Processor) verifyWithCache(cache *verifyMsgCache, vmsg *model.ConsensusVerifyMessage)  {
	msg := &model.ConsensusCastMessage{
		BH: cache.castMsg.BH,
		ProveHash: cache.castMsg.ProveHash,
		BaseSignedMessage: vmsg.BaseSignedMessage,
	}
	msg.BH.Random = vmsg.RandomSign.Serialize()
	p.verifyCastMessage("OMV", msg)
}

//收到组内成员的出块消息，出块人（KING）用组分片密钥进行了签名
//有可能没有收到OnMessageCurrent就提前接收了该消息（网络时序问题）
func (p *Processor) OnMessageCast(ccm *model.ConsensusCastMessage) {
	slog := newSlowLog("OnMessageCast", 0.5)
	bh := &ccm.BH
	defer func() {
		slog.log("hash=%v, sender=%v, height=%v, preHash=%v", bh.Hash.ShortS(), ccm.SI.GetID().ShortS(), bh.Height, bh.PreHash.ShortS())
	}()

	//le := &monitor.LogEntry{
	//	LogType:  monitor.LogTypeProposal,
	//	Height:   bh.Height,
	//	Hash:     bh.Hash.Hex(),
	//	PreHash:  bh.PreHash.Hex(),
	//	Proposer: ccm.SI.GetID().GetHexString(),
	//	Verifier: groupsig.DeserializeId(bh.GroupId).GetHexString(),
	//	Ext:      fmt.Sprintf("external:qn:%v,totalQN:%v", 0, bh.TotalQN),
	//}
	slog.addStage("getGroup")
	slog.endStage()
	slog.addStage("addLog")
	//detalHeight := int(bh.Height - p.MainChain.Height())
	//group := p.GetGroup(groupsig.DeserializeId(bh.GroupId))
	//if mathext.AbsInt(detalHeight) < 100 && monitor.Instance.IsFirstNInternalNodesInGroup(group.GetMembers(), 10) {
	//	monitor.Instance.AddLogIfNotInternalNodes(le)
	//}
	slog.endStage()

	slog.addStage("addtoCache")
	p.addCastMsgToCache(ccm)
	cache := p.getVerifyMsgCache(ccm.BH.Hash)
	slog.endStage()

	slog.addStage("OMC")
	// 主要耗时点
	middleware.PerfLogger.Infof("verify msg1, cost: %v, height: %v, hash: %v", time.Since(bh.CurTime), bh.Height, bh.Hash.String())
	p.verifyCastMessage("OMC", ccm)
	slog.endStage()

	verifys := cache.getVerifyMsgs()
	if len(verifys) > 0 {
		slog.addStage("OMCVerifies")
		stdLogger.Infof("OMC:getVerifyMsgs from cache size %v, hash=%v", len(verifys), ccm.BH.Hash.ShortS())
		for _, vmsg := range verifys {
			p.verifyWithCache(cache, vmsg)
		}
		cache.removeVerifyMsgs()
		slog.endStage()
	}

}

//收到组内成员的出块验证通过消息（组内成员消息）
func (p *Processor) OnMessageVerify(cvm *model.ConsensusVerifyMessage) {
	//statistics.AddBlockLog(common.BootId, statistics.RcvVerified, cvm.BH.Height, cvm.BH.ProveValue.Uint64(), -1, -1,
	//	time.Now().UnixNano(), "", "", common.InstanceIndex, cvm.BH.CurTime.UnixNano())
	if p.blockOnChain(cvm.BlockHash) {
		return
	}
	cache := p.getVerifyMsgCache(cvm.BlockHash)
	if cache != nil && cache.castMsg != nil {
		p.verifyWithCache(cache, cvm)
	} else {
		stdLogger.Infof("OMV:no cast msg, cached, hash=%v", cvm.BlockHash.ShortS())
		p.addVerifyMsgToCache(cvm)
	}
}

//func (p *Processor) receiveBlock(block *types.Block, preBH *types.BlockHeader) bool {
//	if ok, err := p.isCastLegal(block.Header, preBH); ok { //铸块组合法
//		result := p.doAddOnChain(block)
//		if result == 0 || result == 1 {
//			return true
//		}
//	} else {
//		//丢弃该块
//		newBizLog("receiveBlock").log("received invalid new block, height=%v, err=%v", block.Header.Height, err.Error())
//	}
//	return false
//}

func (p *Processor) cleanVerifyContext(currentHeight uint64) {
	p.blockContexts.forEachBlockContext(func(bc *BlockContext) bool {
		bc.CleanVerifyContext(currentHeight)
		return true
	})
}

//收到铸块上链消息(组外矿工节点处理)
func (p *Processor) OnMessageBlock(cbm *model.ConsensusBlockMessage) {
	//statistics.AddBlockLog(common.BootId,statistics.RcvNewBlock,cbm.Block.Header.Height,cbm.Block.Header.ProveValue.Uint64(),len(cbm.Block.Transactions),-1,
	//	time.Now().UnixNano(),"","",common.InstanceIndex,cbm.Block.Header.CurTime.UnixNano())
	//bh := cbm.Block.Header
	//blog := newBizLog("OMB")
	//tlog := newHashTraceLog("OMB", bh.Hash, groupsig.DeserializeId(bh.Castor))
	//tlog.logStart("height=%v, preHash=%v", bh.Height, bh.PreHash.ShortS())
	//result := ""
	//defer func() {
	//	tlog.logEnd("height=%v, preHash=%v, result=%v", bh.Height, bh.PreHash.ShortS(), result)
	//}()
	//
	//if p.getBlockHeaderByHash(cbm.Block.Header.Hash) != nil {
	//	//blog.log("OMB receive block already on chain! bh=%v", p.blockPreview(cbm.Block.Header))
	//	result = "已经在链上"
	//	return
	//}
	//var gid = groupsig.DeserializeId(cbm.Block.Header.GroupId)
	//
	//blog.log("proc(%v) begin OMB, group=%v, height=%v, hash=%v...", p.getPrefix(),
	//	gid.ShortS(), cbm.Block.Header.Height, bh.Hash.ShortS())
	//
	//block := &cbm.Block
	//
	//preHeader := p.MainChain.GetTraceHeader(block.Header.PreHash.Bytes())
	//if preHeader == nil {
	//	p.addFutureBlockMsg(cbm)
	//	result = "父块未到达"
	//	return
	//}
	////panic("isBHCastLegal: cannot find pre block header!,ignore block")
	//verify := p.verifyGroupSign(cbm, preHeader)
	//if !verify {
	//	result = "组签名未通过"
	//	blog.log("OMB verifyGroupSign result=%v.", verify)
	//	return
	//}
	//
	//ret := p.receiveBlock(block, preHeader)
	//if ret {
	//	result = "上链成功"
	//} else {
	//	result = "上链失败"
	//}

	//blog.log("proc(%v) end OMB, group=%v, sender=%v...", p.getPrefix(), GetIDPrefix(cbm.GroupID), GetIDPrefix(cbm.SI.GetID()))
	return
}

//新的交易到达通知（用于处理大臣验证消息时缺失的交易）
func (p *Processor) OnMessageNewTransactions(ths []common.Hashes) {
	mtype := "OMNT"
	blog := newBizLog(mtype)

	txstrings := make([]string, len(ths))
	for idx, tx := range ths {
		txstrings[idx] = tx.ShortS()
	}

	blog.debug("proc(%v) begin %v, trans count=%v %v...", p.getPrefix(),mtype, len(ths), txstrings)

	p.blockContexts.forEachBlockContext(func(bc *BlockContext) bool {
		for _, vctx := range bc.SafeGetVerifyContexts() {
			for _, slot := range vctx.GetSlots() {
				acceptRet := vctx.AcceptTrans(slot, ths)
				tlog := newHashTraceLog(mtype, slot.BH.Hash, groupsig.ID{})
				switch acceptRet {
				case TRANS_INVALID_SLOT, TRANS_DENY:

				case TRANS_ACCEPT_NOT_FULL:
					blog.debug("accept trans bh=%v, ret %v", p.blockPreview(slot.BH), acceptRet)
					tlog.log("preHash=%v, height=%v, %v,收到 %v, 总交易数 %v, 仍缺失数 %v", slot.BH.PreHash.ShortS(), slot.BH.Height, TRANS_ACCEPT_RESULT_DESC(acceptRet), len(ths), len(slot.BH.Transactions), slot.lostTransSize())

				case TRANS_ACCEPT_FULL_PIECE:
					blog.debug("accept trans bh=%v, ret %v", p.blockPreview(slot.BH), acceptRet)
					tlog.log("preHash=%v, height=%v %v, 当前分片数%v", slot.BH.PreHash.ShortS(), slot.BH.Height, TRANS_ACCEPT_RESULT_DESC(acceptRet), slot.MessageSize())

				case TRANS_ACCEPT_FULL_THRESHOLD:
					blog.debug("accept trans bh=%v, ret %v", p.blockPreview(slot.BH), acceptRet)
					tlog.log("preHash=%v, height=%v, %v", slot.BH.PreHash.ShortS(), slot.BH.Height, TRANS_ACCEPT_RESULT_DESC(acceptRet))
					if len(slot.BH.Signature) == 0 {
						blog.log("slot bh sign is empty hash=%v", slot.BH.Hash.ShortS())
					}
					p.thresholdPieceVerify(mtype, p.getPrefix(), bc.MinerID.Gid, vctx, slot, tlog)
				}

			}
		}
		return true
	})

	return
}
