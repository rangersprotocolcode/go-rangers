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
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/middleware"
	"com.tuntun.rangers/node/src/middleware/log"
	"com.tuntun.rangers/node/src/middleware/notify"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/network"
	"com.tuntun.rangers/node/src/utility"
	"time"
)

const (
	blockSyncInterval          = 3 * time.Second
	broadcastBlockInfoInterval = 3 * time.Second
	syncCandidatePoolSize      = 3
	syncReqTimeout             = 1 * time.Second

	blockForkDBPrefix            = "blockFork"
	blockCommonAncestorHeightKey = "blockCommonAncestor"
	latestBlockHeightKey         = "latestBlock"
	blockChainPieceLength        = 9
	syncedBlockCount             = 16

	groupForkDBPrefix            = "groupFork"
	groupCommonAncestorHeightKey = "groupCommonAncestorGroup"
	latestGroupHeightKey         = "latestGroup"
	syncedGroupCount             = 16
)

type CandidateInfo struct {
	Id          string
	Height      uint64
	TotalQn     uint64
	GroupHeight uint64
}

var SyncProcessor *syncProcessor

type syncProcessor struct {
	privateKey common.PrivateKey
	id         string

	candidateInfo CandidateInfo
	candidatePool map[string]chainInfo

	canSync bool
	syncing bool

	syncTimer      *time.Timer
	blockReqTimer  *time.Timer
	groupReqTimer  *time.Timer
	broadcastTimer *time.Timer

	blockFork *blockChainFork
	groupFork *groupChainFork

	blockChain *blockChain
	groupChain *groupChain

	lock   middleware.Loglock
	logger log.Logger
}

func InitSyncProcessor(privateKey common.PrivateKey, id string) {
	SyncProcessor = &syncProcessor{privateKey: privateKey, id: id, syncing: false, candidatePool: make(map[string]chainInfo)}

	SyncProcessor.broadcastTimer = time.NewTimer(broadcastBlockInfoInterval)
	SyncProcessor.blockReqTimer = time.NewTimer(syncReqTimeout)
	SyncProcessor.groupReqTimer = time.NewTimer(syncReqTimeout)
	SyncProcessor.syncTimer = time.NewTimer(blockSyncInterval)

	SyncProcessor.blockChain = blockChainImpl
	SyncProcessor.groupChain = groupChainImpl

	SyncProcessor.lock = middleware.NewLoglock("sync")
	SyncProcessor.logger = syncLogger

	notify.BUS.Subscribe(notify.TopBlockInfo, SyncProcessor.chainInfoNotifyHandler)
	notify.BUS.Subscribe(notify.BlockChainPieceReq, SyncProcessor.blockChainPieceReqHandler)
	notify.BUS.Subscribe(notify.BlockChainPiece, SyncProcessor.blockChainPieceHandler)
	notify.BUS.Subscribe(notify.BlockReq, SyncProcessor.syncBlockReqHandler)
	notify.BUS.Subscribe(notify.BlockResponse, SyncProcessor.blockResponseMsgHandler)
	notify.BUS.Subscribe(notify.GroupReq, SyncProcessor.syncGroupReqHandler)
	notify.BUS.Subscribe(notify.GroupResponse, SyncProcessor.groupResponseMsgHandler)
	go SyncProcessor.loop()
}

func (p *syncProcessor) GetCandidateInfo() CandidateInfo {
	p.lock.RLock("GetCandidateInfo")
	defer p.lock.RUnlock("GetCandidateInfo")
	return p.candidateInfo
}

func StartSync() {
	if SyncProcessor != nil {
		SyncProcessor.canSync = true
	}
}

func (p *syncProcessor) loop() {
	for {
		select {
		case <-p.broadcastTimer.C:
			go p.broadcastChainInfo(p.blockChain.TopBlock())
		case <-p.syncTimer.C:
			go p.trySync()
		case <-p.blockReqTimer.C:
			candidateId := p.GetCandidateInfo().Id
			p.logger.Debugf("Sync to %s time out!", candidateId)
			p.finishCurrentSyncWithLock(false)
		case <-p.groupReqTimer.C:
			candidateId := p.GetCandidateInfo().Id
			p.logger.Debugf("Sync to %s time out!", candidateId)
			p.finishCurrentSyncWithLock(false)
		}
	}
}

func (p *syncProcessor) broadcastChainInfo(bh *types.BlockHeader) {
	if bh.Height == 0 {
		return
	}
	topBlockInfo := chainInfo{TopBlockHash: bh.Hash, TotalQn: bh.TotalQN, TopBlockHeight: bh.Height, PreHash: bh.PreHash, TopGroupHeight: p.groupChain.height()}
	topBlockInfo.SignInfo = common.NewSignData(p.privateKey, p.id, &topBlockInfo)

	body, e := marshalChainInfo(topBlockInfo)
	if e != nil {
		p.logger.Errorf("marshal chain info error:%s", e.Error())
		return
	}
	p.logger.Tracef("Send local total qn %d to neighbor!", topBlockInfo.TotalQn)
	message := network.Message{Code: network.TopBlockInfoMsg, Body: body}
	network.GetNetInstance().Broadcast(message)
	p.broadcastTimer.Reset(broadcastBlockInfoInterval)
}

// rcv chain info from neighborhood
func (p *syncProcessor) chainInfoNotifyHandler(msg notify.Message) {
	bnm, ok := msg.GetData().(*notify.ChainInfoMessage)
	if !ok {
		syncHandleLogger.Errorf("ChainInfoMessage assert not ok!")
		return
	}
	chainInfo, e := unMarshalChainInfo(bnm.ChainInfo)
	if e != nil {
		syncHandleLogger.Errorf("Discard message! ChainInfoMessage unmarshal error:%s", e.Error())
		return
	}
	err := chainInfo.SignInfo.ValidateSign(chainInfo)
	if err != nil {
		syncHandleLogger.Errorf("Sign verify error! ChainInfoMessage:%s", e.Error())
		return
	}
	syncHandleLogger.Tracef("Rcv chain info! Height:%d,qn:%d,group height:%d,source:%s", chainInfo.TopBlockHeight, chainInfo.TotalQn, chainInfo.TopGroupHeight, chainInfo.SignInfo.Id)
	source := chainInfo.SignInfo.Id
	if PeerManager.isEvil(source) {
		syncHandleLogger.Debugf("[chainInfoNotifyHandler]%s is marked evil.Drop!", source)
		return
	}

	if p.isUseful(*chainInfo) {
		p.addCandidate(source, *chainInfo)
	}
}

func (p *syncProcessor) addCandidate(id string, chainInfo chainInfo) {
	p.lock.Lock("addCandidatePool")
	defer p.lock.Unlock("addCandidatePool")

	if len(p.candidatePool) < syncCandidatePoolSize {
		p.candidatePool[id] = chainInfo
		return
	}
	totalQnMinId := ""
	var minTotalQn uint64 = utility.MaxUint64
	for id, tbi := range p.candidatePool {
		if tbi.TotalQn <= minTotalQn {
			totalQnMinId = id
			minTotalQn = tbi.TotalQn
		}
	}
	if chainInfo.TotalQn >= minTotalQn {
		delete(p.candidatePool, totalQnMinId)
		p.candidatePool[id] = chainInfo
		go p.trySync()
	}
}

func (p *syncProcessor) trySync() {
	if !p.canSync {
		return
	}
	if p.syncing {
		candidateId := p.GetCandidateInfo().Id
		p.logger.Debugf("Syncing to %s,do not sync!", candidateId)
		return
	}
	p.lock.Lock("trySync")
	defer p.lock.Unlock("trySync")
	if p.syncing {
		p.logger.Debugf("Syncing to %s,do not sync!", p.candidateInfo.Id)
		return
	}
	p.logger.Debugf("Try sync!")
	p.syncTimer.Reset(blockSyncInterval)

	topBlock := blockChainImpl.TopBlock()
	localTotalQN, localBlockHeight := topBlock.TotalQN, topBlock.Height
	localGroupHeight := p.groupChain.height()
	p.logger.Tracef("Local totalQn:%d,height:%d,topHash:%s,groupHeight:%d", localTotalQN, localBlockHeight, topBlock.Hash.String(), localGroupHeight)
	candidateInfo := p.chooseSyncCandidate()
	if candidateInfo.Id == "" || candidateInfo.TotalQn < localTotalQN {
		p.logger.Debugf("No valid candidate for sync!")
		return
	}

	p.syncing = true
	p.candidateInfo = candidateInfo
	p.logger.Debugf("Begin sync!")
	p.logger.Debugf("Candidate info:%s,%d-%d,%d", candidateInfo.Id, candidateInfo.Height, candidateInfo.TotalQn, candidateInfo.GroupHeight)
	p.logger.Debugf("Local info:%d-%d,%d", localBlockHeight, localTotalQN, localGroupHeight)
	go p.requestBlockChainPiece(candidateInfo.Id, localBlockHeight)
}

func (p *syncProcessor) isUseful(candidateInfo chainInfo) bool {
	topBlock := blockChainImpl.TopBlock()
	localTotalQn, localTopHash := topBlock.TotalQN, topBlock.Hash
	localGroupHeight := p.groupChain.height()
	if candidateInfo.TotalQn > localTotalQn {
		return true
	}
	if localTotalQn == candidateInfo.TotalQn && localTopHash != candidateInfo.TopBlockHash {
		return true
	}
	if localTotalQn == candidateInfo.TotalQn && localTopHash == candidateInfo.TopBlockHash && localGroupHeight < candidateInfo.TopGroupHeight {
		return true
	}
	return false
}

func (p *syncProcessor) chooseSyncCandidate() CandidateInfo {
	evilCandidates := make([]string, 0, syncCandidatePoolSize)
	for id, _ := range p.candidatePool {
		if PeerManager.isEvil(id) {
			evilCandidates = append(evilCandidates, id)
		}
	}
	if len(evilCandidates) != 0 {
		for _, id := range evilCandidates {
			delete(p.candidatePool, id)
		}
	}

	candidateInfo := CandidateInfo{}
	for id, chainInfo := range p.candidatePool {
		if p.isUseful(chainInfo) {
			candidateInfo.Id = id
			candidateInfo.TotalQn = chainInfo.TotalQn
			candidateInfo.Height = chainInfo.TopBlockHeight
			candidateInfo.GroupHeight = chainInfo.TopGroupHeight
			break
		}
	}
	return candidateInfo
}

func (p *syncProcessor) readyOnFork() bool {
	if p.blockFork == nil || p.groupFork == nil {
		return false
	}
	blockReady := p.blockFork.rcvLastBlock || p.blockFork.pending.Size() >= syncedBlockCount
	groupReady := p.groupFork.rcvLastGroup || p.groupFork.pending.Size() >= syncedGroupCount
	return blockReady && groupReady
}

func (p *syncProcessor) triggerOnFork() {
	p.lock.Lock("triggerBlockOnFork")
	defer p.lock.Unlock("triggerBlockOnFork")

	if p.blockFork == nil || p.groupFork == nil {
		return
	}
	var pausedBlock *types.Block
	p.logger.Debugf("Trigger on fork....Block fork head %d,Group fork head %d", p.blockFork.header, p.groupFork.header)
	for {
		blockForkErr, currentBlock := p.blockFork.triggerOnFork(p.groupFork)
		groupForkErr, currentGroup := p.groupFork.triggerOnFork(p.blockFork)
		if p.tryTriggerOnChain() {
			return
		}
		if blockForkErr == nil && !p.blockFork.rcvLastBlock {
			go p.syncBlock(p.candidateInfo.Id, *p.blockFork.latestBlock)
			return
		}
		if groupForkErr == nil && !p.groupFork.rcvLastGroup {
			go p.syncGroup(p.candidateInfo.Id, p.groupFork.latestGroup)
			return
		}
		if blockForkErr != nil && blockForkErr != verifyGroupNotOnChainErr && blockForkErr != common.ErrSelectGroupInequal {
			p.finishCurrentSync(false)
			return
		}
		if groupForkErr != nil && groupForkErr != common.ErrCreateBlockNil {
			p.finishCurrentSync(false)
			return
		}

		if pausedBlock == currentBlock {
			p.logger.Warnf("sync deadlock! rcv last block:%v,rcv last group:%v", p.blockFork.rcvLastBlock, p.groupFork.rcvLastGroup)
			if currentBlock != nil {
				p.logger.Debugf("paused block %s,%d-%d,verify group:%s", currentBlock.Header.Hash.Str(), currentBlock.Header.Height, currentBlock.Header.TotalQN, common.ToHex(currentBlock.Header.GroupId))
			}
			if currentGroup != nil {
				p.logger.Debugf("paused group %s,%d,create block:%s", common.ToHex(currentGroup.Id), currentGroup.GroupHeight, common.ToHex(currentGroup.Header.CreateBlockHash))
			}
			p.finishCurrentSync(false)
			return
		}
		pausedBlock = currentBlock
	}
}

func (p *syncProcessor) tryTriggerOnChain() (canOnChain bool) {
	if p.blockFork.latestBlock.TotalQN >= p.blockChain.latestBlock.TotalQN || p.groupFork.latestGroup.GroupHeight > p.groupChain.height() {
		var pausedBlockHeight, pausedGroupHeight uint64
		for {
			addBlockResult := p.blockFork.triggerOnChain(p.blockChain)
			addGroupResult := p.groupFork.triggerOnChain(p.groupChain)
			if addBlockResult && addGroupResult {
				p.logger.Debugf("Trigger on chain success.")
				p.finishCurrentSync(true)
				return true
			}

			if pausedBlockHeight == p.blockFork.current && pausedGroupHeight == p.groupFork.current {
				p.logger.Debugf("Trigger on chain failed.")
				p.finishCurrentSync(false)
				return true
			}
			pausedBlockHeight = p.blockFork.current
			pausedGroupHeight = p.groupFork.current
		}
	}
	p.logger.Debugf("Try trigger on chain failed.Local:%d,%d,fork:%d,%d", p.blockChain.latestBlock.TotalQN, p.groupChain.height(), p.blockFork.latestBlock.TotalQN, p.groupFork.latestGroup.GroupHeight)
	return false
}

func (p *syncProcessor) finishCurrentSyncWithLock(syncResult bool) {
	p.lock.Lock("finish sync")
	defer p.lock.Unlock("finish sync")
	p.finishCurrentSync(syncResult)
}

func (p *syncProcessor) finishCurrentSync(syncResult bool) {
	if !syncResult {
		PeerManager.markEvil(p.candidateInfo.Id)
	}

	if p.blockFork != nil {
		p.blockFork.destroy()
		p.blockFork = nil
	}
	if p.groupFork != nil {
		p.groupFork.destroy()
		p.groupFork = nil
	}
	p.blockReqTimer.Stop()
	p.groupReqTimer.Stop()
	p.candidateInfo = CandidateInfo{}
	p.syncing = false
	p.logger.Debugf("finish current sync:%v", syncResult)
}
