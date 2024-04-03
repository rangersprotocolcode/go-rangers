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
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/middleware/types"
	"fmt"
	"time"

	"com.tuntun.rangers/node/src/consensus/logical/group_create"
	"com.tuntun.rangers/node/src/middleware"
	"com.tuntun.rangers/node/src/utility"
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

func (p *Processor) thresholdPieceVerify(vctx *VerifyContext) {
	blog := newBizLog("thresholdPieceVerify")

	bh := vctx.slot.BH
	if vctx.castSuccess() {
		blog.debug("already cast success, height=%v", bh.Height)
		return
	}

	p.reserveBlock(vctx)
}

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

			p.NetServer.SendVerifiedCast(&cvm, gid)
		} else {
			blog.log("genSign fail, id=%v, sk=%v %v", p.GetMinerID().ShortS(), skey.ShortS(), p.belongGroups.BelongGroup(gid))
		}
	}
}

func (p *Processor) doVerify(mtype string, msg *model.ConsensusCastMessage, traceLog *msgTraceLog, blog *bizLog, slog *slowLog) (err error) {
	bh := msg.BH
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
		return fmt.Errorf("no parent block")
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
		err = fmt.Errorf("cannot get blockcontext, gid=" + gid.ShortS())
		return
	}

	if _, same := bc.IsHashCasted(bh.Hash, bh.PreHash); same {
		err = fmt.Errorf("hash already casted %v", bh.Hash)
		return
	}

	if err = bc.CheckQN(bh); err != nil {
		return err
	}

	vctx := bc.GetOrNewVerifyContext(bh, preBH)
	if vctx == nil {
		err = fmt.Errorf("no vctx，preBH may be deleted")
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
	if isProposal {
		castorDO := p.minerReader.GetProposeMiner(castor, preBH.StateTree)
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
	middleware.PerfLogger.Infof("verify before UserVerified %s, id: %d, cost: %v, height: %v, hash: %v", mtype, id, utility.GetTime().Sub(bh.CurTime), bh.Height, bh.Hash.String())

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

	case CBMR_PIECE_LOSINGTRANS:
	}
	return
}

func (p *Processor) verifyCastMessage(mtype string, msg *model.ConsensusCastMessage) {
	bh := msg.BH
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

	if !p.belongGroups.BelongGroup(groupId) {
		result = fmt.Sprintf("don't belong to group, gid=%v, hash=%v, id=%v", groupId.ShortS(), bh.Hash.ShortS(), p.GetMinerID().ShortS())
		return
	}

	if castor.IsEqual(p.GetMinerID()) && si.GetSignerID().IsEqual(p.GetMinerID()) {
		result = "ignore self message"
		return
	}

	if msg.GenHash() != si.GetDataHash() {
		blog.debug("msg proveHash=%v", msg.ProveHash)
		result = fmt.Sprintf("msg genHash %v diff from si.DataHash %v", msg.GenHash().ShortS(), si.GetDataHash().ShortS())
		return
	}

	bc := p.GetBlockContext(groupId)
	if bc == nil {
		result = fmt.Sprintf("no blockcontext, gid=" + groupId.ShortS())
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

	err := p.doVerify(mtype, msg, traceLog, blog, slog)
	if err != nil {
		result = err.Error()
	}

	id := utility.GetGoroutineId()
	middleware.PerfLogger.Infof("verified %s, id: %d, cost: %v, height: %v, hash: %v", mtype, id, utility.GetTime().Sub(bh.CurTime), bh.Height, bh.Hash.String())
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

func (p *Processor) OnMessageCastV2(ccm *model.ConsensusCastMessage) {
	slog := newSlowLog("OnMessageCast", 0.5)
	bh := ccm.BH
	defer func() {
		slog.log("hash=%v, sender=%v, height=%v, preHash=%v", bh.Hash.ShortS(), ccm.SignInfo.GetSignerID().ShortS(), bh.Height, bh.PreHash.ShortS())
	}()

	slog.addStage("addtoCache")
	p.addCastMsgToCache(ccm)
	cache := p.getVerifyMsgCache(ccm.BH.Hash)
	slog.endStage()

	slog.addStage("OMC")

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

func (p *Processor) OnMessageVerifyV2(cvm *model.ConsensusVerifyMessage) {
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

func (p *Processor) cleanVerifyContext(currentHeight uint64) {
	p.blockContexts.forEachBlockContext(func(bc *BlockContext) bool {
		bc.CleanVerifyContext(currentHeight)
		return true
	})
}

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
			if slot == nil {
				continue
			}
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
