package logical

import (
	"sync"
	"time"

	"fmt"
	"x/src/consensus/groupsig"
	"x/src/consensus/model"
	"x/src/consensus/access"
)

// OnMessageSharePieceReq receives share piece request from other members
// It happens in the case that the current node didn't heard from the other part during the piece-sharing with each other process.
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
		Share: piece,
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

func (p *groupCreateProcessor) askSignPK(minerId groupsig.ID,groupId groupsig.ID) {
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
	groupCreateLogger.Debugf("Rcv sign pk req! sender:%s",sender)
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
		GroupHash:   joinedGroupInfo.GroupHash,
		GroupID: msg.GroupID,
		SignPK:  *groupsig.GeneratePubkey(joinedGroupInfo.SignSecKey),
	}
	if signInfo, ok := model.NewSignInfo(p.minerInfo.SecKey, p.minerInfo.ID, msg); ok {
		resp.SignInfo = signInfo
		groupCreateLogger.Debugf("answer signPKReq Message, receiver %v, gid %v", sender.ShortS(), msg.GroupID.ShortS())
		p.NetServer.AnswerSignPkMessage(resp, sender)
	} else {
		err = fmt.Errorf("gen Sign fail, ski=%v,%v", p.minerInfo.ID.ShortS(), p.minerInfo.SecKey.GetHexString())
	}
}



type signPKReqRecord struct {
	reqTime time.Time
	reqMinerID  groupsig.ID
}

func (r *signPKReqRecord) reqTimeout() bool {
	return time.Now().After(r.reqTime.Add(60 * time.Second))
}

//recordMap mapping idHex to signPKReqRecord
var recordMap sync.Map

func addSignPkReq(id groupsig.ID) bool {
	r := &signPKReqRecord{
		reqTime: time.Now(),
		reqMinerID:  id,
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
