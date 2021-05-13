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
	SyncProcessor = &syncProcessor{privateKey: privateKey, id: id, syncing: false, candidatePool: make(map[string]chainInfo)}

	SyncProcessor.broadcastTimer = time.NewTimer(broadcastBlockInfoInterval)
	SyncProcessor.reqTimer = time.NewTimer(syncReqTimeout)
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
			go p.broadcastChainInfo(p.blockChain.TopBlock())
		case <-p.syncTimer.C:
			go p.trySyncBlock()
		case <-p.reqTimer.C:
			p.logger.Debugf("Sync to %s time out!", p.candidateInfo.Id)
			p.finishCurrentSync(false)
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

//rcv chain info from neighborhood
func (p *syncProcessor) chainInfoNotifyHandler(msg notify.Message) {
	bnm, ok := msg.GetData().(*notify.ChainInfoMessage)
	if !ok {
		p.logger.Errorf("ChainInfoMessage assert not ok!")
		return
	}
	chainInfo, e := unMarshalChainInfo(bnm.ChainInfo)
	if e != nil {
		p.logger.Errorf("Discard message! ChainInfoMessage unmarshal error:%s", e.Error())
		return
	}
	err := chainInfo.SignInfo.ValidateSign(chainInfo)
	if err != nil {
		p.logger.Errorf("Sign verify error! ChainInfoMessage:%s", e.Error())
		return
	}
	syncHandleLogger.Tracef("Rcv chain info! Height:%d,qn:%d,group height:%d,source:%s", chainInfo.TopBlockHeight, chainInfo.TotalQn, chainInfo.TopGroupHeight, chainInfo.SignInfo.Id)
	topBlock := blockChainImpl.TopBlock()
	localTotalQn, localTopHash := topBlock.TotalQN, topBlock.Hash
	localGroupHeight := p.groupChain.height()
	if chainInfo.TotalQn < localTotalQn {
		return
	}

	if localTotalQn == chainInfo.TotalQn && localTopHash == chainInfo.TopBlockHash && localGroupHeight >= chainInfo.TopGroupHeight {
		return
	}
	source := chainInfo.SignInfo.Id
	if PeerManager.isEvil(source) {
		p.logger.Debugf("[chainInfoNotifyHandler]%s is marked evil.Drop!", source)
		return
	}
	p.addCandidate(source, *chainInfo)
}

func (p *syncProcessor) addCandidate(id string, chainInfo chainInfo) {
	p.lock.Lock("addCandidatePool")
	defer p.lock.Unlock("addCandidatePool")

	if len(p.candidatePool) < syncCandidatePoolSize {
		p.candidatePool[id] = chainInfo
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
	if chainInfo.TotalQn >= minTotalQn {
		delete(p.candidatePool, totalQnMinId)
		p.candidatePool[id] = chainInfo
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
	if candidateInfo.TotalQn > localTotalQN {
		p.logger.Debugf("Begin sync!Candidate:%s!Req block height:%d", candidateInfo.Id, localBlockHeight)
		go p.requestBlockChainPiece(candidateInfo.Id, localBlockHeight)
	} else {
		p.logger.Debugf("Begin sync!Candidate:%s!Req group height:%d,candidate group height:%d", candidateInfo.Id, localGroupHeight, candidateInfo.GroupHeight)
		go p.requestGroupChainPiece(candidateInfo.Id, localGroupHeight)
	}

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
		if chainInfo.TotalQn >= candidateInfo.TotalQn {
			candidateInfo.Id = id
			candidateInfo.TotalQn = chainInfo.TotalQn
			candidateInfo.Height = chainInfo.TopBlockHeight
			candidateInfo.GroupHeight = chainInfo.TopGroupHeight
		}
	}
	return candidateInfo
}

func (p *syncProcessor) candidatePoolDump() {
	p.logger.Debugf("Candidate Pool Dump:")
	for id, chainInfo := range p.candidatePool {
		p.logger.Debugf("Candidate id:%s,totalQn:%d,height:%d,topHash:%s,groupHeight:%d", id, chainInfo.TotalQn, chainInfo.TopBlockHeight, chainInfo.TopBlockHash.String(), chainInfo.TopGroupHeight)
	}
}

func (p *syncProcessor) triggerSync() {
	if p.groupFork == nil {
		go p.requestGroupChainPiece(p.candidateInfo.Id, p.groupChain.height())
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
	p.tryAddGroupOnChain()
	return true
}

func (p *syncProcessor) tryAddGroupOnChain() bool {
	if p.groupFork == nil {
		return false
	}
	p.logger.Debugf("try add group on chain...", p.groupFork.current)
	for p.groupFork.current <= p.groupFork.latestGroup.GroupHeight {
		forkGroup := p.groupFork.getGroup(p.groupFork.current)
		if forkGroup == nil {
			return false
		}
		//todo verify
		err := p.groupChain.AddGroup(forkGroup)
		if err == nil {
			p.groupFork.current++
			p.logger.Debugf("add group on chain success.%d-%s", forkGroup.GroupHeight, common.ToHex(forkGroup.Id))
			continue
		} else {
			p.logger.Debugf("add group on chain failed.%d-%s,err:%s", forkGroup.GroupHeight, common.ToHex(forkGroup.Id), err.Error())
			return false
		}
	}
	return true
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
