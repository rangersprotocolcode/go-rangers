package core

import (
	"bytes"
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/notify"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/network"
)

func (p *syncProcessor) requestGroupChainPiece(targetNode string, localHeight uint64) {
	req := groupChainPieceReq{Height: localHeight}
	req.SignInfo = common.NewSignData(p.privateKey, p.id, &req)

	body, e := marshalGroupChainPieceReq(req)
	if e != nil {
		p.logger.Errorf("marshal group chain piece req error:%s", e.Error())
		return
	}
	p.logger.Debugf("req group chain piece to %s, local group height:%d", targetNode, localHeight)
	message := network.Message{Code: network.GroupChainPieceReqMsg, Body: body}
	network.GetNetInstance().SendToStranger(common.FromHex(targetNode), message)
	p.reqTimer.Reset(syncReqTimeout)
}

func (p *syncProcessor) groupChainPieceReqHandler(msg notify.Message) {
	chainPieceReqMessage, ok := msg.GetData().(*notify.GroupChainPieceReqMessage)
	if !ok {
		syncHandleLogger.Errorf("GroupChainPieceReqMessage assert not ok!")
		return
	}
	chainPieceReq, e := unMarshalGroupChainPieceReq(chainPieceReqMessage.GroupChainPieceReq)
	if e != nil {
		syncHandleLogger.Errorf("Discard message! GroupChainPieceReqMessage unmarshal error:%s", e.Error())
		return
	}
	err := chainPieceReq.SignInfo.ValidateSign(chainPieceReq)
	if err != nil {
		syncHandleLogger.Errorf("Sign verify error! GroupChainPieceReqMessage:%s", e.Error())
		return
	}

	from := chainPieceReq.SignInfo.Id
	syncHandleLogger.Debugf("Rcv group chain piece req from:%s,req height:%d", from, chainPieceReq.Height)
	chainPiece := p.groupChain.getGroupChainPiece(chainPieceReq.Height)

	chainPieceMsg := groupChainPiece{GroupChainPiece: chainPiece}
	chainPieceMsg.SignInfo = common.NewSignData(p.privateKey, p.id, &chainPieceMsg)
	syncHandleLogger.Debugf("Send group chain piece  %d-%d to:%s", chainPiece[0].GroupHeight, chainPiece[len(chainPiece)-1].GroupHeight, from)
	body, e := marshalGroupChainPiece(chainPieceMsg)
	if e != nil {
		syncHandleLogger.Errorf("Marshal group chain piece error:%s!", e.Error())
		return
	}
	message := network.Message{Code: network.GroupChainPieceMsg, Body: body}
	network.GetNetInstance().SendToStranger(common.FromHex(from), message)
}

func (p *syncProcessor) groupChainPieceHandler(msg notify.Message) {
	chainPieceInfoMessage, ok := msg.GetData().(*notify.GroupChainPieceMessage)
	if !ok {
		p.logger.Errorf("GroupChainPieceMessage assert not ok!")
		return
	}
	chainPieceInfo, err := unMarshalGroupChainPiece(chainPieceInfoMessage.GroupChainPieceByte)
	if err != nil {
		p.logger.Errorf("Discard message! GroupChainPieceMessage unmarshal error:%s", err.Error())
		return
	}

	err = chainPieceInfo.SignInfo.ValidateSign(chainPieceInfo)
	if err != nil {
		p.logger.Errorf("Sign verify error! GroupChainPieceMessage:%s", err.Error())
		return
	}

	from := chainPieceInfo.SignInfo.Id
	if from != p.candidateInfo.Id {
		p.logger.Debugf("[GroupChainPieceMessage]Unexpected candidate! Expect from:%s, actual:%s,!", p.candidateInfo.Id, from)
		PeerManager.markEvil(from)
		return
	}
	p.reqTimer.Stop()

	chainPiece := chainPieceInfo.GroupChainPiece
	p.logger.Debugf("Rcv group chain piece from:%s,%d-%d", p.candidateInfo.Id, chainPiece[0].GroupHeight, chainPiece[len(chainPiece)-1].GroupHeight)
	if !verifyGroupChainPieceInfo(chainPiece) {
		p.logger.Debugf("Illegal group chain piece!", from)
		p.finishCurrentSync(false)
		return
	}

	//index bigger,height bigger
	var commonAncestor *types.Group
	for i := 0; i < len(chainPiece); i++ {
		height := chainPiece[i].GroupHeight
		group := p.groupChain.getGroupByHeight(height)
		if group == nil {
			syncLogger.Errorf("Group chain get nil group!Height:%d", height)
			p.finishCurrentSync(true)
			return
		}
		if group.Header.Hash != chainPiece[i].Header.Hash {
			break
		}
		commonAncestor = chainPiece[i]
	}

	if commonAncestor == nil {
		if chainPiece[0].GroupHeight == 0 {
			p.logger.Error("Genesis block is different.Can not sync!")
			p.finishCurrentSync(true)
			return
		}
		p.logger.Debugf("Do not find group common ancestor.Req:%d", chainPiece[len(chainPiece)-1].GroupHeight)
		go p.requestGroupChainPiece(from, chainPiece[len(chainPiece)-1].GroupHeight)
		return
	}
	p.logger.Debugf("Common ancestor group.height:%d,hash:%s", commonAncestor.GroupHeight, commonAncestor.Header.Hash.String())

	commonAncestorGroup := p.groupChain.GetGroupById(commonAncestor.Id)
	if commonAncestorGroup == nil {
		p.logger.Error("GroupChain get common ancestor nil! Height:%d,Hash:%s", commonAncestor.GroupHeight, commonAncestor.Header.Hash.String())
		p.finishCurrentSync(true)
		return
	}
	go p.syncGroup(from, commonAncestorGroup)
}

func verifyGroupChainPieceInfo(chainPiece []*types.Group) bool {
	if len(chainPiece) == 0 {
		return false
	}

	//can not verify top header group sign
	for i := 0; i < len(chainPiece)-1; i++ {
		group := chainPiece[i]
		if group == nil {
			return false
		}

		if i > 0 && !bytes.Equal(group.Header.PreGroup, chainPiece[i-1].Id) {
			return false
		}

		//todo 创始块组签名没写
		//if bh.Height > 0 {
		//	signVerifyResult, _ := consensusHelper.VerifyBlockHeader(bh)
		//	if !signVerifyResult {
		//		return false
		//	}
		//}
	}
	return true
}

func (p *syncProcessor) syncGroup(id string, commonAncestor *types.Group) {
	p.lock.Lock("syncGroup")
	if p.groupFork == nil {
		p.groupFork = newGroupChainFork(commonAncestor)
	}
	p.lock.Unlock("syncGroup")

	syncHeight := commonAncestor.GroupHeight + 1
	p.logger.Debugf("Sync group from:%s,reqHeight:%d", id, syncHeight)
	req := groupSyncReq{Height: syncHeight}
	req.SignInfo = common.NewSignData(p.privateKey, p.id, &req)

	body, e := marshalGroupSyncReq(req)
	if e != nil {
		p.logger.Errorf("marshal group req error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.ReqGroupMsg, Body: body}
	go network.GetNetInstance().SendToStranger(common.FromHex(id), message)
	p.reqTimer.Reset(syncReqTimeout)
}

func (p *syncProcessor) syncGroupReqHandler(msg notify.Message) {
	m, ok := msg.(*notify.GroupReqMessage)
	if !ok {
		syncHandleLogger.Errorf("GroupReqMessage assert not ok!")
		return
	}
	req, err := unMarshalGroupSyncReq(m.ReqInfoByte)
	if err != nil {
		syncHandleLogger.Errorf("Discard message! GroupReqMessage unmarshal error:%s", err.Error())
		return
	}
	err = req.SignInfo.ValidateSign(req)
	if err != nil {
		syncHandleLogger.Errorf("Sign verify error! GroupReqMessage:%s", err.Error())
		return
	}

	reqHeight := req.Height
	localHeight := p.groupChain.height()
	syncHandleLogger.Debugf("Rcv group request from %s.reqHeight:%d,localHeight:%d", req.SignInfo.Id, reqHeight, localHeight)

	group := p.groupChain.getGroupByHeight(reqHeight)
	if group == nil {
		syncHandleLogger.Errorf("Group chain get nil group!Height:%d", reqHeight)
		return
	}
	isLastGroup := false
	if reqHeight == localHeight {
		isLastGroup = true
	}
	response := groupMsgResponse{Group: group, IsLastGroup: isLastGroup}
	response.SignInfo = common.NewSignData(p.privateKey, p.id, &response)
	body, e := marshalGroupMsgResponse(response)
	if e != nil {
		syncHandleLogger.Errorf("Marshal group msg response error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.GroupResponseMsg, Body: body}
	network.GetNetInstance().SendToStranger(common.FromHex(req.SignInfo.Id), message)
	syncHandleLogger.Debugf("Send group %d to %s,last:%v", group.GroupHeight, req.SignInfo.Id, isLastGroup)
}

func (p *syncProcessor) groupResponseMsgHandler(msg notify.Message) {
	m, ok := msg.(*notify.GroupResponseMessage)
	if !ok {
		p.logger.Errorf("GroupResponseMessage assert not ok!")
		return
	}
	groupResponse, err := unMarshalGroupMsgResponse(m.GroupResponseByte)
	if err != nil {
		p.logger.Errorf("Discard message! GroupResponseMessage unmarshal error:%s", err.Error())
		return
	}
	err = groupResponse.SignInfo.ValidateSign(groupResponse)
	if err != nil {
		p.logger.Errorf("Sign verify error! GroupResponseMessage:%s", err.Error())
		return
	}
	from := groupResponse.SignInfo.Id
	if from != p.candidateInfo.Id {
		p.logger.Debugf("[GroupResponseMessage]Unexpected candidate! Expect from:%s, actual:%s,!", p.candidateInfo.Id, from)
		return
	}
	group := groupResponse.Group
	p.logger.Debugf("Rcv synced group.ID:%s,Height:%d.Pre:%s", common.ToHex(group.Id), group.GroupHeight, common.ToHex(group.Header.PreGroup))
	p.reqTimer.Stop()

	if p.groupFork == nil {
		return
	}
	needMore := p.groupFork.rcv(group, groupResponse.IsLastGroup)
	if needMore {
		p.syncGroup(from, group)
	} else {
		p.triggerGroupOnFork()
	}
}

func (p *syncProcessor) triggerGroupOnFork() {
	err, rcvLastGroup, group := p.groupFork.triggerOnFork(p.blockFork)
	if err == common.ErrCreateBlockNil {
		p.triggerOnFork(false)
		return
	}
	if err == verifyGroupErr {
		p.finishCurrentSync(false)
		return
	}

	if p.blockFork == nil {
		result := p.groupFork.triggerOnChain(p.groupChain)
		p.finishCurrentSync(result)
		return
	}

	if !rcvLastGroup && group != nil {
		p.syncGroup(p.candidateInfo.Id, group)
		return
	}
	p.triggerBlockOnFork()
}
