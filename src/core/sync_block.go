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

package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/notify"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/network"
)

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
	p.reqTimer.Reset(syncReqTimeout)
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

	syncHandleLogger.Debugf("Send chain piece %d-%d to:%s", chainPiece[0].Height, chainPiece[len(chainPiece)-1].Height, from)
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
	if from != p.candidateInfo.Id {
		p.logger.Debugf("[BlockChainPieceMessage]Unexpected candidate! Expect from:%s, actual:%s,!", p.candidateInfo.Id, from)
		PeerManager.markEvil(from)
		return
	}
	p.reqTimer.Stop()

	chainPiece := chainPieceInfo.ChainPiece
	p.logger.Debugf("Rcv block chain piece from:%s,%d-%d", p.candidateInfo.Id, chainPiece[0].Height, chainPiece[len(chainPiece)-1].Height)
	if !verifyBlockChainPiece(chainPiece, chainPieceInfo.TopHeader) {
		p.logger.Debugf("Illegal block chain piece!", from)
		PeerManager.markEvil(from)
		p.finishCurrentSync(false)
		return
	}

	localTopHeader := p.blockChain.TopBlock()
	if chainPieceInfo.TopHeader.TotalQN < localTopHeader.TotalQN {
		p.finishCurrentSync(true)
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
			p.finishCurrentSync(true)
			return
		}
		p.logger.Debugf("Do not find block common ancestor.Req:%d", chainPiece[len(chainPiece)-1].Height)
		go p.requestBlockChainPiece(from, chainPiece[len(chainPiece)-1].Height)
		return
	}
	p.logger.Debugf("Common ancestor block. height:%d,hash:%s", commonAncestor.Height, commonAncestor.Hash.String())

	commonAncestorBlock := p.blockChain.queryBlockByHash(commonAncestor.Hash)
	if commonAncestorBlock == nil {
		p.logger.Error("Chain get common ancestor nil! Height:%d,Hash:%s", commonAncestor.Height, commonAncestor.Hash.String())
		p.finishCurrentSync(true)
		return
	}
	go p.syncBlock(from, *commonAncestorBlock)
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

		//todo 创始块组签名没写
		if bh.Height > 0 {
			signVerifyResult, _ := consensusHelper.VerifyBlockHeader(bh)
			if !signVerifyResult {
				return false
			}
		}
	}
	return true
}

func (p *syncProcessor) syncBlock(id string, commonAncestor types.Block) {
	p.lock.Lock("syncBlock")
	if p.blockFork == nil {
		p.blockFork = newBlockChainFork(commonAncestor)
	}
	p.lock.Unlock("syncBlock")

	syncHeight := commonAncestor.Header.Height + 1
	p.logger.Debugf("Sync block to:%s,reqHeight:%d", id, syncHeight)
	req := blockSyncReq{Height: syncHeight}
	req.SignInfo = common.NewSignData(p.privateKey, p.id, &req)

	body, e := marshalBlockSyncReq(req)
	if e != nil {
		p.logger.Errorf("marshal block req error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.ReqBlockMsg, Body: body}
	go network.GetNetInstance().SendToStranger(common.FromHex(id), message)
	p.reqTimer.Reset(syncReqTimeout)
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

	blockList := p.blockChain.getSyncedBlock(reqHeight)
	isLastBlock := false
	for i := 0; i <= len(blockList)-1; i++ {
		block := blockList[i]
		if i == len(blockList)-1 && localHeight <= block.Header.Height {
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
	}

	if len(blockList) == 0 {
		syncHandleLogger.Debugf("Synced block len 0!")
	} else {
		syncHandleLogger.Debugf("Send block %d-%d to %s,last:%v", blockList[0].Header.Height, blockList[len(blockList)-1].Header.Height, req.SignInfo.Id, isLastBlock)
	}
}

func (p *syncProcessor) blockResponseMsgHandler(msg notify.Message) {
	p.syncedBlockCh <- msg
}

func (p *syncProcessor) processSyncedBlock(msg notify.Message) {
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
	if from != p.candidateInfo.Id {
		p.logger.Debugf("[BlockResponseMessage]Unexpected candidate! Expect from:%s, actual:%s,!", p.candidateInfo.Id, from)
		return
	}
	block := blockResponse.Block
	p.logger.Debugf("Rcv synced block.Hash:%s,%d-%d.Pre:%s", block.Header.Hash.String(), block.Header.Height, block.Header.TotalQN, block.Header.PreHash.String())
	p.reqTimer.Reset(syncReqTimeout)

	if p.blockFork == nil || !p.blockFork.enableRcvBlock {
		return
	}
	p.blockFork.pending.Enqueue(block)
	p.blockFork.rcvLastBlock = blockResponse.IsLastBlock
	if p.blockFork.rcvLastBlock || p.blockFork.pending.Capacity() >= syncedBlockCount {
		p.reqTimer.Stop()
		p.blockFork.enableRcvBlock = false
		p.tryAcceptBlock()
	}
}

func (p *syncProcessor) tryAcceptBlock() {
	p.logger.Debugf("Try accept block")
	var err error
	var block *types.Block
	for !p.blockFork.pending.Empty() {
		b := p.blockFork.pending.Head().(*types.Block)
		err = p.blockFork.acceptBlock(b)
		if err != nil {
			p.blockFork.logger.Debugf("Accept block failed!%s,%d-%d", b.Header.Hash.String(), block.Header.Height, block.Header.TotalQN)
			break
		}
		block = p.blockFork.pending.Pop().(*types.Block)
		p.blockFork.waitingGroup = false
	}

	mergeResult := p.tryMergeFork()
	p.logger.Debugf("Try merge fork result:%v", mergeResult)
	if mergeResult {
		p.finishCurrentSync(true)
		return
	}

	if err == verifyGroupNotOnChainErr {
		p.logger.Debugf("block fork waiting group..")
		p.blockFork.waitingGroup = true
		go p.triggerSync()
		return
	}
	if err == verifyBlockErr {
		p.finishCurrentSync(false)
		return
	}
	if p.blockFork.rcvLastBlock {
		p.finishCurrentSync(true)
		return
	}
	if block != nil {
		p.blockFork.enableRcvBlock = true
		go p.syncBlock(p.candidateInfo.Id, *block)
	}
}
