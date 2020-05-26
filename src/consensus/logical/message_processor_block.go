package logical

import (
	"fmt"
	"time"
	"x/src/common"
	"x/src/consensus/groupsig"
	"x/src/consensus/model"
	"x/src/middleware/types"

	"x/src/consensus/logical/group_create"
	"x/src/middleware"
	"x/src/utility"
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

// 已经恢复出组签名
// 出块
func (p *Processor) thresholdPieceVerify(vctx *VerifyContext) {
	blog := newBizLog("thresholdPieceVerify")

	bh := vctx.slot.BH
	if vctx.castSuccess() {
		blog.debug("already cast success, height=%v", bh.Height)
		return
	}

	p.reserveBlock(vctx)
}

// 收集签名
func (p *Processor) normalPieceVerify(gid groupsig.ID, vctx *VerifyContext, traceLog *msgTraceLog) {
	slot := vctx.slot
	bh := slot.BH
	castor := groupsig.DeserializeID(bh.Castor)
	if slot.StatusTransform(SS_WAITING, SS_SIGNED) && !castor.IsEqual(p.GetMinerID()) {
		vctx.increaseSignedNum()
		skey := p.getSignKey(gid)
		var cvm model.ConsensusVerifyMessage
		cvm.BlockHash = bh.Hash
		//cvm.GroupID = gId
		blog := newBizLog("normalPieceVerify")
		if signInfo, ok := model.NewSignInfo(skey, p.mi.ID, &cvm); ok {
			cvm.SignInfo = signInfo
			blog.debug("SendVerifiedCast seckey=%v, miner id=%v,data hash:%v,sig:%v", skey.GetHexString(), p.mi.ID.GetHexString(), cvm.SignInfo.GetDataHash().String(), cvm.SignInfo.GetSignature().GetHexString())
			cvm.GenRandomSign(skey, vctx.prevBH.Random)
			blog.debug("call network service SendVerifiedCast hash=%v, height=%v", bh.Hash.ShortS(), bh.Height)
			traceLog.log("SendVerifiedCast height=%v, castor=%v", bh.Height, slot.castor.ShortS())
			//验证消息需要给自己也发一份，否则自己的分片中将不包含自己的签名，导致分红没有???
			p.NetServer.SendVerifiedCast(&cvm, gid)
		} else {
			blog.log("genSign fail, id=%v, sk=%v %v", p.GetMinerID().ShortS(), skey.ShortS(), p.belongGroups.BelongGroup(gid))
		}
	}
}

// mtype 输出日志用的
func (p *Processor) doVerify(mtype string, msg *model.ConsensusCastMessage, traceLog *msgTraceLog, blog *bizLog, slog *slowLog) (err error) {
	bh := &msg.BH
	bh.CurTime = utility.FormatTime(bh.CurTime)
	si := &msg.SignInfo

	gid := groupsig.DeserializeID(bh.GroupId)
	castor := groupsig.DeserializeID(bh.Castor)

	// step1
	slog.addStage("checkOnChain")
	if p.blockOnChain(bh.Hash) {
		slog.endStage()
		err = fmt.Errorf("block onchain already")
		return
	}
	slog.endStage()

	// step2
	slog.addStage("checkPreBlock")
	preBH := p.getBlockHeaderByHash(bh.PreHash)
	slog.endStage()

	// step3
	slog.addStage("baseCheck")
	if preBH == nil {
		p.addFutureVerifyMsg(msg)
		return fmt.Errorf("父块未到达")
	}

	timeNow := utility.GetTime()
	deviationTime := bh.CurTime.Add(time.Second * -1)
	if !bh.CurTime.After(preBH.CurTime) || !timeNow.After(deviationTime) {
		return fmt.Errorf("cast time illegal! current block cast time %v, pre block cast time %v,deviation time:%v,time now %v", bh.CurTime, preBH.CurTime, deviationTime, timeNow)
	}

	if !p.belongGroups.BelongGroup(gid) {
		return fmt.Errorf("%v is not in group %v", p.GetMinerID().ShortS(), gid.ShortS())
	}

	bc := p.GetBlockContext(gid)
	if bc == nil {
		err = fmt.Errorf("未获取到blockcontext, gid=" + gid.ShortS())
		return
	}

	if _, same := bc.IsHashCasted(bh.Hash, bh.PreHash); same {
		err = fmt.Errorf("该hash已铸过 %v", bh.Hash)
		return
	}

	if err = bc.CheckQN(bh); err != nil {
		return err
	}

	vctx := bc.GetOrNewVerifyContext(bh, preBH)
	if vctx == nil {
		err = fmt.Errorf("获取vctx为空，可能preBH已经被删除")
		return
	}
	var slot *SlotContext
	slot, err = vctx.baseCheck(bh, si.GetSignerID())
	if err != nil {
		return
	}
	slog.endStage()

	var pk groupsig.Pubkey
	isProposal := castor.IsEqual(si.GetSignerID())

	slog.addStage("getPK")
	if isProposal { //提案者
		castorDO := p.minerReader.GetProposeMiner(castor)
		if castorDO == nil {
			err = fmt.Errorf("castorDO nil id=%v", castor.ShortS())
			return
		}
		pk = castorDO.PubKey

	} else {
		memPK, ok := group_create.GroupCreateProcessor.GetMemberSignPubKey(gid, si.GetSignerID())
		if !ok {
			blog.log("GetMemberSignPubKey not ok, ask id %v", si.GetSignerID().ShortS())
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

		slog.endStage()
	}

	slog.addStage("UVCheck")
	blog.debug("%v start UserVerified, height=%v, hash=%v", mtype, bh.Height, bh.Hash.ShortS())

	id := utility.GetGoroutineId()
	middleware.PerfLogger.Infof("verify before UserVerified %s, id: %d, cost: %v, height: %v, hash: %v", mtype, id, time.Since(bh.CurTime), bh.Height, bh.Hash.String())

	verifyResult, err := vctx.UserVerified(bh, si, pk, slog)
	slog.endStage()
	blog.log("proc(%v) UserVerified height=%v, hash=%v, result=%v.%v", p.getPrefix(), bh.Height, bh.Hash.ShortS(), CBMR_RESULT_DESC(verifyResult), err)
	if err != nil {
		return
	}

	slot = vctx.slot
	err = fmt.Errorf("%v, %v, %v", CBMR_RESULT_DESC(verifyResult), slot.gSignGenerator.String(), slot.TransBrief())

	switch verifyResult {
	case CBMR_THRESHOLD_SUCCESS:
		if !slot.HasTransLost() {
			p.thresholdPieceVerify(vctx)
		}

	case CBMR_PIECE_NORMAL:
		slog.addStage("normPiece")
		p.normalPieceVerify(gid, vctx, traceLog)
		slog.endStage()

	case CBMR_PIECE_LOSINGTRANS: //交易缺失
	}
	return
}

// 这个方法会被调用多次：
// 1：收到candidate块，验证 OMC
// 2：收到别的验证者的消息，要验证 OMV
func (p *Processor) verifyCastMessage(mtype string, msg *model.ConsensusCastMessage) {
	bh := &msg.BH
	si := &msg.SignInfo
	blog := newBizLog(mtype)
	traceLog := newHashTraceLog(mtype, bh.Hash, si.GetSignerID())
	castor := groupsig.DeserializeID(bh.Castor)
	groupId := groupsig.DeserializeID(bh.GroupId)

	slog := newSlowLog(mtype, 0.5)

	traceLog.logStart("height=%v, castor=%v", bh.Height, castor.ShortS())
	blog.debug("proc(%v) begin hash=%v, height=%v, sender=%v, castor=%v, groupId=%v", p.getPrefix(), bh.Hash.ShortS(), bh.Height, si.GetSignerID().ShortS(), castor.ShortS(), groupId.ShortS())

	result := ""

	defer func() {
		traceLog.logEnd("height=%v, hash=%v, preHash=%v,groupId=%v, result=%v", bh.Height, bh.Hash.ShortS(), bh.PreHash.ShortS(), groupId.ShortS(), result)
		blog.debug("height=%v, hash=%v, preHash=%v, groupId=%v, result=%v", bh.Height, bh.Hash.ShortS(), bh.PreHash.ShortS(), groupId.ShortS(), result)
		slog.log("sender=%v, hash=%v, gid=%v, height=%v", si.GetSignerID().ShortS(), bh.Hash.ShortS(), groupId.ShortS(), bh.Height)
	}()

	// 不要重复验证
	if !p.belongGroups.BelongGroup(groupId) { //检测当前节点是否在该铸块组
		result = fmt.Sprintf("don't belong to group, gid=%v, hash=%v, id=%v", groupId.ShortS(), bh.Hash.ShortS(), p.GetMinerID().ShortS())
		return
	}

	//castor要忽略自己的消息
	// 不要重复验证
	if castor.IsEqual(p.GetMinerID()) && si.GetSignerID().IsEqual(p.GetMinerID()) {
		result = "ignore self message"
		return
	}

	// 要重复验证
	if msg.GenHash() != si.GetDataHash() {
		blog.debug("msg proveHash=%v", msg.ProveHash)
		result = fmt.Sprintf("msg genHash %v diff from si.DataHash %v", msg.GenHash().ShortS(), si.GetDataHash().ShortS())
		return
	}

	bc := p.GetBlockContext(groupId)
	if bc == nil {
		result = fmt.Sprintf("未获取到blockcontext, gid=" + groupId.ShortS())
		return
	}
	vctx := bc.GetVerifyContextByHash(bh.Hash)
	if vctx != nil {
		_, err := vctx.baseCheck(bh, si.GetSignerID())
		if err != nil {
			result = err.Error()
			return
		}
	}

	// 主要耗时点
	err := p.doVerify(mtype, msg, traceLog, blog, slog)
	if err != nil {
		result = err.Error()
	}

	id := utility.GetGoroutineId()
	middleware.PerfLogger.Infof("verified %s, id: %d, cost: %v, height: %v, hash: %v", mtype, id, time.Since(bh.CurTime), bh.Height, bh.Hash.String())
	return
}

func (p *Processor) verifyWithCache(cache *verifyMsgCache, vmsg *model.ConsensusVerifyMessage) {
	msg := &model.ConsensusCastMessage{
		BH:        cache.castMsg.BH,
		ProveHash: cache.castMsg.ProveHash,
		SignInfo:  vmsg.SignInfo,
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
		slog.log("hash=%v, sender=%v, height=%v, preHash=%v", bh.Hash.ShortS(), ccm.SignInfo.GetSignerID().ShortS(), bh.Height, bh.PreHash.ShortS())
	}()

	slog.addStage("addtoCache")
	p.addCastMsgToCache(ccm)
	cache := p.getVerifyMsgCache(ccm.BH.Hash)
	slog.endStage()

	slog.addStage("OMC")

	// 主要耗时点
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
	//	utility.GetTime().UnixNano(), "", "", common.InstanceIndex, cvm.BH.CurTime.UnixNano())
	if p.blockOnChain(cvm.BlockHash) {
		return
	}

	cache := p.getVerifyMsgCache(cvm.BlockHash)
	if cache != nil && cache.castMsg != nil {
		p.verifyWithCache(cache, cvm)
	} else {
		stdLogger.Infof("OMV:no cast msg, cached, hash=%v", cvm.BlockHash.ShortS())

		// 块没收到，验证消息先到？？？
		// 那为啥不触发校验？
		p.addVerifyMsgToCache(cvm)
	}
}

func (p *Processor) cleanVerifyContext(currentHeight uint64) {
	p.blockContexts.forEachBlockContext(func(bc *BlockContext) bool {
		bc.CleanVerifyContext(currentHeight)
		return true
	})
}

//新的交易到达通知（用于处理大臣验证消息时缺失的交易）
func (p *Processor) OnMessageNewTransactions(ths []common.Hashes) {
	mtype := "OMNT"
	blog := newBizLog(mtype)

	txstrings := make([]string, len(ths))
	for idx, tx := range ths {
		txstrings[idx] = tx.ShortS()
	}

	blog.debug("proc(%v) begin %v, trans count=%v %v...", p.getPrefix(), mtype, len(ths), txstrings)

	p.blockContexts.forEachBlockContext(func(bc *BlockContext) bool {
		for _, vctx := range bc.SafeGetVerifyContexts() {
			slot := vctx.slot
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
				p.thresholdPieceVerify(vctx)
			}

		}
		return true
	})

	return
}
