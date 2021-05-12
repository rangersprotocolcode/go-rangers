package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/notify"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/network"
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
	latestGroupHeightKey         = "lastestGroup"
	groupChainPieceLength        = 9
	syncedGroupCount             = 16
)

type CandidateInfo struct {
	Id      string
	Height  uint64
	TotalQn uint64
}

var SyncProcessor *syncProcessor

type syncProcessor struct {
	privateKey common.PrivateKey
	id         string

	candidateInfo CandidateInfo
	candidatePool map[string]topBlockInfo

	syncing bool

	syncTimer      *time.Timer
	reqTimer       *time.Timer
	broadcastTimer *time.Timer

	blockFork *blockChainFork
	groupFork *groupChainFork

	blockChain *blockChain
	groupChain *groupChain

	lock   middleware.Loglock
	logger log.Logger
}

func InitSyncerProcessor(privateKey common.PrivateKey, id string) {
	SyncProcessor = &syncProcessor{privateKey: privateKey, id: id, syncing: false, candidatePool: make(map[string]topBlockInfo)}

	SyncProcessor.broadcastTimer = time.NewTimer(broadcastBlockInfoInterval)
	SyncProcessor.reqTimer = time.NewTimer(syncReqTimeout)
	SyncProcessor.syncTimer = time.NewTimer(blockSyncInterval)

	SyncProcessor.blockChain = blockChainImpl
	SyncProcessor.groupChain = groupChainImpl

	SyncProcessor.lock = middleware.NewLoglock("sync")
	SyncProcessor.logger = syncLogger

	notify.BUS.Subscribe(notify.TopBlockInfo, SyncProcessor.topBlockInfoNotifyHandler)
	notify.BUS.Subscribe(notify.BlockChainPieceReq, SyncProcessor.blockChainPieceReqHandler)
	notify.BUS.Subscribe(notify.BlockChainPiece, SyncProcessor.blockChainPieceHandler)
	notify.BUS.Subscribe(notify.BlockReq, SyncProcessor.syncBlockReqHandler)
	notify.BUS.Subscribe(notify.BlockResponse, SyncProcessor.blockResponseMsgHandler)

	notify.BUS.Subscribe(notify.GroupChainPieceReq, SyncProcessor.groupChainPieceReqHandler)
	notify.BUS.Subscribe(notify.GroupChainPiece, SyncProcessor.groupChainPieceHandler)
	notify.BUS.Subscribe(notify.GroupReq, SyncProcessor.syncGroupReqHandler)
	notify.BUS.Subscribe(notify.GroupResponse, SyncProcessor.groupResponseMsgHandler)
	go SyncProcessor.loop()
}

func (p *syncProcessor) GetCandidateInfo() CandidateInfo {
	return p.candidateInfo
}

func (p *syncProcessor) loop() {
	for {
		select {
		case <-p.broadcastTimer.C:
			go p.broadcastTopBlockInfo(p.blockChain.TopBlock())
		case <-p.syncTimer.C:
			go p.trySyncBlock()
		case <-p.reqTimer.C:
			p.logger.Debugf("Sync to %s time out!", p.candidateInfo.Id)
			p.finishCurrentSync(false)
		}
	}
}

//broadcast local top block info to neighborhood
func (p *syncProcessor) broadcastTopBlockInfo(bh *types.BlockHeader) {
	if bh.Height == 0 {
		return
	}
	topBlockInfo := topBlockInfo{Hash: bh.Hash, TotalQn: bh.TotalQN, Height: bh.Height, PreHash: bh.PreHash}
	topBlockInfo.SignInfo = common.NewSignData(p.privateKey, p.id, &topBlockInfo)

	body, e := marshalTopBlockInfo(topBlockInfo)
	if e != nil {
		p.logger.Errorf("marshal top block info error:%s", e.Error())
		return
	}
	p.logger.Tracef("Send local total qn %d to neighbor!", topBlockInfo.TotalQn)
	message := network.Message{Code: network.TopBlockInfoMsg, Body: body}
	network.GetNetInstance().Broadcast(message)
	p.broadcastTimer.Reset(broadcastBlockInfoInterval)
}

//rcv block info from neighborhood
func (p *syncProcessor) topBlockInfoNotifyHandler(msg notify.Message) {
	bnm, ok := msg.GetData().(*notify.TopBlockInfoMessage)
	if !ok {
		p.logger.Errorf("TopBlockInfoMessage assert not ok!")
		return
	}
	blockInfo, e := unMarshalTopBlockInfo(bnm.BlockInfo)
	if e != nil {
		p.logger.Errorf("Discard message! BlockInfoNotifyMessage unmarshal error:%s", e.Error())
		return
	}
	err := blockInfo.SignInfo.ValidateSign(blockInfo)
	if err != nil {
		p.logger.Errorf("Sign verify error! BlockInfoNotifyMessage:%s", e.Error())
		return
	}
	p.logger.Tracef("Rcv top block info! Height:%d,qn:%d,source:%s", blockInfo.Height, blockInfo.TotalQn, blockInfo.SignInfo.Id)
	topBlock := blockChainImpl.TopBlock()
	localTotalQn, localTopHash := topBlock.TotalQN, topBlock.Hash
	if blockInfo.TotalQn < localTotalQn || (localTotalQn == blockInfo.TotalQn && localTopHash == blockInfo.Hash) {
		return
	}

	source := blockInfo.SignInfo.Id
	if PeerManager.isEvil(source) {
		p.logger.Debugf("[TopBlockInfoNotify]%s is marked evil.Drop!", source)
		return
	}
	p.addCandidate(source, *blockInfo)
}

func (p *syncProcessor) addCandidate(id string, topBlockInfo topBlockInfo) {
	p.lock.Lock("addCandidatePool")
	defer p.lock.Unlock("addCandidatePool")

	if len(p.candidatePool) < syncCandidatePoolSize {
		p.candidatePool[id] = topBlockInfo
		return
	}
	totalQnMinId := ""
	var minTotalQn uint64 = common.MaxUint64
	for id, tbi := range p.candidatePool {
		if tbi.TotalQn <= minTotalQn {
			totalQnMinId = id
			minTotalQn = tbi.TotalQn
		}
	}
	if topBlockInfo.TotalQn > minTotalQn {
		delete(p.candidatePool, totalQnMinId)
		p.candidatePool[id] = topBlockInfo
		go p.trySyncBlock()
	}
}

func (p *syncProcessor) trySyncBlock() {
	if p.syncing {
		p.logger.Debugf("Syncing to %s,do not sync!", p.candidateInfo.Id)
		return
	}
	p.lock.Lock("trySync")
	defer p.lock.Unlock("trySync")
	p.logger.Debugf("Try sync!")
	p.syncTimer.Reset(blockSyncInterval)

	topBlock := blockChainImpl.TopBlock()
	localTotalQN, localHeight := topBlock.TotalQN, topBlock.Height
	p.logger.Tracef("Local totalQn:%d,height:%d,topHash:%s", localTotalQN, localHeight, topBlock.Hash.String())
	candidateInfo := p.chooseSyncCandidate()
	if candidateInfo.Id == "" || candidateInfo.TotalQn <= localTotalQN {
		p.logger.Debugf("No valid candidate for sync!")
		return
	}
	p.logger.Debugf("Begin sync!Candidate:%s!Req block height:%d", candidateInfo.Id, localHeight)
	p.syncing = true
	p.candidateInfo = candidateInfo

	go p.requestBlockChainPiece(candidateInfo.Id, localHeight)
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
	for id, topBlockInfo := range p.candidatePool {
		if topBlockInfo.TotalQn > candidateInfo.TotalQn {
			candidateInfo.Id = id
			candidateInfo.TotalQn = topBlockInfo.TotalQn
			candidateInfo.Height = topBlockInfo.Height
		}
	}
	return candidateInfo
}

func (p *syncProcessor) candidatePoolDump() {
	p.logger.Debugf("Candidate Pool Dump:")
	for id, topBlockInfo := range p.candidatePool {
		p.logger.Debugf("Candidate id:%s,totalQn:%d,height:%d,topHash:%s", id, topBlockInfo.TotalQn, topBlockInfo.Height, topBlockInfo.Hash.String())
	}
}

func (p *syncProcessor) triggerSync() {
	if p.groupFork == nil {
		go p.requestGroupChainPiece(p.candidateInfo.Id, p.groupChain.count)
		return
	}

	if p.groupFork == nil {
		return
	}

	if p.blockFork.waitingGroup && p.groupFork.waitingBlock {
		p.finishCurrentSync(false)
		return
	}
	if p.blockFork.waitingGroup {
		p.tryAcceptGroup()
	} else {
		p.tryAcceptBlock()
	}
}

func (p *syncProcessor) tryMergeFork() bool {
	if p.blockFork == nil {
		return false
	}
	localTopHeader := p.blockChain.latestBlock
	syncLogger.Debugf("Try merge fork.Local chain:%d-%d,fork:%d-%d", localTopHeader.Height, localTopHeader.TotalQN, p.blockFork.latestBlock.Height, p.blockFork.latestBlock.TotalQN)
	if p.blockFork.latestBlock.TotalQN < localTopHeader.TotalQN {
		return false
	}

	var commonAncestor *types.BlockHeader
	for height := p.blockFork.header; height <= p.blockFork.latestBlock.Height; height++ {
		forkBlock := p.blockFork.getBlock(height)
		chainBlockHeader := p.blockChain.QueryBlockHeaderByHeight(height, true)
		if forkBlock == nil || chainBlockHeader == nil {
			break
		}
		if chainBlockHeader.Hash != forkBlock.Header.Hash {
			break
		}
		commonAncestor = forkBlock.Header
	}

	if commonAncestor == nil {
		syncLogger.Debugf("Try merge fork. common ancestor is nil.")
		return false
	}
	syncLogger.Debugf("Try merge fork. common ancestor:%d", commonAncestor.Height)
	if p.blockFork.latestBlock.TotalQN == localTopHeader.TotalQN && p.blockChain.nextPvGreatThanFork(commonAncestor, *p.blockFork) {
		return false
	}

	p.blockChain.removeFromCommonAncestor(commonAncestor)
	if p.groupFork != nil {
		p.groupChain.removeFromCommonAncestor(p.groupFork.getGroup(p.groupFork.header))
	}
	for height := p.blockFork.header + 1; height <= p.blockFork.latestBlock.Height; {
		forkBlock := p.blockFork.getBlock(height)
		if forkBlock == nil {
			return false
		}
		validateCode, consensusVerifyResult := p.blockChain.consensusVerify(forkBlock)
		if consensusVerifyResult {
			var result types.AddBlockResult
			syncLogger.Debugf("add block on chain.%d,%s", forkBlock.Header.Height, forkBlock.Header.Hash.String())
			result = blockChainImpl.addBlockOnChain(forkBlock)
			if result != types.AddBlockSucc {
				return false
			}
			height++
			continue
		} else if validateCode != types.DependOnGroup {
			return false
		}
		p.tryAddGroupOnChain()

		_, consensusVerifyResult = p.blockChain.consensusVerify(forkBlock)
		if consensusVerifyResult {
			return false
		}
		var result types.AddBlockResult
		syncLogger.Debugf("add block on chain.%d,%s", forkBlock.Header.Height, forkBlock.Header.Hash.String())
		result = blockChainImpl.addBlockOnChain(forkBlock)
		if result != types.AddBlockSucc {
			return false
		}
		height++
	}
	return true
}

func (p *syncProcessor) tryAddGroupOnChain() {
	for p.groupFork.current <= p.groupFork.latestGroup.GroupHeight {
		forkGroup := p.groupFork.getGroup(p.groupFork.current)
		if forkGroup == nil {
			return
		}
		//todo verify
		err := p.groupChain.AddGroup(forkGroup)
		if err == nil {
			p.groupFork.current++
			continue
		} else {
			return
		}
	}
}

func (p *syncProcessor) finishCurrentSync(syncResult bool) {
	p.lock.Lock("finish sync")
	p.lock.Unlock("finish sync")

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
	p.reqTimer.Stop()
	p.candidateInfo = CandidateInfo{}
	p.syncing = false
	p.logger.Debugf("finish current sync:%v", syncResult)
}
