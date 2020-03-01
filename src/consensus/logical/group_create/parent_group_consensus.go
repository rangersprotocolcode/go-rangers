package group_create

import (
	"x/src/consensus/model"
	"fmt"
	"time"
	"x/src/consensus/groupsig"
	"x/src/consensus/access"
)

//发送PING 信息
// pingNodes send ping messages to the new members,
// in order to avoid too much ping messages, the current node does this only when he is one of kings.
func (p *groupCreateProcessor) pingNodes() {
	ctx := p.context
	if ctx == nil || !ctx.isKing() {
		return
	}
	msg := &model.CreateGroupPingMessage{
		FromGroupID: ctx.parentGroupInfo.GroupID,
		PingID:      ctx.pingID,
		BaseHeight:  ctx.baseBlockHeader.Height,
	}
	if signInfo, ok := model.NewSignInfo(p.minerInfo.SecKey, p.minerInfo.ID, msg); ok {
		msg.SignInfo = signInfo
		for _, id := range ctx.candidates {
			groupCreateLogger.Debugf("Send ping to id=%v,baseHeight=%v, pingID=%v, ", id.ShortS(), ctx.baseBlockHeader.Height, msg.PingID)
			p.NetServer.SendGroupPingMessage(msg, id)
		}
	}
}

//组成员候选人处理PING 信息
// OnMessageCreateGroupPing handles Ping request from parent nodes
// It only happens when current node is chosen to join a new group
func (p *groupCreateProcessor) OnMessageCreateGroupPing(msg *model.CreateGroupPingMessage) {
	var err error
	defer func() {
		if err != nil {
			groupCreateLogger.Errorf("Rcv create group ping. from %v, groupId %v, pingId %v, height=%v, won't pong, err=%v", msg.SignInfo.GetSignerID().ShortS(), msg.FromGroupID.ShortS(), msg.PingID, msg.BaseHeight, err)
		} else {
			groupCreateLogger.Debugf("Rcv create group ping. from %v, groupId %v, pingId %v, height=%v, pong!", msg.SignInfo.GetSignerID().ShortS(), msg.FromGroupID.ShortS(), msg.PingID, msg.BaseHeight)
		}
	}()
	pk := access.GetMinerPubKey(msg.SignInfo.GetSignerID())
	if pk == nil {
		return
	}
	if msg.VerifySign(*pk) {
		top := p.blockChain.Height()
		if top <= msg.BaseHeight {
			err = fmt.Errorf("localheight is %v, not enough", top)
			return
		}
		pongMsg := &model.CreateGroupPongMessage{
			PingID:    msg.PingID,
			Timestamp: time.Now(),
		}

		if signInfo, ok := model.NewSignInfo(p.minerInfo.SecKey, p.minerInfo.ID, pongMsg); ok {
			pongMsg.SignInfo = signInfo
			var belongGroup = false
			if p.joinedGroupStorage.BelongGroup(msg.FromGroupID) {
				belongGroup = true
			}
			p.NetServer.SendGroupPongMessage(pongMsg, msg.FromGroupID.GetHexString(), belongGroup)
		} else {
			err = fmt.Errorf("gen sign fail")
		}
	} else {
		err = fmt.Errorf("verify sign fail")
	}
}

//父亲组成员收到PONG信息
// OnMessageCreateGroupPong handles Pong response from new group candidates
// It only happens among the parent group nodes
func (p *groupCreateProcessor) OnMessageCreateGroupPong(msg *model.CreateGroupPongMessage) {
	var err error
	defer func() {
		groupCreateLogger.Debugf("OnMessageCreateGroupPong:rcv from %v, pingId %v, got pong, result=%v", msg.SignInfo.GetSignerID().ShortS(), msg.PingID, err)
	}()

	ctx := p.context
	if ctx == nil {
		err = fmt.Errorf("creatingGroupCtx is nil")
		return
	}
	if ctx.pingID != msg.PingID {
		err = fmt.Errorf("pingId not equal, expect=%v, got=%v", p.context.pingID, msg.PingID)
		return
	}
	pk := access.GetMinerPubKey(msg.SignInfo.GetSignerID())
	if pk == nil {
		return
	}

	if msg.VerifySign(*pk) {
		add, got := ctx.handlePong(p.blockChain.Height(), msg.SignInfo.GetSignerID())
		err = fmt.Errorf("size %v", got)
		if add {
			p.tryStartParentConsensus(p.blockChain.Height())
		}
	} else {
		err = fmt.Errorf("verify sign fail")
	}
}

//检查条件 生成组初始化信息，用自己的组签名私钥 签名并发送
//checkReqCreateGroupSign
func (p *groupCreateProcessor) tryStartParentConsensus(topHeight uint64) bool {
	ctx := p.context
	if ctx == nil {
		return false
	}

	var desc string
	defer func() {
		if desc != "" {
			groupCreateLogger.Infof("tryStartConsensus:context info=%v, %v", ctx.String(), desc)
		}
	}()

	if ctx.timeout(topHeight) {
		return false
	}

	pongsize := ctx.receivedPongCount()
	if ctx.getStatus() != waitingPong {
		return false
	}

	if !ctx.genGroupInitInfo(topHeight) {
		desc = fmt.Sprintf("cannot generate group info, pongsize %v, pongdeadline %v", pongsize, ctx.isPongTimeout(topHeight))
		return false
	}

	ctx.setStatus(waitingSign)
	gInfo := ctx.groupInitInfo
	gh := gInfo.GroupHeader

	if !ctx.isKing() {
		return false
	}
	if gInfo.MemberSize() < model.Param.GroupMemberMin {
		desc = fmt.Sprintf("got not enough pongs!, got %v", pongsize)
		return false
	}

	msg := &model.ParentGroupConsensusMessage{
		GroupInitInfo: *gInfo,
	}
	inGroupSignSecKey := p.getInGroupSignSecKey(gInfo.ParentGroupID())
	if signInfo, ok := model.NewSignInfo(inGroupSignSecKey, p.minerInfo.ID, msg); !ok {
		desc = fmt.Sprintf("genSign fail, id=%v, sk=%v", p.minerInfo.ID.ShortS(), p.minerInfo.SecKey.ShortS())
		return false
	} else {
		msg.SignInfo = signInfo
	}

	var belongGroup = false
	if p.joinedGroupStorage.BelongGroup(gInfo.ParentGroupID()) {
		belongGroup = true
	}
	p.NetServer.SendCreateGroupRawMessage(msg, belongGroup)
	desc = fmt.Sprintf("start parent group consensus. groupHash=%v, memsize=%v", gh.Hash.ShortS(), gInfo.MemberSize())
	return true
}

//父亲组成员收到建组消息后 用组签名私钥签名发送
// OnMessageCreateGroupRaw triggered when receives raw group-create message from other nodes of the parent group
// It check and sign the group-create message for the requester
//
// Before the formation of the new group, the parent group needs to reach a consensus on the information of the new group
// which transited by ConsensusCreateGroupRawMessage.
func (p *groupCreateProcessor) OnMessageParentGroupConsensus(msg *model.ParentGroupConsensusMessage) {
	gh := msg.GroupInitInfo.GroupHeader
	groupCreateLogger.Debugf("(%v)Rcv ParentGroupConsensus: groupHash=%v sender=%v", p.minerInfo.ID.ShortS(), gh.Hash.ShortS(), msg.SignInfo.GetSignerID().ShortS())

	if p.minerInfo.GetMinerID().IsEqual(msg.SignInfo.GetSignerID()) {
		return
	}
	parentGid := msg.GroupInitInfo.ParentGroupID()

	gpk, ok := p.GetMemberSignPubKey(parentGid, msg.SignInfo.GetSignerID())
	if !ok {
		groupCreateLogger.Errorf("getMemberSignPubKey not ok, ask id %v", parentGid.ShortS())
		return
	}

	if !msg.VerifySign(gpk) {
		groupCreateLogger.Errorf("ParentGroupConsensus verify sign error! pk:%s,sign:%s")
		return
	}
	if gh.Hash != gh.GenHash() || gh.Hash != msg.SignInfo.GetDataHash() {
		groupCreateLogger.Errorf("group hash diff! expect %v, receive %v", gh.GenHash().ShortS(), gh.Hash.ShortS())
		return
	}

	if ok, err := p.validateCreateGroupInfo(msg); ok {
		signMsg := &model.ParentGroupConsensusSignMessage{
			Launcher:  msg.SignInfo.GetSignerID(),
			GroupHash: gh.Hash,
		}
		inGroupSignSecKey := p.getInGroupSignSecKey(parentGid)
		if signInfo, ok := model.NewSignInfo(inGroupSignSecKey, p.minerInfo.ID, signMsg); ok {
			signMsg.SignInfo = signInfo
			p.NetServer.SendCreateGroupSignMessage(signMsg, parentGid)
			groupCreateLogger.Debugf("Send create group sign to: sender=%v,groupHash=%v", msg.SignInfo.GetSignerID().ShortS(), gh.Hash.ShortS(), )
		} else {
			groupCreateLogger.Errorf("ParentGroupConsensusSignMessage sign fail, signer id=%v,seckey=%v", p.minerInfo.ID.ShortS(), p.minerInfo.SecKey.ShortS())
		}
	} else {
		groupCreateLogger.Errorf("validate create group info failed , err:%v", err.Error())
	}
}

//onMessageCreateGroupRaw
func (p *groupCreateProcessor) validateCreateGroupInfo(msg *model.ParentGroupConsensusMessage) (bool, error) {
	ctx := p.context
	if ctx == nil {
		return false, fmt.Errorf("ctx is nil")
	}
	if ctx.getStatus() == sendInit {
		return false, fmt.Errorf("has send inited")
	}
	top := p.blockChain.Height()
	if ctx.timeout(top) {
		return false, fmt.Errorf("ready timeout")
	}
	if !ctx.genGroupInitInfo(top) {
		return false, fmt.Errorf("generate group init info fail")
	}
	if ctx.groupInitInfo.GroupHash() != msg.GroupInitInfo.GroupHash() {
		groupCreateLogger.Errorf("Illegal group header! expect gh %+v, real gh %+v", ctx.groupInitInfo.GroupHeader, msg.GroupInitInfo.GroupHeader)
		return false, fmt.Errorf("grouphash diff")
	}
	return true, nil
}

// OnMessageCreateGroupSign receives sign message from other members after ConsensusCreateGroupRawMessage was sent
// during the new-group-info consensus process
//OnMessageCreateGroupSign
func (p *groupCreateProcessor) OnMessageParentGroupConsensusSign(msg *model.ParentGroupConsensusSignMessage) {
	groupCreateLogger.Debugf("(%v)Rcv ParentGroupConsensusSignMessage, groupHash=%v, sender=%v", p.minerInfo.ID.ShortS(), msg.GroupHash.ShortS(), msg.SignInfo.GetSignerID().ShortS())
	if p.minerInfo.GetMinerID().IsEqual(msg.SignInfo.GetSignerID()) {
		return
	}

	if msg.GenHash() != msg.SignInfo.GetDataHash() {
		groupCreateLogger.Errorf("Msg hash validate error!Except:%s,real:%s", msg.SignInfo.GetDataHash().String(), msg.GenHash().String())
		return
	}

	ctx := p.context
	if ctx == nil {
		groupCreateLogger.Warnf("context is nil")
		return
	}
	mpk, ok := p.GetMemberSignPubKey(ctx.parentGroupInfo.GroupID, msg.SignInfo.GetSignerID())
	if !ok {
		groupCreateLogger.Errorf("can not get member sign pubkey , ask for %v", ctx.parentGroupInfo.GroupID.ShortS())
		return
	}
	if !msg.VerifySign(mpk) {
		groupCreateLogger.Errorf("ParentGroupConsensusSign verify sign error! pk:%s,sign:%s")
		return
	}

	if ok, err := p.tryRecoverParentGroupSig(msg); ok {
		groupPubkey := ctx.parentGroupInfo.GroupPK
		if !groupsig.VerifySig(groupPubkey, msg.SignInfo.GetDataHash().Bytes(), ctx.groupInitInfo.ParentGroupSign) {
			groupCreateLogger.Errorf("(%v)Verify group sign fail", p.minerInfo.ID.ShortS())
			return
		}
		initMsg := &model.GroupInitMessage{
			GroupInitInfo: *ctx.groupInitInfo,
		}

		if signInfo, ok := model.NewSignInfo(p.minerInfo.SecKey, p.minerInfo.ID, initMsg); ok && ctx.getStatus() != sendInit {
			initMsg.SignInfo = signInfo
			p.NetServer.SendGroupInitMessage(initMsg)
			ctx.setStatus(sendInit)
			groupCreateLogger.Infof("Send group init: context=%v, groupHash=%v, costHeight=%v", ctx.String(), ctx.groupInitInfo.GroupHash().ShortS(), p.blockChain.Height()-ctx.createTopHeight)
		} else {
			groupCreateLogger.Errorf("GroupInitMessage sign failed, signer id=%v,seckey=%v", p.minerInfo.ID.ShortS(), p.minerInfo.SecKey.ShortS())
		}
	} else {
		groupCreateLogger.Errorf("recover parent group sig failed, err=%v", err)
	}
}

//父亲组成员收到 带签名的组创建信息
func (p *groupCreateProcessor) tryRecoverParentGroupSig(msg *model.ParentGroupConsensusSignMessage) (bool, error) {
	ctx := p.context
	if ctx == nil {
		return false, fmt.Errorf("context is nil")
	}

	height := p.blockChain.TopBlock().Height
	if ctx.timeout(height) {
		return false, fmt.Errorf("ready timeout")
	}
	if ctx.groupInitInfo.GroupHash() != msg.GroupHash {
		return false, fmt.Errorf("gHash diff")
	}

	accept, recovered := ctx.acceptPiece(msg.SignInfo.GetSignerID(), msg.SignInfo.GetSignature())
	groupCreateLogger.Debugf("accept parent group consensus sign result: %v,recovered group sign:%v", accept, recovered)
	//newHashTraceLog("OMCGS", msg.GHash, msg.SI.GetID()).log("onMessageCreateGroupSign ret %v, %v", recover, ctx.gSignGenerator.Brief())
	if recovered {
		ctx.groupInitInfo.ParentGroupSign = ctx.groupSignGenerator.GetGroupSign()
		return true, nil
	}
	return false, fmt.Errorf("waiting more sign piece")
}
