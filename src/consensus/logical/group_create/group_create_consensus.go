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

package group_create

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus/access"
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/middleware/notify"
	"com.tuntun.rangers/node/src/utility"
	"fmt"
)

// OnMessageGroupInit receives new-group-info messages from parent nodes and starts the group formation process
// That indicates the current node is chosen to be a member of the new group
func (p *groupCreateProcessor) OnMessageGroupInit(msg *model.GroupInitMessage) {
	groupInitInfo := msg.GroupInitInfo
	groupHash := groupInitInfo.GroupHash()
	groupHeader := groupInitInfo.GroupHeader

	groupCreateLogger.Debugf("(%v)Rcv group init message, sender=%v, groupHash=%v...", p.minerInfo.ID.ShortS(), msg.SignInfo.GetSignerID().ShortS(), groupHash.ShortS())

	if msg.SignInfo.GetDataHash() != msg.GenHash() || groupHeader.Hash != groupHeader.GenHash() {
		panic("msg gis hash diff")
	}

	// Non-group members do not follow the follow-up process
	if !msg.MemberExist(p.minerInfo.GetMinerID()) {
		return
	}

	groupInitContext := p.groupInitContextCache.GetContext(groupHash)
	if groupInitContext != nil && groupInitContext.GetGroupStatus() != GisInit {
		groupCreateLogger.Debugf("already handle group init, status=%v", groupInitContext.GetGroupStatus())
		return
	}

	topHeight := p.blockChain.TopBlock().Height
	if groupInitInfo.ReadyTimeout(topHeight) {
		groupCreateLogger.Debugf("on group init message ready timeout, readyHeight=%v, now=%v", groupHeader.CreateHeight+model.Param.GroupReadyGap, topHeight)
		return
	}

	candidates, ok, err := p.ValidateGroupInfo(&msg.GroupInitInfo)
	if !ok {
		groupCreateLogger.Debugf("group header illegal, err=%v", err)
		return
	}

	groupInitContext = p.groupInitContextCache.GetOrNewContext(&groupInitInfo, candidates, &p.minerInfo)
	if groupInitContext == nil {
		panic("Processor::OMGI failed, ConfirmGroupFromRaw return nil.")
	}

	// join group by group hash
	p.NetServer.JoinGroupNet(groupHash.Hex())

	groupCreateLogger.Debugf("groupHash:%s,current status=%v.", groupHash.ShortS(), groupInitContext.GetGroupStatus())
	// Use CAS operation to make sure the logical below executed once
	if groupInitContext.TransformStatus(GisInit, GisSendSharePiece) {
		// Generate secret sharing
		shares := groupInitContext.GenSharePieces()
		sharePieceMessage := &model.SharePieceMessage{
			GroupHash:      groupHash,
			GroupMemberNum: int32(groupInitInfo.MemberSize()),
		}

		// Send each node a different piece
		for id, piece := range shares {
			if id != "0x0" && piece.IsValid() {
				sharePieceMessage.ReceiverId.SetHexString(id)
				sharePieceMessage.Share = piece

				if signInfo, ok := model.NewSignInfo(p.minerInfo.SecKey, p.minerInfo.ID, sharePieceMessage); ok {
					sharePieceMessage.SignInfo = signInfo
					groupCreateLogger.Debugf("piece to ID(%v), gHash=%v, share=%v, pub=%v.", sharePieceMessage.ReceiverId.ShortS(), groupHash.ShortS(), sharePieceMessage.Share.Share.ShortS(), sharePieceMessage.Share.Pub.ShortS())
					p.NetServer.SendKeySharePiece(sharePieceMessage)
				} else {
					groupCreateLogger.Errorf("genSign fail, id=%v, sk=%v", p.minerInfo.ID, p.minerInfo.SecKey.ShortS())
				}
			} else {
				groupCreateLogger.Errorf("GenSharePieces data is not Valid.")
			}
		}
	}
	return
}

// ValidateGroupInfo check whether the group info is legal
func (p *groupCreateProcessor) ValidateGroupInfo(groupInitInfo *model.GroupInitInfo) ([]groupsig.ID, bool, error) {
	groupHeader := groupInitInfo.GroupHeader
	if groupHeader.Hash != groupHeader.GenHash() {
		return nil, false, fmt.Errorf("gh hash error, hash=%v, genHash=%v", groupHeader.Hash.ShortS(), groupHeader.GenHash().ShortS())
	}

	// check if the member count is legal
	if !model.Param.IsGroupMemberCountLegal(len(groupInitInfo.GroupMembers)) {
		return nil, false, fmt.Errorf("group member size error %v(%v-%v)", len(groupInitInfo.GroupMembers), model.Param.GroupMemberMin, model.Param.GroupMemberMax)
	}
	// check if the created height is legal
	if !validateHeight(groupHeader.CreateHeight) {
		return nil, false, fmt.Errorf("cannot create at the height %v", groupHeader.CreateHeight)
	}
	baseBH := p.blockChain.QueryBlockByHash(common.BytesToHash(groupHeader.CreateBlockHash))
	if baseBH == nil {
		return nil, false, common.ErrCreateBlockNil
	}
	if baseBH.Header.Height != groupHeader.CreateHeight {
		return nil, false, fmt.Errorf("group base block height diff %v-%v", baseBH.Header.Height, groupHeader.CreateHeight)
	}
	// The previous group, whether the parent group exists
	preGroup := p.groupChain.GetGroupById(groupHeader.PreGroup)
	if preGroup == nil {
		return nil, false, fmt.Errorf("preGroup is nil, gid=%v", groupsig.DeserializeID(groupHeader.PreGroup).ShortS())
	}
	parentGroup := p.groupChain.GetGroupById(groupHeader.Parent)
	if parentGroup == nil {
		return nil, false, fmt.Errorf("parentGroup is nil, gid=%v", groupsig.DeserializeID(groupHeader.Parent).ShortS())
	}

	// check if it is the specified parent group
	sgi, err := p.selectParentGroup(baseBH.Header, groupHeader.PreGroup)
	if err != nil {
		return nil, false, fmt.Errorf("select parent group err %v", err)
	}
	pid := groupsig.DeserializeID(parentGroup.Id)
	if !sgi.GroupID.IsEqual(pid) {
		return nil, false, fmt.Errorf("select parent group not equal, expect %v, recieve %v", sgi.GroupID.ShortS(), pid.ShortS())
	}

	gpk := p.getGroupPubKey(groupsig.DeserializeID(groupHeader.Parent))

	// check the signature of the parent group
	if !groupsig.VerifySig(gpk, groupHeader.Hash.Bytes(), groupInitInfo.ParentGroupSign) {
		return nil, false, fmt.Errorf("verify parent sign fail")
	}

	// check if the candidates are legal
	enough, candidates := p.selectCandidates(baseBH.Header)
	if !enough {
		return nil, false, fmt.Errorf("not enough candidates")
	}
	// Whether the selected member is in the designated candidate
	for _, mem := range groupInitInfo.GroupMembers {
		find := false
		for _, cand := range candidates {
			if mem.IsEqual(cand) {
				find = true
				break
			}
		}
		if !find {
			return nil, false, fmt.Errorf("mem error: %v is not a legal candidate", mem.ShortS())
		}
	}
	return candidates, true, nil
}

// OnMessageSharePiece handles sharepiece message received from other members during the group formation process.
func (p *groupCreateProcessor) OnMessageSharePiece(sharePieceMessage *model.SharePieceMessage) {
	p.handleSharePieceMessage(sharePieceMessage.GroupHash, &sharePieceMessage.Share, &sharePieceMessage.SignInfo, false)
	return
}

// handleSharePieceMessage handles a piece information from other nodes
// It has two sources:
// One is that shared with each other during the group formation process.
// The other is the response obtained after actively requesting from the other party.
func (p *groupCreateProcessor) handleSharePieceMessage(groupHash common.Hash, share *model.SharePiece, signInfo *model.SignInfo, isShareReqResponse bool) (recover bool, err error) {
	groupCreateLogger.Debugf("Rcv share piece! groupHash=%v, sender=%v, isShareReqResponse=%v", groupHash.ShortS(), signInfo.GetSignerID().ShortS(), isShareReqResponse)
	defer func() {
		groupCreateLogger.Debugf("recovered sign pubkey:%v, err %v", recover, err)
	}()

	context := p.groupInitContextCache.GetContext(groupHash)
	if context == nil {
		err = fmt.Errorf("failed, receive SHAREPIECE msg but gc=nil.gHash=%v", groupHash.Hex())
		return
	}
	if context.groupInitInfo.GroupHash() != groupHash {
		err = fmt.Errorf("failed, gisHash diff")
		return
	}

	pk := access.GetMinerPubKey(signInfo.GetSignerID())
	if pk == nil {
		err = fmt.Errorf("miner pk is nil, id=%v", signInfo.GetSignerID().ShortS())
		return
	}
	if !signInfo.VerifySign(*pk) {
		err = fmt.Errorf("miner sign verify fail")
		return
	}

	groupHeader := context.groupInitInfo.GroupHeader
	topHeight := p.blockChain.TopBlock().Height

	if !isShareReqResponse && context.groupInitInfo.ReadyTimeout(topHeight) {
		err = fmt.Errorf("ready timeout, readyHeight=%v, now=%v", groupHeader.CreateHeight+model.Param.GroupReadyGap, topHeight)
		return
	}

	result := context.HandleSharePiece(signInfo.GetSignerID(), share)
	waitPieceIds := make([]string, 0)
	for _, mem := range context.groupInitInfo.GroupMembers {
		if !context.nodeInfo.hasSharePiece(mem) {
			waitPieceIds = append(waitPieceIds, mem.ShortS())
			if len(waitPieceIds) >= 10 {
				break
			}
		}
	}

	messageType := "On message share piece:"
	if isShareReqResponse {
		messageType = "On message share piece response"
	}

	// All piece collected
	if result == 1 {
		recover = true
		groupCreateLogger.Infof("Collected all share piece: groupHash=%v, cost=%v.", groupHash.ShortS(), utility.GetTime().Sub(context.createTime).String())
		joinedGroupInfo := model.NewJoindGroupInfo(context.nodeInfo.getSignSecKey(), context.nodeInfo.getGroupPubKey(), context.groupInitInfo.GroupHash())
		p.joinedGroupStorage.JoinGroup(joinedGroupInfo, p.minerInfo.ID)

		inGroupSignSecKey := joinedGroupInfo.SignSecKey
		if joinedGroupInfo.GroupPK.IsValid() && joinedGroupInfo.SignSecKey.IsValid() {
			// 1. Broadcast the group-related public key to other members
			if context.TransformStatus(GisSendSharePiece, GisSendSignPk) {
				signPubKeyMessage := &model.SignPubKeyMessage{
					GroupID:        joinedGroupInfo.GroupID,
					SignPK:         *groupsig.GeneratePubkey(joinedGroupInfo.SignSecKey),
					GroupHash:      groupHash,
					GroupMemberNum: int32(context.groupInitInfo.MemberSize()),
				}
				if !signPubKeyMessage.SignPK.IsValid() {
					panic("signPK is InValid")
				}
				if signInfo, ok := model.NewSignInfo(inGroupSignSecKey, p.minerInfo.ID, signPubKeyMessage); ok {
					signPubKeyMessage.SignInfo = signInfo
					groupCreateLogger.Debugf("(%V)Send Sign PubKey.Group id:%s", p.minerInfo.ID.ShortS(), joinedGroupInfo.GroupID.ShortS())
					p.NetServer.SendSignPubKey(signPubKeyMessage)
				} else {
					err = fmt.Errorf("genSign fail, id=%v, sk=%v", p.minerInfo.ID.ShortS(), p.minerInfo.SecKey.ShortS())
					return
				}
			}
			// 2. Broadcast the complete group information that has been initialized
			if !isShareReqResponse && context.TransformStatus(GisSendSignPk, GisSendInited) {
				groupInitedMessage := &model.GroupInitedMessage{
					GroupHash:       groupHash,
					GroupPK:         joinedGroupInfo.GroupPK,
					GroupID:         joinedGroupInfo.GroupID,
					CreateHeight:    groupHeader.CreateHeight,
					ParentGroupSign: context.groupInitInfo.ParentGroupSign,
					MemberNum:       int32(context.groupInitInfo.MemberSize()),
					MemberMask:      context.generateMemberMask(),
				}

				if signInfo, ok := model.NewSignInfo(p.minerInfo.SecKey, p.minerInfo.ID, groupInitedMessage); ok {
					groupInitedMessage.SignInfo = signInfo
					groupCreateLogger.Debugf("Broadcast group inited message:%v", joinedGroupInfo.GroupID.ShortS())
					p.NetServer.BroadcastGroupInfo(groupInitedMessage)
				} else {
					err = fmt.Errorf("genSign fail, id=%v, sk=%v", p.minerInfo.ID.ShortS(), p.minerInfo.SecKey.ShortS())
					return
				}
			}
		} else {
			err = fmt.Errorf("%v failed, aggr key error", messageType)
			return
		}
	}
	return
}

// OnMessageSignPK handles group-related public key messages received from other members
// Simply stores the public key for future use
func (p *groupCreateProcessor) OnMessageSignPK(signPubKeyMessage *model.SignPubKeyMessage) {
	groupCreateLogger.Debugf("(%v)Rcv sign pubkey , sender=%v, groupHash=%v, groupId=%v...", p.minerInfo.ID.ShortS(), signPubKeyMessage.SignInfo.GetSignerID().ShortS(), signPubKeyMessage.GroupHash.ShortS(), signPubKeyMessage.GroupID.ShortS())
	if signPubKeyMessage.GenHash() != signPubKeyMessage.SignInfo.GetDataHash() {
		groupCreateLogger.Errorf("spkm hash diff")
		return
	}
	if !signPubKeyMessage.VerifySign(signPubKeyMessage.SignPK) {
		groupCreateLogger.Errorf("miner sign verify fail")
		return
	}

	removeSignPkRecord(signPubKeyMessage.SignInfo.GetSignerID())
	joinedGroupInfo, ret := p.joinedGroupStorage.AddMemberSignPk(signPubKeyMessage.SignInfo.GetSignerID(), signPubKeyMessage.GroupID, signPubKeyMessage.SignPK)
	if joinedGroupInfo != nil {
		groupCreateLogger.Debugf("add member sign pk result=%v,received member sign pk count=%v,", ret, joinedGroupInfo.MemberSignPKNum())
		for mem, pk := range joinedGroupInfo.GetMemberPKs() {
			groupCreateLogger.Debugf("signPKS: %v, %v", mem, pk.GetHexString())
		}
		return
	}
}

// OnMessageGroupInited is a network-wide node processing function.
// The entire network node receives a group of initialized completion messages from all of the members in the group
// and when 51% of the same message received from the group members, the group will be added on chain
func (p *groupCreateProcessor) OnMessageGroupInited(msg *model.GroupInitedMessage) {
	groupHash := msg.GroupHash

	groupCreateLogger.Debugf("(%v)Rcv group inited message!sender=%v, groupHash=%v, groupId=%v, groupPK=%v,member mask:%v", p.minerInfo.ID.ShortS(),
		msg.SignInfo.GetSignerID().ShortS(), groupHash.ShortS(), msg.GroupID.ShortS(), msg.GroupPK.ShortS(), msg.MemberMask)

	if msg.SignInfo.GetDataHash() != msg.GenHash() {
		panic("grm gis hash diff")
	}

	// The group already added on chain before because of synchronization process
	g := p.groupChain.GetGroupById(msg.GroupID.Serialize())
	if g != nil {
		groupCreateLogger.Debugf("group already on chain")
		p.removeGroupPubkeyCollector(groupHash)
		p.groupInitContextCache.Clean(groupHash)
		return
	}

	pk := access.GetMinerPubKey(msg.SignInfo.GetSignerID())
	if !msg.VerifySign(*pk) {
		groupCreateLogger.Errorf("verify sign fail, id=%v, pk=%v, sign=%v", msg.SignInfo.GetSignerID().ShortS(), pk.GetHexString(), msg.SignInfo.GetSignature().GetHexString())
		return
	}

	groupPubkeyCollector := p.getGroupPubkeyCollector(msg.GroupHash)
	if groupPubkeyCollector == nil {
		groupInitInfo, err := p.recoverGroupInitInfo(msg.CreateHeight, msg.MemberMask)
		if err != nil {
			groupCreateLogger.Errorf("recover group info fail, err %v", err)
			return
		}
		if groupInitInfo.GroupHash() != msg.GroupHash {
			groupCreateLogger.Errorf("groupHeader hash error, expect %v, receive %v", groupInitInfo.GroupHash().Hex(), msg.GroupHash.Hex())
			return
		}
		groupInitInfo.ParentGroupSign = msg.ParentGroupSign
		groupPubkeyCollector = NewGroupPubkeyCollector(groupInitInfo)
		groupCreateLogger.Debugf("new groupPubkeyCollector")
	}

	groupInitInfo := groupPubkeyCollector.groupInitInfo

	// Check the time window, deny messages out of date
	if groupInitInfo.ReadyTimeout(p.blockChain.Height()) {
		groupCreateLogger.Warnf("group ready timeout, gid=%v", msg.GroupID.ShortS())
		return
	}

	parentID := groupInitInfo.ParentGroupID()
	parentGroup := p.getGroupInfo(parentID)

	gpk := parentGroup.GroupPK
	if !groupsig.VerifySig(gpk, msg.GroupHash.Bytes(), msg.ParentGroupSign) {
		groupCreateLogger.Errorf("verify parent groupsig fail! gHash=%v", groupHash.ShortS())
		return
	}
	if !groupInitInfo.ParentGroupSign.IsEqual(msg.ParentGroupSign) {
		groupCreateLogger.Errorf("signature differ, old %v, new %v", groupInitInfo.ParentGroupSign.GetHexString(), msg.ParentGroupSign.GetHexString())
		return
	}
	groupPubkeyCollector = p.addGroupPubkeyCollector(groupPubkeyCollector)

	result := groupPubkeyCollector.handleGroupSign(msg.SignInfo.GetSignerID(), msg.GroupPK)

	waitIds := make([]string, 0)
	for _, mem := range groupInitInfo.GroupMembers {
		if !groupPubkeyCollector.hasReceived(mem) {
			waitIds = append(waitIds, mem.ShortS())
			if len(waitIds) >= 10 {
				break
			}
		}
	}

	groupCreateLogger.Debugf("Group inited message received %v, required %v, missing list:%v etc.handle group pubkey result:%v,", groupPubkeyCollector.receiveGroupPKCount(), groupPubkeyCollector.threshold, waitIds, result)
	switch result {
	case InitSuccess: // Receive the same message in the group >= threshold, can add on chain
		staticGroup := model.NewGroupInfo(msg.GroupID, msg.GroupPK, groupInitInfo)
		gh := staticGroup.GetGroupHeader()
		groupCreateLogger.Debugf("SUCCESS accept a new group, groupHash=%v, groupId=%v, workHeight=%v, dismissHeight=%v.", groupHash.ShortS(), msg.GroupID.ShortS(), gh.WorkHeight, gh.DismissHeight)

		p.addGroupOnChain(staticGroup)
		p.removeGroupPubkeyCollector(groupHash)
		p.groupInitContextCache.Clean(groupHash)
	case InitFail: // The group is initialized abnormally and cannot be recovered
		groupCreateLogger.Debugf("initialization failed")
		p.removeGroupPubkeyCollector(groupHash)
	}
	return
}

// recoverGroupInitInfo recover group info from mask
func (p *groupCreateProcessor) recoverGroupInitInfo(baseHeight uint64, mask []byte) (*model.GroupInitInfo, error) {
	ctx, err := p.genCreateGroupBaseInfo(baseHeight)
	if err != nil {
		return nil, err
	}
	return ctx.createGroupInitInfo(mask), nil
}

func (p *groupCreateProcessor) addGroupOnChain(groupInfo *model.GroupInfo) {
	group := convertToGroup(groupInfo)
	groupCreateLogger.Infof("addGroupOnChain height:%d,id:%s\n", group.GroupHeight, groupInfo.GroupID.ShortS())

	var err error
	defer func() {
		var s string
		if err != nil {
			s = err.Error()
		}
		groupCreateLogger.Debugf("AddGroupOnChain! groupId=%v, workHeight=%v, result %v", groupInfo.GroupID.ShortS(), group.Header.WorkHeight, s)
	}()

	if p.groupChain.GetGroupById(group.Id) != nil {
		groupCreateLogger.Debugf("group already onchain, accept, id=%v\n", groupInfo.GroupID.ShortS())

		msg := notify.GroupMessage{Group: *convertToGroup(groupInfo)}
		notify.BUS.Publish(notify.AcceptGroup, &msg)
		err = fmt.Errorf("group already onchain")
	} else {
		top := p.blockChain.Height()
		if !groupInfo.GetReadyTimeout(top) {
			err1 := p.groupChain.AddGroup(group)
			if err1 != nil {
				groupCreateLogger.Errorf("ERROR:add group fail! hash=%v, gid=%v, err=%v\n", group.Header.Hash.ShortS(), groupInfo.GroupID.ShortS(), err1.Error())
				err = err1
				return
			}
			err = fmt.Errorf("success")
			p.addGroupCreatedHeight(group.Header.CreateHeight)
			groupCreateLogger.Infof("addGroupOnChain success, ID=%v, height=%v\n", groupInfo.GroupID.ShortS(), p.groupChain.Count())
		} else {
			err = fmt.Errorf("ready timeout, currentHeight %v", top)
			groupCreateLogger.Infof("addGroupOnChain group ready timeout, gid %v, timeout height %v, top %v\n", groupInfo.GroupID.ShortS(), groupInfo.GroupInitInfo.GroupHeader.CreateHeight+model.Param.GroupReadyGap, top)
		}
	}

}

// addHeightCreated
func (p *groupCreateProcessor) addGroupCreatedHeight(h uint64) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	p.createdHeights[p.createdHeightsIndex] = h
	p.createdHeightsIndex = (p.createdHeightsIndex + 1) % len(p.createdHeights)
}

// getGroupPubKey get the public key of an ingot group (loaded from
// the chain when the processer is initialized)
func (p *groupCreateProcessor) getGroupPubKey(groupId groupsig.ID) groupsig.Pubkey {
	if g, err := p.groupAccessor.GetGroupByID(groupId); err != nil {
		panic("GetSelfGroup failed.")
	} else {
		return g.GetGroupPubKey()
	}

}

// GetGroup get a specific group
func (p *groupCreateProcessor) getGroupInfo(gid groupsig.ID) *model.GroupInfo {
	if g, err := p.groupAccessor.GetGroupByID(gid); err != nil {
		panic("GetSelfGroup failed.")
	} else {
		return g
	}
}
