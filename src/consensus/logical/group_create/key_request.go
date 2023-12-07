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
	"com.tuntun.rangers/node/src/utility"
	"sync"
	"time"

	"com.tuntun.rangers/node/src/consensus/access"
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/consensus/model"
	"fmt"
)

// OnMessageSharePieceReq receives share piece request from other members
// It happens in the case that the current node didn't hear from the other part during the piece-sharing with each other process.
func (p *groupCreateProcessor) OnMessageSharePieceReq(msg *model.ReqSharePieceMessage) {
	groupCreateLogger.Debugf("Rcv share piece req! groupHash=%v, sender=%v", msg.GroupHash.ShortS(), msg.SignInfo.GetSignerID().ShortS())

	pk := access.GetMinerPubKey(msg.SignInfo.GetSignerID())
	if pk == nil || !msg.VerifySign(*pk) {
		groupCreateLogger.Errorf("verify sign fail")
		return
	}
	context := p.groupInitContextCache.GetContext(msg.GroupHash)
	if context == nil {
		groupCreateLogger.Warnf("initing group context is nil")
		return
	}
	if context.sharePieceMap == nil {
		groupCreateLogger.Warnf("sharePiece map is nil")
		return
	}
	piece := context.sharePieceMap[msg.SignInfo.GetSignerID().GetHexString()]

	pieceMsg := &model.ResponseSharePieceMessage{
		GroupHash: msg.GroupHash,
		Share:     piece,
	}
	if signInfo, ok := model.NewSignInfo(p.minerInfo.SecKey, p.minerInfo.ID, pieceMsg); ok {
		pieceMsg.SignInfo = signInfo
		groupCreateLogger.Debugf("response share piece to %v, gHash=%v, share=%v", msg.SignInfo.GetSignerID().ShortS(), msg.GroupHash.ShortS(), piece.Share.ShortS())
		p.NetServer.ResponseSharePiece(pieceMsg, msg.SignInfo.GetSignerID())
	}
}

// OnMessageSharePieceResponse receives share piece message from other member after requesting
func (p *groupCreateProcessor) OnMessageSharePieceResponse(msg *model.ResponseSharePieceMessage) {
	p.handleSharePieceMessage(msg.GroupHash, &msg.Share, &msg.SignInfo, true)
	return
}

func (p *groupCreateProcessor) askSignPK(minerId groupsig.ID, groupId groupsig.ID) {
	if !addSignPkReq(minerId) {
		return
	}
	msg := &model.SignPubkeyReqMessage{
		GroupID: groupId,
	}
	if signInfo, ok := model.NewSignInfo(p.minerInfo.SecKey, p.minerInfo.ID, msg); ok {
		msg.SignInfo = signInfo
		groupCreateLogger.Debugf("ask sign pk message, receiver %v, gid %v", minerId, groupId)
		p.NetServer.AskSignPkMessage(msg, minerId)
	}
}

// OnMessageSignPKReq receives group-related public key request from other members and
// responses own public key
func (p *groupCreateProcessor) OnMessageSignPKReq(msg *model.SignPubkeyReqMessage) {
	sender := msg.SignInfo.GetSignerID()
	groupCreateLogger.Debugf("Rcv sign pk req! sender:%s", sender.GetHexString())
	var err error
	defer func() {
		groupCreateLogger.Debugf("sender=%v, gid=%v, result=%v", sender.ShortS(), msg.GroupID.ShortS(), err)
	}()

	joinedGroupInfo := p.joinedGroupStorage.GetJoinedGroupInfo(msg.GroupID)
	if joinedGroupInfo == nil {
		err = fmt.Errorf("failed, local node not found joinedGroup with group id=%v", msg.GroupID.ShortS())
		return
	}

	pk := access.GetMinerPubKey(sender)
	if pk == nil {
		err = fmt.Errorf("get minerPK is nil, id=%v", sender.ShortS())
		return
	}
	if !msg.VerifySign(*pk) {
		err = fmt.Errorf("verifySign fail, pk=%v, sign=%v", pk.GetHexString(), msg.SignInfo.GetSignature().GetHexString())
		return
	}
	if !joinedGroupInfo.SignSecKey.IsValid() {
		err = fmt.Errorf("invalid sign secKey, id=%v, sk=%v", p.minerInfo.ID.ShortS(), joinedGroupInfo.SignSecKey.ShortS())
		return
	}

	resp := &model.SignPubKeyMessage{
		GroupHash: joinedGroupInfo.GroupHash,
		GroupID:   msg.GroupID,
		SignPK:    *groupsig.GeneratePubkey(joinedGroupInfo.SignSecKey),
	}
	if signInfo, ok := model.NewSignInfo(p.minerInfo.SecKey, p.minerInfo.ID, msg); ok {
		resp.SignInfo = signInfo
		groupCreateLogger.Debugf("answer signPKReq Message, receiver %v, groupId:%v,groupHash:%v,signPK:%s,msg hash:%s", sender.ShortS(), msg.GroupID.GetHexString())
		p.NetServer.AnswerSignPkMessage(resp, sender)
	} else {
		err = fmt.Errorf("gen Sign fail, ski=%v,%v", p.minerInfo.ID.ShortS(), p.minerInfo.SecKey.GetHexString())
	}
}

type signPKReqRecord struct {
	reqTime    time.Time
	reqMinerID groupsig.ID
}

func (r *signPKReqRecord) reqTimeout() bool {
	return utility.GetTime().After(r.reqTime.Add(60 * time.Second))
}

// recordMap mapping idHex to signPKReqRecord
var recordMap sync.Map

func addSignPkReq(id groupsig.ID) bool {
	r := &signPKReqRecord{
		reqTime:    utility.GetTime(),
		reqMinerID: id,
	}
	_, load := recordMap.LoadOrStore(id.GetHexString(), r)
	return !load
}

func removeSignPkRecord(id groupsig.ID) {
	recordMap.Delete(id.GetHexString())
}

func cleanSignPkReqRecord() {
	recordMap.Range(func(key, value interface{}) bool {
		r := value.(*signPKReqRecord)
		if r.reqTimeout() {
			recordMap.Delete(key)
		}
		return true
	})
}
