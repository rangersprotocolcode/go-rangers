// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package net

import (
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"com.tuntun.rocket/node/src/consensus/model"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/network"
	"com.tuntun.rocket/node/src/utility"
	"time"
)

type NetworkServerImpl struct {
	net network.Network
}

func NewNetworkServer() NetworkServer {
	return &NetworkServerImpl{
		net: network.GetNetInstance(),
	}
}

//====================================建组前共识=======================
func (ns *NetworkServerImpl) SendGroupPingMessage(msg *model.CreateGroupPingMessage, receiver groupsig.ID) {
	body, e := marshalCreateGroupPingMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send SendGroupPingMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.GroupPing, Body: body}

	ns.net.SendToStranger(receiver.Serialize(), m)
}

func (ns *NetworkServerImpl) SendGroupPongMessage(msg *model.CreateGroupPongMessage, groupId string, belongGroup bool) {
	body, e := marshalCreateGroupPongMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send SendGroupPongMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.GroupPong, Body: body}
	ns.net.SpreadToGroup(groupId, m)
	if belongGroup {
		ns.send2Self(msg.GetSignerID(), m)
	}
	logger.Debug("SendGroupPongMessage to group:%s,ping id:%d", groupId, msg.PingID)
}

func (ns *NetworkServerImpl) SendCreateGroupSignMessage(msg *model.ParentGroupConsensusSignMessage, parentGid groupsig.ID) {
	body, e := marshalConsensusCreateGroupSignMessage(msg)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusCreateGroupSignMessage because of marshal error:%s", e.Error())
		return
	}

	m := network.Message{Code: network.CreateGroupSign, Body: body}
	go ns.net.SendToStranger(msg.Launcher.Serialize(), m)
}

//开始建组
func (ns *NetworkServerImpl) SendCreateGroupRawMessage(msg *model.ParentGroupConsensusMessage, belongGroup bool) {
	body, e := marshalConsensusCreateGroupRawMessage(msg)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusCreateGroupRawMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.CreateGroupaRaw, Body: body}

	var groupId = msg.GroupInitInfo.ParentGroupID()
	go ns.net.SpreadToGroup(groupId.GetHexString(), m)
	if belongGroup {
		ns.send2Self(msg.GetSignerID(), m)
	}
}

//----------------------------------------------------组初始化-----------------------------------------------------------
//广播 组初始化消息  全网广播
func (ns *NetworkServerImpl) SendGroupInitMessage(grm *model.GroupInitMessage) {
	body, e := marshalConsensusGroupRawMessage(grm)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusGroupRawMessage because of marshal error:%s", e.Error())
		return
	}

	m := network.Message{Code: network.GroupInitMsg, Body: body}
	//目标组还未建成，需要点对点发送
	for _, mem := range grm.GroupInitInfo.GroupMembers {
		logger.Debugf("%v SendGroupInitMessage gHash %v to %v", grm.SignInfo.GetSignerID().GetHexString(), grm.GroupInitInfo.GroupHash().Hex(), mem.GetHexString())
		ns.net.SendToStranger(mem.Serialize(), m)
	}
	//logger.Debugf("SendGroupInitMessage hash:%s,  gHash %v", m.Hash(), grm.GInfo.GroupHash().Hex())
}

//组内广播密钥   for each定向发送 组内广播
func (ns *NetworkServerImpl) SendKeySharePiece(spm *model.SharePieceMessage) {
	body, e := marshalConsensusSharePieceMessage(spm)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusSharePieceMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.KeyPieceMsg, Body: body}
	if spm.SignInfo.GetSignerID().IsEqual(spm.ReceiverId) {
		go ns.send2Self(spm.SignInfo.GetSignerID(), m)
		return
	}

	begin := utility.GetTime()
	go ns.net.SendToStranger(spm.ReceiverId.Serialize(), m)
	logger.Debugf("SendKeySharePiece to id:%s,hash:%s, gHash:%v, cost time:%v", spm.ReceiverId.GetHexString(), m.Hash(), spm.GroupHash.Hex(), time.Since(begin))
}

//组内广播签名公钥
func (ns *NetworkServerImpl) SendSignPubKey(spkm *model.SignPubKeyMessage) {
	body, e := marshalConsensusSignPubKeyMessage(spkm)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send ConsensusSignPubKeyMessage because of marshal error:%s", e.Error())
		return
	}

	m := network.Message{Code: network.SignPubkeyMsg, Body: body}
	//给自己发
	ns.send2Self(spkm.SignInfo.GetSignerID(), m)

	begin := utility.GetTime()
	go ns.net.SpreadToGroup(spkm.GroupHash.Hex(), m)
	logger.Debugf("SendSignPubKey hash:%s, dummyId:%v, cost time:%v", m.Hash(), spkm.GroupHash.Hex(), time.Since(begin))
}

//组初始化完成 广播组信息 全网广播
func (ns *NetworkServerImpl) BroadcastGroupInfo(cgm *model.GroupInitedMessage) {
	body, e := marshalConsensusGroupInitedMessage(cgm)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send ConsensusGroupInitedMessage because of marshal error:%s", e.Error())
		return
	}

	m := network.Message{Code: network.GroupInitDoneMsg, Body: body}
	//给自己发
	ns.send2Self(cgm.SignInfo.GetSignerID(), m)

	go ns.net.Broadcast(m)
	logger.Debugf("Broadcast GROUP_INIT_DONE_MSG, hash:%s, gHash:%v", m.Hash(), cgm.GroupHash.Hex())

}

//-----------------------------------------------------------------组铸币----------------------------------------------

// 提案节点完成铸币，将blockheader签名后发送至验证组内节点进行验证
// 组内广播
func (ns *NetworkServerImpl) SendCastVerify(ccm *model.ConsensusCastMessage, group *GroupBrief, body []*types.Transaction) {
	var groupId groupsig.ID
	e1 := groupId.Deserialize(ccm.BH.GroupId)
	if e1 != nil {
		logger.Errorf("[peer]Discard send ConsensusCurrentMessage because of Deserialize groupsig id error::%s", e1.Error())
		return
	}
	begin := utility.GetTime()
	timeFromCast := begin.Sub(ccm.BH.CurTime)

	ccMsg, e := marshalConsensusCastMessage(ccm)
	if e != nil {
		logger.Errorf("[peer]Discard send cast verify because of marshalConsensusCastMessage error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.CastVerifyMsg, Body: ccMsg}
	go ns.net.SpreadToGroup(groupId.GetHexString(), m)
	logger.Debugf("send CAST_VERIFY_MSG,%d-%d to group:%s,invoke SpreadToGroup cost time:%v,time from cast:%v,hash:%s", ccm.BH.Height, ccm.BH.TotalQN, groupId.GetHexString(), utility.GetTime().Sub(begin), timeFromCast, ccm.BH.Hash.String())
}

// 组内节点  验证通过后 自身签名 广播验证块 组内广播
// 验证不通过 保持静默
func (ns *NetworkServerImpl) SendVerifiedCast(cvm *model.ConsensusVerifyMessage, receiver groupsig.ID) {
	body, e := marshalConsensusVerifyMessage(cvm)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusVerifyMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.VerifiedCastMsg, Body: body}

	// 验证消息需要给自己也发一份，否则自己的分片中将不包含自己的签名，导致分红没有
	go ns.send2Self(cvm.SignInfo.GetSignerID(), m)

	go ns.net.SpreadToGroup(receiver.GetHexString(), m)
	logger.Debugf("[peer]send VARIFIED_CAST_MSG,hash:%s", cvm.BlockHash.String())
	//statistics.AddBlockLog(common.BootId, statistics.SendVerified, cvm.BH.Height, cvm.BH.ProveValue.Uint64(), -1, -1,
	//	utility.GetTime().UnixNano(), "", "", common.InstanceIndex, cvm.BH.CurTime.UnixNano())
}

//对外广播经过组签名的block 全网广播
func (ns *NetworkServerImpl) BroadcastNewBlock(cbm *model.ConsensusBlockMessage) {
	//timeFromCast := time.Since(cbm.Block.Header.CurTime)
	body, e := types.MarshalBlock(&cbm.Block)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusBlockMessage because of marshal error:%s", e.Error())
		return
	}
	blockMsg := network.Message{Code: network.NewBlockMsg, Body: body}

	go ns.net.Broadcast(blockMsg)

	//core.Logger.Debugf("Broad new block %d-%d,hash:%v,tx count:%d,msg size:%d, time from cast:%v,spread over group:%s", cbm.Block.Header.Height, cbm.Block.Header.TotalQN, cbm.Block.Header.Hash.Hex(), len(cbm.Block.Header.Transactions), len(blockMsg.Body), timeFromCast, nextVerifyGroupId)
	//statistics.AddBlockLog(common.BootId, statistics.BroadBlock, cbm.Block.Header.Height, cbm.Block.Header.ProveValue.Uint64(), len(cbm.Block.Transactions), len(body),
	//	utility.GetTime().UnixNano(), "", "", common.InstanceIndex, cbm.Block.Header.CurTime.UnixNano())
}

//-----------------------------------------------------------------密钥请求----------------------------------------------
func (ns *NetworkServerImpl) AskSignPkMessage(msg *model.SignPubkeyReqMessage, receiver groupsig.ID) {
	body, e := marshalConsensusSignPubKeyReqMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send ConsensusSignPubkeyReqMessage because of marshal error:%s", e.Error())
		return
	}

	m := network.Message{Code: network.AskSignPkMsg, Body: body}

	begin := utility.GetTime()
	go ns.net.SendToStranger(receiver.Serialize(), m)
	logger.Debugf("AskSignPkMessage %v, hash:%s, cost time:%v", receiver.GetHexString(), m.Hash(), time.Since(begin))
}

func (ns *NetworkServerImpl) AnswerSignPkMessage(msg *model.SignPubKeyMessage, receiver groupsig.ID) {
	body, e := marshalConsensusSignPubKeyMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send ConsensusSignPubKeyMessage because of marshal error:%s", e.Error())
		return
	}

	m := network.Message{Code: network.AnswerSignPkMsg, Body: body}

	begin := utility.GetTime()
	go ns.net.SendToStranger(receiver.Serialize(), m)
	logger.Debugf("AnswerSignPkMessage %v, hash:%s, dummyId:%v, cost time:%v", receiver.GetHexString(), m.Hash(), msg.GroupHash.Hex(), time.Since(begin))
}

func (ns *NetworkServerImpl) ReqSharePiece(msg *model.ReqSharePieceMessage, receiver groupsig.ID) {
	body, e := marshalSharePieceReqMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send marshalSharePieceReqMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.ReqSharePiece, Body: body}

	ns.net.SendToStranger(receiver.Serialize(), m)
}

func (ns *NetworkServerImpl) ResponseSharePiece(msg *model.ResponseSharePieceMessage, receiver groupsig.ID) {
	body, e := marshalSharePieceResponseMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send marshalSharePieceResponseMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.ResponseSharePiece, Body: body}

	ns.net.SendToStranger(receiver.Serialize(), m)
}

//------------------------------------组网络管理-----------------------

func (ns *NetworkServerImpl) JoinGroupNet(groupId string) {
	ns.net.JoinGroupNet(groupId)
}

func (ns *NetworkServerImpl) ReleaseGroupNet(groupId string) {
	ns.net.QuitGroupNet(groupId)
}

func (ns *NetworkServerImpl) send2Self(self groupsig.ID, m network.Message) {
	go MessageHandler.Handle(self.GetHexString(), m)
}
