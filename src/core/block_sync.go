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
	"time"

	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/notify"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/network"
)

const (
	blockSyncInterval          = 3 * time.Second
	broadcastBlockInfoInterval = 3 * time.Second
	blockSyncCandidatePoolSize = 3
	blockSyncReqTimeout        = 1 * time.Second
)

type BlockSyncInfo struct {
	LocalHeight  uint64
	LocalTotalQn uint64
	Candidate    *CandidateInfo
}

type CandidateInfo struct {
	Id      string
	Height  uint64
	TotalQn uint64
}

var BlockSyncer *blockSyncer

type blockSyncer struct {
	privateKey common.PrivateKey
	id         string

	candidateInfo CandidateInfo
	candidatePool map[string]TopBlockInfo

	syncing bool
	fork    *fork

	syncTimer      *time.Timer
	reqTimer       *time.Timer
	broadcastTimer *time.Timer

	chain  *blockChain
	Lock   middleware.Loglock
	logger log.Logger
}

func InitBlockSyncer(privateKey common.PrivateKey, id string) {
	BlockSyncer = &blockSyncer{privateKey: privateKey, id: id, syncing: false, candidatePool: make(map[string]TopBlockInfo), chain: blockChainImpl, Lock: middleware.NewLoglock("")}
	BlockSyncer.logger = blockSyncLogger
	BlockSyncer.broadcastTimer = time.NewTimer(broadcastBlockInfoInterval)
	BlockSyncer.reqTimer = time.NewTimer(blockSyncReqTimeout)
	BlockSyncer.syncTimer = time.NewTimer(blockSyncInterval)

	notify.BUS.Subscribe(notify.BlockInfoNotify, BlockSyncer.topBlockInfoNotifyHandler)
	notify.BUS.Subscribe(notify.ChainPieceInfoReq, BlockSyncer.chainPieceReqHandler)
	notify.BUS.Subscribe(notify.ChainPieceInfo, BlockSyncer.chainPieceHandler)
	notify.BUS.Subscribe(notify.BlockReq, BlockSyncer.syncBlockReqHandler)
	notify.BUS.Subscribe(notify.BlockResponse, BlockSyncer.blockResponseMsgHandler)
	go BlockSyncer.loop()
}

func (bs *blockSyncer) loop() {
	for {
		select {
		case <-bs.broadcastTimer.C:
			go bs.broadcastTopBlockInfo(bs.chain.TopBlock())
		case <-bs.syncTimer.C:
			go bs.trySync()
		case <-bs.reqTimer.C:
			bs.logger.Debugf("Block sync to %s time out!", bs.candidateInfo.Id)
			PeerManager.markEvil(bs.candidateInfo.Id)
			bs.finishCurrentSync()
		}
	}
}

func (bs *blockSyncer) GetSyncInfo() BlockSyncInfo {
	localTopHeader := bs.chain.latestBlock
	return BlockSyncInfo{LocalHeight: localTopHeader.Height, LocalTotalQn: localTopHeader.TotalQN, Candidate: &bs.candidateInfo}
}

func (bs *blockSyncer) finishCurrentSync() {
	bs.Lock.Lock("finish current sync")
	bs.logger.Debugf("finish current sync!")
	bs.reqTimer.Stop()
	bs.candidateInfo = CandidateInfo{}
	bs.syncing = false
	destoryFork(bs.fork)
	bs.Lock.Unlock("finish current")
}

//----------------------------------------rcv block info from neighborhood and choose candidate------------------------------------

//broadcast local top block info to neighborhood
func (bs *blockSyncer) broadcastTopBlockInfo(bh *types.BlockHeader) {
	if bh.Height == 0 {
		return
	}
	topBlockInfo := TopBlockInfo{Hash: bh.Hash, TotalQn: bh.TotalQN, Height: bh.Height, PreHash: bh.PreHash}
	topBlockInfo.SignInfo = common.NewSignData(bs.privateKey, bs.id, &topBlockInfo)

	body, e := marshalTopBlockInfo(topBlockInfo)
	if e != nil {
		bs.logger.Errorf("marshal top block info error:%s", e.Error())
		return
	}
	bs.logger.Debugf("Send local total qn %d to neighbor!", topBlockInfo.TotalQn)
	message := network.Message{Code: network.BlockInfoNotifyMsg, Body: body}
	network.GetNetInstance().Broadcast(message)

	bs.Lock.Lock("broadcastTopBlockInfo")
	bs.broadcastTimer.Reset(broadcastBlockInfoInterval)
	bs.Lock.Unlock("broadcastTopBlockInfo")
}

//rcv block info from neighborhood
func (bs *blockSyncer) topBlockInfoNotifyHandler(msg notify.Message) {
	bnm, ok := msg.GetData().(*notify.BlockInfoNotifyMessage)
	if !ok {
		bs.logger.Errorf("BlockInfoNotifyMessage GetData assert not ok!")
		return
	}
	blockInfo, e := unMarshalTopBlockInfo(bnm.BlockInfo)
	if e != nil {
		bs.logger.Errorf("Discard BlockInfoNotifyMessage because of unmarshal error:%s", e.Error())
		return
	}

	err := blockInfo.SignInfo.ValidateSign(blockInfo)
	if err != nil {
		bs.logger.Errorf("BlockInfoNotifyMessage sign validate error:%s", e.Error())
		return
	}
	bs.logger.Debugf("Rcv top block info! Height:%d,qn:%d,source:%s", blockInfo.Height, blockInfo.TotalQn, blockInfo.SignInfo.Id)
	topBlock := blockChainImpl.TopBlock()
	localTotalQn, localTopHash := topBlock.TotalQN, topBlock.Hash
	if blockInfo.TotalQn < localTotalQn || (localTotalQn == blockInfo.TotalQn && localTopHash == blockInfo.Hash) {
		return
	}

	source := blockInfo.SignInfo.Id
	if PeerManager.isEvil(source) {
		bs.logger.Debugf("Top block info notify id:%s is marked evil.Drop it!", source)
		return
	}
	bs.addCandidate(source, *blockInfo)
}

func (bs *blockSyncer) addCandidate(id string, topBlockInfo TopBlockInfo) {
	bs.Lock.Lock("addCandidatePool")
	defer bs.Lock.Unlock("addCandidatePool")

	if len(bs.candidatePool) < blockSyncCandidatePoolSize {
		bs.candidatePool[id] = topBlockInfo
		return
	}
	totalQnMinId := ""
	var minTotalQn uint64 = common.MaxUint64
	for id, tbi := range bs.candidatePool {
		if tbi.TotalQn <= minTotalQn {
			totalQnMinId = id
			minTotalQn = tbi.TotalQn
		}
	}
	if topBlockInfo.TotalQn > minTotalQn {
		delete(bs.candidatePool, totalQnMinId)
		bs.candidatePool[id] = topBlockInfo
		go bs.trySync()
	}
}

func (bs *blockSyncer) trySync() {
	if bs.syncing {
		bs.logger.Debugf("Syncing to %s,do not sync anymore!", bs.candidateInfo.Id)
		return
	}
	bs.Lock.Lock("trySync")
	defer bs.Lock.Unlock("trySync")
	bs.logger.Debugf("Try sync!")
	bs.syncTimer.Reset(blockSyncInterval)

	topBlock := blockChainImpl.TopBlock()
	localTotalQN, localHeight := topBlock.TotalQN, topBlock.Height
	//bs.logger.Debugf("Local totalQn:%d,height:%d,topHash:%s", localTotalQN, localHeight, localTopHash.String())
	candidateInfo := bs.chooseSyncCandidate()
	if candidateInfo.Id == "" || candidateInfo.TotalQn <= localTotalQN {
		bs.logger.Debugf("There is no valid candidate for sync!")
		return
	}
	bs.logger.Debugf("Sync from %s!Req height:%d", candidateInfo.Id, localHeight)
	bs.syncing = true
	bs.candidateInfo = candidateInfo

	go bs.requestChainPiece(candidateInfo.Id, localHeight)
}

func (bs *blockSyncer) chooseSyncCandidate() CandidateInfo {
	evilCandidates := make([]string, 0, blockSyncCandidatePoolSize)
	for id, _ := range bs.candidatePool {
		if PeerManager.isEvil(id) {
			evilCandidates = append(evilCandidates, id)
		}
	}
	if len(evilCandidates) != 0 {
		for _, id := range evilCandidates {
			delete(bs.candidatePool, id)
		}
	}

	candidateInfo := CandidateInfo{}
	for id, topBlockInfo := range bs.candidatePool {
		if topBlockInfo.TotalQn > candidateInfo.TotalQn {
			candidateInfo.Id = id
			candidateInfo.TotalQn = topBlockInfo.TotalQn
			candidateInfo.Height = topBlockInfo.Height
		}
	}
	return candidateInfo
}

//--------------------------------------request block headers for common ancestor--------------------------------------------------------------------------------
//request block header list for finding common ancestor
func (bs *blockSyncer) requestChainPiece(targetNode string, localHeight uint64) {

	req := ChainPieceReq{Height: localHeight}
	req.SignInfo = common.NewSignData(bs.privateKey, bs.id, &req)

	body, e := marshalChainPieceReq(req)
	if e != nil {
		bs.logger.Errorf("marshal chain piece req error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.ChainPieceInfoReq, Body: body}
	network.GetNetInstance().SendToStranger(common.FromHex(targetNode), message)
	bs.reqTimer.Reset(blockSyncReqTimeout)
}

func (bs *blockSyncer) chainPieceReqHandler(msg notify.Message) {
	chainPieceReqMessage, ok := msg.GetData().(*notify.ChainPieceInfoReqMessage)
	if !ok {
		return
	}
	chainPieceReq, e := unMarshalChainPieceReq(chainPieceReqMessage.ChainPieceReq)
	if e != nil {
		bs.logger.Errorf("unmarshal chain piece req error:%s", e.Error())
		return
	}

	err := chainPieceReq.SignInfo.ValidateSign(chainPieceReq)
	if err != nil {
		bs.logger.Errorf("ChainPieceInfoReqMessage sign validate error:%s", e.Error())
		return
	}

	from := chainPieceReq.SignInfo.Id
	bs.logger.Debugf("Rcv chain piece info req from:%s,req height:%d", from, chainPieceReq.Height)
	chainPiece := bs.chain.getChainPiece(chainPieceReq.Height)

	chainPieceMsg := chainPieceInfo{ChainPiece: chainPiece, TopHeader: blockChainImpl.TopBlock()}
	chainPieceMsg.SignInfo = common.NewSignData(bs.privateKey, bs.id, &chainPieceMsg)
	bs.logger.Debugf("Send chain piece %d-%d to:%s", chainPiece[0].Height, chainPiece[len(chainPiece)-1].Height, from)
	body, e := marshalChainPieceInfo(chainPieceMsg)
	if e != nil {
		bs.logger.Errorf("Marshal chain piece info error:%s!", e.Error())
		return
	}
	message := network.Message{Code: network.ChainPieceInfo, Body: body}
	network.GetNetInstance().SendToStranger(common.FromHex(from), message)
}

func (bs *blockSyncer) chainPieceHandler(msg notify.Message) {
	chainPieceInfoMessage, ok := msg.GetData().(*notify.ChainPieceInfoMessage)
	if !ok {
		return
	}
	chainPieceInfo, err := unMarshalChainPieceInfo(chainPieceInfoMessage.ChainPieceInfoByte)
	if err != nil {
		bs.logger.Errorf("Unmarshal chain piece info error:%s", err.Error())
		bs.finishCurrentSync()
		return
	}

	err = chainPieceInfo.SignInfo.ValidateSign(chainPieceInfo)
	if err != nil {
		bs.logger.Errorf("ChainPieceInfoMessage sign validate error:%s", err.Error())
		bs.finishCurrentSync()
		return
	}

	from := chainPieceInfo.SignInfo.Id
	if from != bs.candidateInfo.Id {
		bs.logger.Debugf("Unexpected chain piece info from %s, expect from %s!", from, bs.candidateInfo.Id)
		PeerManager.markEvil(from)
		bs.finishCurrentSync()
		return
	}
	bs.reqTimer.Stop()

	chainPiece := chainPieceInfo.ChainPiece
	if !verifyChainPieceInfo(chainPiece, chainPieceInfo.TopHeader) {
		bs.logger.Debugf("Bad chain piece info from %s", from)
		PeerManager.markEvil(from)
		bs.finishCurrentSync()
		return
	}

	localTopHeader := bs.chain.TopBlock()
	if chainPieceInfo.TopHeader.TotalQN < localTopHeader.TotalQN {
		bs.finishCurrentSync()
		return
	}

	//piece 正序 index大  height高
	var commonAncestor *types.BlockHeader
	for i := 0; i < len(chainPiece); i++ {
		height := chainPiece[i].Height
		if bs.chain.GetBlockHash(height) != chainPiece[i].Hash {
			break
		}
		commonAncestor = chainPiece[i]
	}

	if commonAncestor == nil {
		if chainPiece[0].Height == 0 {
			bs.logger.Error("Genesis block is different.Can not sync!")
			bs.finishCurrentSync()
			return
		}
		bs.requestChainPiece(from, chainPiece[0].Height)
		return
	}

	bs.logger.Debugf("Common ancestor height:%d", commonAncestor.Height)
	//if commonAncestor == chainPiece[len(chainPiece)-1] {
	//	bs.finishCurrentSync()
	//	return
	//}

	commonAncestorBlock := bs.chain.queryBlockByHash(commonAncestor.Hash)
	if commonAncestorBlock == nil {
		bs.logger.Error("Chain get common ancestor nil! Height:%d,Hash:%s", commonAncestor.Height, commonAncestor.Hash.String())
		bs.finishCurrentSync()
		return
	}
	bs.syncBlock(from, *commonAncestorBlock)
}

func verifyChainPieceInfo(chainPiece []*types.BlockHeader, topHeader *types.BlockHeader) bool {
	if len(chainPiece) == 0 || topHeader == nil {
		return false
	}

	//can not verify top header group sign
	for i := 0; i < len(chainPiece)-1; i++ {
		bh := chainPiece[i]
		if bh == nil {
			return false
		}

		if i > 0 && bh.PreHash != chainPiece[i-1].Hash {
			return false
		}

		signVerifyResult, _ := consensusHelper.VerifyBlockHeader(bh)
		if !signVerifyResult {
			return false
		}
	}
	return true
}

//--------------------------------------------------sync block--------------------------------------------------------------------
func (bs *blockSyncer) syncBlock(id string, commonAncestor types.Block) {
	bs.fork = newFork(commonAncestor, id, bs.logger)
	syncHeight := commonAncestor.Header.Height + 1
	bs.logger.Debugf("Sync block to:%s,reqHeight:%d", id, syncHeight)
	req := BlockSyncReq{Height: syncHeight}
	req.SignInfo = common.NewSignData(bs.privateKey, bs.id, &req)

	body, e := marshalBlockSyncReq(req)
	if e != nil {
		bs.logger.Errorf("marshal block req error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.ReqBlock, Body: body}
	go network.GetNetInstance().SendToStranger(common.FromHex(id), message)
	bs.reqTimer.Reset(blockSyncReqTimeout)
}

func (bs *blockSyncer) syncBlockReqHandler(msg notify.Message) {
	bs.logger.Debugf("Rcv BlockReqMessage")
	m, ok := msg.(*notify.BlockReqMessage)
	if !ok {
		bs.logger.Debugf("BlockReqMessage:Message assert not ok!")
		return
	}

	bs.logger.Debugf("Rcv BlockReqMessage.%d", m.Peer)
	req, e := unMarshalBlockSyncReq(m.ReqInfoByte)
	if e != nil {
		bs.logger.Debugf("Discard block req msg because unMarshalBlockSyncReq error:%d", e.Error())
		return
	}

	err := req.SignInfo.ValidateSign(req)
	if err != nil {
		bs.logger.Errorf("syncBlockReqHandler sign validate error:%s", e.Error())
		return
	}

	reqHeight := req.Height
	localHeight := blockChainImpl.Height()
	bs.logger.Debugf("Rcv block request:reqHeight:%d,localHeight:%d", reqHeight, localHeight)

	blockList := bs.chain.getSyncedBlock(reqHeight)
	for i := 0; i <= len(blockList)-1; i++ {
		block := blockList[i]
		isLastBlock := false
		if i == len(blockList)-1 {
			isLastBlock = true
		}
		response := BlockMsgResponse{Block: block, IsLastBlock: isLastBlock}
		response.SignInfo = common.NewSignData(bs.privateKey, bs.id, &response)
		body, e := marshalBlockMsgResponse(response)
		if e != nil {
			bs.logger.Errorf("Marshal block msg response error:%s", e.Error())
			return
		}
		message := network.Message{Code: network.BlockResponseMsg, Body: body}
		network.GetNetInstance().Send(m.Peer, message)
		bs.logger.Debugf("Send %d to %s,is last:%v", response.Block.Header.Height, m.Peer, response.IsLastBlock)
	}
}

func (bs *blockSyncer) blockResponseMsgHandler(msg notify.Message) {
	m, ok := msg.(*notify.BlockResponseMessage)
	if !ok {
		return
	}

	blockResponse, e := bs.unMarshalBlockMsgResponse(m.BlockResponseByte)
	if e != nil {
		bs.logger.Debugf("Discard block response msg because unMarshalBlockMsgResponse error:%d", e.Error())
		return
	}

	err := blockResponse.SignInfo.ValidateSign(blockResponse)
	if err != nil {
		bs.logger.Errorf("BlockResponseMessage sign validate error:%s", e.Error())
		return
	}
	from := blockResponse.SignInfo.Id
	bs.logger.Debugf("blockResponseMsgHandler rcv from %s!", from)
	if from != bs.candidateInfo.Id {
		bs.logger.Debugf("Unexpected block response from %s, expect from %s!", from, bs.candidateInfo.Id)
		return
	}
	bs.reqTimer.Reset(blockSyncReqTimeout)

	block := blockResponse.Block
	isLastBlock := blockResponse.IsLastBlock
	if !bs.fork.acceptBlock(*block, from) {
		bs.logger.Debugf("Accept block failed!%s,%d-%d", block.Header.Hash.String(), block.Header.Height, block.Header.TotalQN)
		mergeResult := bs.chain.tryMergeFork(bs.fork)
		bs.logger.Debugf("Try merge fork result:%v", mergeResult)
		if !mergeResult {
			PeerManager.markEvil(from)
		}
		bs.finishCurrentSync()
		return
	}

	if isLastBlock {
		mergeResult := bs.chain.tryMergeFork(bs.fork)
		bs.logger.Debugf("Try merge fork result:%v", mergeResult)
		bs.finishCurrentSync()
	}
}

func (bs *blockSyncer) candidatePoolDump() {
	bs.logger.Debugf("Candidate Pool Dump:")
	for id, topBlockInfo := range bs.candidatePool {
		bs.logger.Debugf("Candidate id:%s,totalQn:%d,height:%d,topHash:%s", id, topBlockInfo.TotalQn, topBlockInfo.Height, topBlockInfo.Hash.String())
	}
}
