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

package core

import (
	"bytes"
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/middleware/notify"
	middleware_pb "com.tuntun.rangers/node/src/middleware/pb"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/network"
	"github.com/golang/protobuf/proto"
	"strconv"
)

type chainInfo struct {
	TotalQn        uint64
	TopBlockHash   common.Hash
	TopBlockHeight uint64
	PreHash        common.Hash

	TopGroupHeight uint64
	SignInfo       common.SignData
}

type blockChainPieceReq struct {
	Height   uint64
	SignInfo common.SignData
}

type blockChainPiece struct {
	ChainPiece []*types.BlockHeader
	TopHeader  *types.BlockHeader
	SignInfo   common.SignData
}

type blockSyncReq struct {
	Height   uint64
	SignInfo common.SignData
}

type blockMsgResponse struct {
	Block       *types.Block
	IsLastBlock bool
	SignInfo    common.SignData
}

type groupSyncReq struct {
	Height   uint64
	SignInfo common.SignData
}

type groupMsgResponse struct {
	Group       *types.Group
	IsLastGroup bool
	SignInfo    common.SignData
}

func (p *syncProcessor) requestBlockChainPiece(targetNode string, reqHeight uint64) {
	req := blockChainPieceReq{Height: reqHeight}
	req.SignInfo = common.NewSignData(p.privateKey, p.id, &req)

	body, e := marshalBlockChainPieceReq(req)
	if e != nil {
		p.logger.Errorf("marshal block chain piece req error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.BlockChainPieceReqMsg, Body: body}
	network.GetNetInstance().SendToStranger(common.FromHex(targetNode), message)
	p.blockReqTimer.Reset(syncReqTimeout)
	p.logger.Debugf("req block chain piece from %s,height:%d", targetNode, reqHeight)
}

func (p *syncProcessor) blockChainPieceReqHandler(m notify.Message) {
	msg, ok := m.GetData().(*notify.BlockChainPieceReqMessage)
	if !ok {
		syncHandleLogger.Errorf("BlockChainPieceReqMessage assert not ok!")
		return
	}
	chainPieceReq, e := unMarshalBlockChainPieceReq(msg.BlockChainPieceReq)
	if e != nil {
		syncHandleLogger.Errorf("Discard message! BlockChainPieceReqMessage unmarshal error:%s", e.Error())
		return
	}
	err := chainPieceReq.SignInfo.ValidateSign(chainPieceReq)
	if err != nil {
		syncHandleLogger.Errorf("Sign verify error! BlockChainPieceReqMessage:%s", e.Error())
		return
	}

	from := chainPieceReq.SignInfo.Id
	syncHandleLogger.Debugf("Rcv block chain piece req from:%s,source height:%d", from, chainPieceReq.Height)
	chainPiece := p.blockChain.getChainPiece(chainPieceReq.Height)
	chainPieceMsg := blockChainPiece{ChainPiece: chainPiece, TopHeader: p.blockChain.TopBlock()}
	chainPieceMsg.SignInfo = common.NewSignData(p.privateKey, p.id, &chainPieceMsg)

	if len(chainPiece) > 0 {
		syncHandleLogger.Debugf("Send chain piece %d-%d to:%s", chainPiece[0].Height, chainPiece[len(chainPiece)-1].Height, from)
	}
	body, e := marshalBlockChainPiece(chainPieceMsg)
	if e != nil {
		syncHandleLogger.Errorf("Marshal block chain piece error:%s!", e.Error())
		return
	}
	message := network.Message{Code: network.BlockChainPieceMsg, Body: body}
	network.GetNetInstance().SendToStranger(common.FromHex(from), message)
}

func (p *syncProcessor) blockChainPieceHandler(m notify.Message) {
	msg, ok := m.GetData().(*notify.BlockChainPieceMessage)
	if !ok {
		p.logger.Errorf("BlockChainPieceMessage assert not ok!")
		return
	}
	chainPieceInfo, err := unMarshalBlockChainPiece(msg.BlockChainPieceByte)
	if err != nil {
		p.logger.Errorf("Discard message! BlockChainPieceMessage unmarshal error:%s", err.Error())
		return
	}

	err = chainPieceInfo.SignInfo.ValidateSign(chainPieceInfo)
	if err != nil {
		p.logger.Errorf("Sign verify error! BlockChainPieceMessage:%s", err.Error())
		return
	}

	from := chainPieceInfo.SignInfo.Id
	candidateId := p.GetCandidateInfo().Id
	if from != candidateId {
		p.logger.Debugf("[BlockChainPieceMessage]Unexpected candidate! Expect from:%s, actual:%s,!", candidateId, from)
		PeerManager.markEvil(from)
		return
	}
	p.blockReqTimer.Stop()

	chainPiece := chainPieceInfo.ChainPiece
	if 0 != len(chainPiece) {
		p.logger.Debugf("Rcv block chain piece from:%s,%d-%d", candidateId, chainPiece[0].Height, chainPiece[len(chainPiece)-1].Height)
	}
	if !verifyBlockChainPiece(chainPiece, chainPieceInfo.TopHeader) {
		p.logger.Debugf("Illegal block chain piece!", from)
		p.finishCurrentSyncWithLock(false)
		return
	}

	localTopHeader := p.blockChain.TopBlock()
	if chainPieceInfo.TopHeader.TotalQN < localTopHeader.TotalQN {
		p.finishCurrentSyncWithLock(true)
		return
	}

	//index bigger,height bigger
	var commonAncestor *types.BlockHeader
	for i := 0; i < len(chainPiece); i++ {
		height := chainPiece[i].Height
		if p.blockChain.GetBlockHash(height) != chainPiece[i].Hash {
			break
		}
		commonAncestor = chainPiece[i]
	}

	if commonAncestor == nil {
		if chainPiece[0].Height == 0 {
			p.logger.Error("Genesis block is different.Can not sync!")
			p.finishCurrentSyncWithLock(true)
			return
		}
		p.logger.Debugf("Do not find block common ancestor.Req:%d", chainPiece[0].Height)
		go p.requestBlockChainPiece(from, chainPiece[0].Height)
		return
	}
	p.logger.Debugf("Common ancestor block. height:%d,hash:%s", commonAncestor.Height, commonAncestor.Hash.String())

	commonAncestorBlock := p.blockChain.queryBlockByHash(commonAncestor.Hash)
	if commonAncestorBlock == nil {
		p.logger.Error("Chain get common ancestor nil! Height:%d,Hash:%s", commonAncestor.Height, commonAncestor.Hash.String())
		p.finishCurrentSyncWithLock(true)
		return
	}
	go p.startSync(from, *commonAncestorBlock)
}

func verifyBlockChainPiece(chainPiece []*types.BlockHeader, topHeader *types.BlockHeader) bool {
	if len(chainPiece) == 0 || topHeader == nil {
		return false
	}

	for i := 0; i < len(chainPiece)-1; i++ {
		bh := chainPiece[i]
		if bh == nil {
			return false
		}

		if i > 0 && bh.PreHash != chainPiece[i-1].Hash {
			return false
		}

		if bh.Height > 0 {
			signVerifyResult, _ := consensusHelper.VerifyBlockHeader(bh)
			if !signVerifyResult {
				return false
			}
		}
	}
	return true
}

func (p *syncProcessor) startSync(id string, commonAncestorBlock types.Block) {
	p.lock.Lock("syncBlock")
	defer p.lock.Unlock("syncBlock")
	p.blockFork = newBlockChainFork(commonAncestorBlock)

	commonAncestorGroup := p.groupChain.getFirstGroupBelowHeight(commonAncestorBlock.Header.Height)
	if commonAncestorGroup == nil {
		p.logger.Error("Common ancestor group is nil!")
		p.finishCurrentSync(false)
		return
	}
	p.logger.Debugf("Common ancestor group %s,height:%d,createBlock height:%d", common.ToHex(commonAncestorGroup.Id), commonAncestorGroup.GroupHeight, commonAncestorGroup.Header.CreateHeight)
	p.groupFork = newGroupChainFork(commonAncestorGroup)

	go p.syncBlock(id, *commonAncestorBlock.Header)
	go p.syncGroup(id, commonAncestorGroup)
}

func (p *syncProcessor) syncBlock(id string, commonAncestor types.BlockHeader) {
	syncHeight := commonAncestor.Height + 1
	p.logger.Debugf("Sync block from:%s,reqHeight:%d", id, syncHeight)
	req := blockSyncReq{Height: syncHeight}
	req.SignInfo = common.NewSignData(p.privateKey, p.id, &req)

	body, e := marshalBlockSyncReq(req)
	if e != nil {
		p.logger.Errorf("marshal block req error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.ReqBlockMsg, Body: body}
	go network.GetNetInstance().SendToStranger(common.FromHex(id), message)
	p.blockReqTimer.Reset(syncReqTimeout)
}

func (p *syncProcessor) syncBlockReqHandler(msg notify.Message) {
	m, ok := msg.(*notify.BlockReqMessage)
	if !ok {
		syncHandleLogger.Errorf("BlockReqMessage assert not ok!")
		return
	}
	req, err := unMarshalBlockSyncReq(m.ReqInfoByte)
	if err != nil {
		syncHandleLogger.Errorf("Discard message! BlockReqMessage unmarshal error:%s", err.Error())
		return
	}
	err = req.SignInfo.ValidateSign(req)
	if err != nil {
		syncHandleLogger.Errorf("Sign verify error! BlockReqMessage:%s", err.Error())
		return
	}
	reqHeight := req.Height
	localHeight := blockChainImpl.Height()
	syncHandleLogger.Debugf("Rcv block request from %s.reqHeight:%d,localHeight:%d", req.SignInfo.Id, reqHeight, localHeight)

	block := p.blockChain.QueryBlock(reqHeight)
	isLastBlock := false
	if reqHeight >= localHeight {
		isLastBlock = true
	}
	response := blockMsgResponse{Block: block, IsLastBlock: isLastBlock}
	response.SignInfo = common.NewSignData(p.privateKey, p.id, &response)
	body, e := marshalBlockMsgResponse(response)
	if e != nil {
		syncHandleLogger.Errorf("Marshal block msg response error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.BlockResponseMsg, Body: body}
	network.GetNetInstance().SendToStranger(common.FromHex(req.SignInfo.Id), message)
	if block != nil {
		syncHandleLogger.Debugf("Send block %d to %s,last:%v", block.Header.Height, req.SignInfo.Id, isLastBlock)
	} else {
		syncHandleLogger.Debugf("Send nil to %s,last:%v", isLastBlock)
	}
}

func (p *syncProcessor) blockResponseMsgHandler(msg notify.Message) {
	m, ok := msg.(*notify.BlockResponseMessage)
	if !ok {
		p.logger.Errorf("BlockResponseMessage assert not ok!")
		return
	}
	blockResponse, err := unMarshalBlockMsgResponse(m.BlockResponseByte)
	if err != nil {
		p.logger.Errorf("Discard message! BlockResponseMessage unmarshal error:%s", err.Error())
		return
	}
	err = blockResponse.SignInfo.ValidateSign(blockResponse)
	if err != nil {
		p.logger.Errorf("Sign verify error! BlockResponseMessage:%s", err.Error())
		return
	}
	from := blockResponse.SignInfo.Id
	candidateId := p.GetCandidateInfo().Id
	if from != candidateId {
		p.logger.Debugf("[BlockResponseMessage]Unexpected candidate! Expect from:%s, actual:%s,!", candidateId, from)
		return
	}
	block := blockResponse.Block
	if block != nil {
		p.logger.Debugf("Rcv synced block.Hash:%s,%d-%d.Pre:%s,is last:%v", block.Header.Hash.String(), block.Header.Height, block.Header.TotalQN, block.Header.PreHash.String(), blockResponse.IsLastBlock)
	} else {
		p.logger.Debugf("Rcv nil block.is last:%v", blockResponse.IsLastBlock)
		blockResponse.IsLastBlock = true
	}
	p.blockReqTimer.Stop()

	p.lock.Lock("rcv block")
	defer p.lock.Unlock("rcv block")
	if p.blockFork == nil {
		return
	}
	needMore := p.blockFork.rcv(block, blockResponse.IsLastBlock)
	if needMore {
		go p.syncBlock(from, *block.Header)
		return
	}
	if p.readyOnFork() {
		go p.triggerOnFork()
	}
}

func (p *syncProcessor) syncGroup(id string, commonAncestor *types.Group) {
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
	p.groupReqTimer.Reset(syncReqTimeout)
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
	isLastGroup := false
	if reqHeight >= localHeight {
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
	if group != nil {
		syncHandleLogger.Debugf("Send group %d to %s,last:%v", group.GroupHeight, req.SignInfo.Id, isLastGroup)
	} else {
		syncHandleLogger.Debugf("Send nil to %s,last:%v", isLastGroup)
	}
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
	candidateId := p.GetCandidateInfo().Id
	if from != candidateId {
		p.logger.Debugf("[GroupResponseMessage]Unexpected candidate! Expect from:%s, actual:%s,!", candidateId, from)
		return
	}
	group := groupResponse.Group
	if group != nil {
		p.logger.Debugf("Rcv synced group.ID:%s,Height:%d.Pre:%s,is last:%v", common.ToHex(group.Id), group.GroupHeight, common.ToHex(group.Header.PreGroup), groupResponse.IsLastGroup)
	} else {
		p.logger.Debugf("Rcv nil group.is last:%v", groupResponse.IsLastGroup)
		groupResponse.IsLastGroup = true
	}
	p.groupReqTimer.Stop()

	p.lock.Lock("rcv group")
	defer p.lock.Unlock("rcv group")
	if p.groupFork == nil {
		return
	}
	needMore := p.groupFork.rcv(group, groupResponse.IsLastGroup)
	if needMore {
		go p.syncGroup(from, group)
	}
	if p.readyOnFork() {
		go p.triggerOnFork()
	}
}

func (chainInfo *chainInfo) GenHash() common.Hash {
	buffer := bytes.Buffer{}

	buffer.Write([]byte(strconv.FormatUint(chainInfo.TotalQn, 10)))
	buffer.Write(chainInfo.TopBlockHash.Bytes())
	buffer.Write([]byte(strconv.FormatUint(chainInfo.TopBlockHeight, 10)))
	buffer.Write(chainInfo.PreHash.Bytes())
	buffer.Write([]byte(strconv.FormatUint(chainInfo.TopGroupHeight, 10)))
	return common.BytesToHash(common.Sha256(buffer.Bytes()))
}

func marshalChainInfo(bi chainInfo) ([]byte, error) {
	blockInfo := middleware_pb.ChainInfo{TopBlockHash: bi.TopBlockHash.Bytes(), TotalQn: &bi.TotalQn, TopBlockHeight: &bi.TopBlockHeight, PreHash: bi.PreHash.Bytes(), TopGroupHeight: &bi.TopGroupHeight}
	blockInfo.SignInfo = signDataToPb(bi.SignInfo)
	return proto.Marshal(&blockInfo)
}

func unMarshalChainInfo(b []byte) (*chainInfo, error) {
	message := new(middleware_pb.ChainInfo)
	e := proto.Unmarshal(b, message)
	if e != nil {
		return nil, e
	}
	blockInfo := chainInfo{TopBlockHash: common.BytesToHash(message.TopBlockHash), TotalQn: *message.TotalQn, TopBlockHeight: *message.TopBlockHeight, PreHash: common.BytesToHash(message.PreHash), TopGroupHeight: *message.TopGroupHeight}
	blockInfo.SignInfo = pbToSignData(*message.SignInfo)
	return &blockInfo, nil
}

func (chainPieceReq *blockChainPieceReq) GenHash() common.Hash {
	buffer := bytes.Buffer{}
	buffer.Write([]byte(strconv.FormatUint(chainPieceReq.Height, 10)))
	return common.BytesToHash(common.Sha256(buffer.Bytes()))
}

func marshalBlockChainPieceReq(req blockChainPieceReq) ([]byte, error) {
	chainPieceReq := middleware_pb.BlockChainPieceReq{Height: &req.Height}
	chainPieceReq.SignInfo = signDataToPb(req.SignInfo)
	return proto.Marshal(&chainPieceReq)
}

func unMarshalBlockChainPieceReq(b []byte) (*blockChainPieceReq, error) {
	message := new(middleware_pb.BlockChainPieceReq)
	e := proto.Unmarshal(b, message)
	if e != nil {
		return nil, e
	}
	chainPieceReq := blockChainPieceReq{Height: *message.Height}
	chainPieceReq.SignInfo = pbToSignData(*message.SignInfo)
	return &chainPieceReq, nil
}

func (blockChainPiece *blockChainPiece) GenHash() common.Hash {
	buffer := bytes.Buffer{}
	for _, bh := range blockChainPiece.ChainPiece {
		buffer.Write(bh.Hash.Bytes())
	}
	if blockChainPiece.TopHeader != nil {
		buffer.Write(blockChainPiece.TopHeader.Hash.Bytes())
	}
	return common.BytesToHash(common.Sha256(buffer.Bytes()))
}

func marshalBlockChainPiece(chainPieceInfo blockChainPiece) ([]byte, error) {
	headers := make([]*middleware_pb.BlockHeader, 0)
	for _, header := range chainPieceInfo.ChainPiece {
		h := types.BlockHeaderToPb(header)
		headers = append(headers, h)
	}
	topHeader := types.BlockHeaderToPb(chainPieceInfo.TopHeader)
	message := middleware_pb.BlockChainPiece{TopHeader: topHeader, BlockHeaders: headers}
	message.SignInfo = signDataToPb(chainPieceInfo.SignInfo)
	return proto.Marshal(&message)
}

func unMarshalBlockChainPiece(b []byte) (*blockChainPiece, error) {
	message := new(middleware_pb.BlockChainPiece)
	e := proto.Unmarshal(b, message)
	if e != nil {
		return nil, e
	}

	chainPiece := make([]*types.BlockHeader, 0)
	for _, header := range message.BlockHeaders {
		h := types.PbToBlockHeader(header)
		chainPiece = append(chainPiece, h)
	}
	topHeader := types.PbToBlockHeader(message.TopHeader)
	chainPieceInfo := blockChainPiece{ChainPiece: chainPiece, TopHeader: topHeader}
	chainPieceInfo.SignInfo = pbToSignData(*message.SignInfo)
	return &chainPieceInfo, nil
}

func (blockReq *blockSyncReq) GenHash() common.Hash {
	buffer := bytes.Buffer{}
	buffer.Write([]byte(strconv.FormatUint(blockReq.Height, 10)))
	return common.BytesToHash(common.Sha256(buffer.Bytes()))
}

func marshalBlockSyncReq(req blockSyncReq) ([]byte, error) {
	blockReq := middleware_pb.BlockReq{Height: &req.Height}
	blockReq.SignInfo = signDataToPb(req.SignInfo)
	return proto.Marshal(&blockReq)
}

func unMarshalBlockSyncReq(b []byte) (*blockSyncReq, error) {
	message := new(middleware_pb.BlockReq)
	e := proto.Unmarshal(b, message)
	if e != nil {
		return nil, e
	}
	blockReq := blockSyncReq{Height: *message.Height}
	blockReq.SignInfo = pbToSignData(*message.SignInfo)
	return &blockReq, nil
}

func (syncedBlockMessage *blockMsgResponse) GenHash() common.Hash {
	buffer := bytes.Buffer{}
	if syncedBlockMessage.Block != nil {
		buffer.Write(syncedBlockMessage.Block.Header.Hash.Bytes())
	}
	if syncedBlockMessage.IsLastBlock {
		buffer.Write([]byte{0})
	} else {
		buffer.Write([]byte{1})
	}
	return common.BytesToHash(common.Sha256(buffer.Bytes()))
}

func marshalBlockMsgResponse(bmr blockMsgResponse) ([]byte, error) {
	message := middleware_pb.BlockMsgResponse{IsLast: &bmr.IsLastBlock, Block: types.BlockToPb(bmr.Block)}
	message.SignInfo = signDataToPb(bmr.SignInfo)
	return proto.Marshal(&message)
}

func unMarshalBlockMsgResponse(b []byte) (*blockMsgResponse, error) {
	message := new(middleware_pb.BlockMsgResponse)
	e := proto.Unmarshal(b, message)
	if e != nil {
		return nil, e
	}
	bmr := blockMsgResponse{IsLastBlock: *message.IsLast, Block: types.PbToBlock(message.Block)}
	bmr.SignInfo = pbToSignData(*message.SignInfo)
	return &bmr, nil
}

func (groupSyncReq *groupSyncReq) GenHash() common.Hash {
	buffer := bytes.Buffer{}
	buffer.Write([]byte(strconv.FormatUint(groupSyncReq.Height, 10)))
	return common.BytesToHash(common.Sha256(buffer.Bytes()))
}

func marshalGroupSyncReq(req groupSyncReq) ([]byte, error) {
	blockReq := middleware_pb.GroupReq{Height: &req.Height}
	blockReq.SignInfo = signDataToPb(req.SignInfo)
	return proto.Marshal(&blockReq)
}

func unMarshalGroupSyncReq(b []byte) (*groupSyncReq, error) {
	message := new(middleware_pb.GroupReq)
	e := proto.Unmarshal(b, message)
	if e != nil {
		return nil, e
	}
	groupReq := groupSyncReq{Height: *message.Height}
	groupReq.SignInfo = pbToSignData(*message.SignInfo)
	return &groupReq, nil
}

func (groupMsgResponse *groupMsgResponse) GenHash() common.Hash {
	buffer := bytes.Buffer{}
	if groupMsgResponse.Group != nil {
		buffer.Write(groupMsgResponse.Group.Header.Hash.Bytes())
	}
	if groupMsgResponse.IsLastGroup {
		buffer.Write([]byte{0})
	} else {
		buffer.Write([]byte{1})
	}
	return common.BytesToHash(common.Sha256(buffer.Bytes()))
}

func marshalGroupMsgResponse(gmr groupMsgResponse) ([]byte, error) {
	message := middleware_pb.GroupMsgResponse{IsLast: &gmr.IsLastGroup, Group: types.GroupToPb(gmr.Group)}
	message.SignInfo = signDataToPb(gmr.SignInfo)
	return proto.Marshal(&message)
}

func unMarshalGroupMsgResponse(b []byte) (*groupMsgResponse, error) {
	message := new(middleware_pb.GroupMsgResponse)
	e := proto.Unmarshal(b, message)
	if e != nil {
		return nil, e
	}
	bmr := groupMsgResponse{IsLastGroup: *message.IsLast, Group: types.PbToGroup(message.Group)}
	bmr.SignInfo = pbToSignData(*message.SignInfo)
	return &bmr, nil
}

func signDataToPb(s common.SignData) *middleware_pb.SignData {
	sign := middleware_pb.SignData{DataHash: s.DataHash.Bytes(), DataSign: s.DataSign.Bytes(), SignMember: []byte(s.Id)}
	return &sign
}

func pbToSignData(s middleware_pb.SignData) common.SignData {
	sign := common.SignData{DataHash: common.BytesToHash(s.DataHash), DataSign: *common.BytesToSign(s.DataSign), Id: string(s.SignMember)}
	return sign
}
