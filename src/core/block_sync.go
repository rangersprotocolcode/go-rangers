package core

import (
	"time"

	"x/src/middleware/log"
	"x/src/common"
	"x/src/network"
	"x/src/middleware/notify"
	"x/src/middleware/pb"
	"x/src/utility"
	"x/src/middleware/types"
	"x/src/middleware"

	"github.com/gogo/protobuf/proto"
)

const (
	blockSyncInterval          = 3 * time.Second
	sendTopBlockInfoInterval   = 3 * time.Second
	blockSyncCandidatePoolSize = 3
	blockSyncReqTimeout        = 3 * time.Second
	blockInitDonePeerNum       = 3
)

var BlockSyncer *blockSyncer

type blockSyncer struct {
	syncing       bool
	candidate     string
	candidatePool map[string]TopBlockInfo

	init                 bool
	reqTimeoutTimer      *time.Timer
	syncTimer            *time.Timer
	blockInfoNotifyTimer *time.Timer

	Lock   middleware.Loglock
	logger log.Logger
}

type TopBlockInfo struct {
	TotalQn uint64
	Hash    common.Hash
	Height  uint64
	PreHash common.Hash
}

func InitBlockSyncer() {
	BlockSyncer = &blockSyncer{syncing: false, candidate: "", candidatePool: make(map[string]TopBlockInfo), Lock: middleware.NewLoglock(""), init: false,}
	BlockSyncer.logger = log.GetLoggerByIndex(log.BlockSyncLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	BlockSyncer.reqTimeoutTimer = time.NewTimer(blockSyncReqTimeout)
	BlockSyncer.syncTimer = time.NewTimer(blockSyncInterval)
	BlockSyncer.blockInfoNotifyTimer = time.NewTimer(sendTopBlockInfoInterval)

	notify.BUS.Subscribe(notify.BlockInfoNotify, BlockSyncer.topBlockInfoNotifyHandler)
	notify.BUS.Subscribe(notify.BlockResponse, BlockSyncer.blockResponseMsgHandler)
	go BlockSyncer.loop()
}

func (bs *blockSyncer) IsInit() bool {
	return bs.init
}

func (bs *blockSyncer) GetCandidateForSync() (string, uint64, uint64, bool) {
	topBlock := blockChainImpl.TopBlock()
	localTotalQN, localTopHash, localHeight := topBlock.TotalQN, topBlock.Hash, topBlock.Height
	bs.logger.Debugf("Local totalQn:%d,height:%d,topHash:%s", localTotalQN, localHeight, localTopHash.String())
	bs.candidatePoolDump()

	uselessCandidate := make([]string, 0, blockSyncCandidatePoolSize)
	for id, _ := range bs.candidatePool {
		if PeerManager.isEvil(id) {
			uselessCandidate = append(uselessCandidate, id)
		}
	}
	if len(uselessCandidate) != 0 {
		for _, id := range uselessCandidate {
			delete(bs.candidatePool, id)
		}
	}
	var hasCandidate = false
	if len(bs.candidatePool) >= blockInitDonePeerNum {
		hasCandidate = true
	}

	candidateId := ""
	var candidateMaxTotalQn uint64 = 0
	var candidateHeight uint64 = 0
	for id, topBlockInfo := range bs.candidatePool {
		if topBlockInfo.TotalQn > candidateMaxTotalQn {
			candidateId = id
			candidateMaxTotalQn = topBlockInfo.TotalQn
			candidateHeight = topBlockInfo.Height
		}
	}

	//if candidateHeight == localHeight && candidateMaxTotalQn == localTotalQN {
	//	candidateId = ""
	//}
	if localHeight >= candidateHeight {
		return candidateId, candidateHeight, candidateHeight, hasCandidate
	}
	return candidateId, localHeight + 1, candidateHeight, hasCandidate
}

func (bs *blockSyncer) trySync() {
	bs.Lock.Lock("trySync")
	defer bs.Lock.Unlock("trySync")

	bs.syncTimer.Reset(blockSyncInterval)
	if bs.syncing {
		bs.logger.Debugf("Syncing to %s,do not sync anymore!", bs.candidate)
		return
	}

	id, height, _, hasCandidate := bs.GetCandidateForSync()
	if id == "" {
		bs.logger.Debugf("Get no candidate for sync!")
		if !bs.init && hasCandidate {
			bs.init = true
		}
		return
	}
	bs.logger.Debugf("Get candidate %s for sync!Req height:%d", id, height)
	bs.syncing = true
	bs.candidate = id
	bs.reqTimeoutTimer.Reset(blockSyncReqTimeout)

	go bs.requestBlock(id, height)
}

func (bs *blockSyncer) loop() {
	for {
		select {
		case <-bs.blockInfoNotifyTimer.C:
			topBlock := blockChainImpl.TopBlock()
			topBlockInfo := TopBlockInfo{Hash: topBlock.Hash, TotalQn: topBlock.TotalQN, Height: topBlock.Height, PreHash: topBlock.PreHash}
			go bs.sendTopBlockInfoToNeighbor(topBlockInfo)
		case <-bs.syncTimer.C:
			bs.logger.Debugf("Block sync time up! Try sync")
			go bs.trySync()
		case <-bs.reqTimeoutTimer.C:
			bs.logger.Debugf("Block sync to %s time out!", bs.candidate)
			PeerManager.markEvil(bs.candidate)
			bs.Lock.Lock("req time out")
			bs.syncing = false
			bs.candidate = ""
			bs.Lock.Unlock("req time out")
		}
	}
}

func (bs *blockSyncer) sendTopBlockInfoToNeighbor(bi TopBlockInfo) {
	bs.Lock.Lock("sendTopBlockInfoToNeighbor")
	bs.blockInfoNotifyTimer.Reset(sendTopBlockInfoInterval)
	bs.Lock.Unlock("sendTopBlockInfoToNeighbor")
	if bi.Height == 0 {
		return
	}

	bs.logger.Debugf("Send local total qn %d to neighbor!", bi.TotalQn)
	body, e := marshalBlockInfo(bi)
	if e != nil {
		bs.logger.Errorf("marshal blockInfo error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.BlockInfoNotifyMsg, Body: body}
	network.GetNetInstance().Broadcast(message)
}

func (bs *blockSyncer) topBlockInfoNotifyHandler(msg notify.Message) {
	bnm, ok := msg.GetData().(*notify.BlockInfoNotifyMessage)
	if !ok {
		bs.logger.Errorf("BlockInfoNotifyMessage GetData assert not ok!")
		return
	}
	blockInfo, e := bs.unMarshalTopBlockInfo(bnm.BlockInfo)
	if e != nil {
		bs.logger.Errorf("Discard BlockInfoNotifyMessage because of unmarshal error:%s", e.Error())
		return
	}

	source := bnm.Peer
	topBlock := blockChainImpl.TopBlock()
	localTotalQn, localTopHash := topBlock.TotalQN, topBlock.Hash
	if !bs.isUsefulCandidate(localTotalQn, localTopHash, blockInfo.TotalQn, blockInfo.Hash) {
		return
	}
	bs.addCandidatePool(source, *blockInfo)
}

func (bs *blockSyncer) requestBlock(id string, height uint64) {
	bs.logger.Debugf("Req block to:%s,height:%d", id, height)
	body := utility.UInt64ToByte(height)
	message := network.Message{Code: network.ReqBlock, Body: body}
	go network.GetNetInstance().Send(id, message)
}

func (bs *blockSyncer) blockResponseMsgHandler(msg notify.Message) {
	m, ok := msg.(*notify.BlockResponseMessage)
	if !ok {
		return
	}
	source := m.Peer
	if bs == nil {
		panic("blockSyncer is nil!")
	}
	bs.logger.Debugf("blockResponseMsgHandler rcv from %s!", source)
	if source != bs.candidate {
		bs.logger.Debugf("Unexpected block response from %s, expect from %s!", source, bs.candidate)
		return
	}
	blockResponse, e := bs.unMarshalBlockMsgResponse(m.BlockResponseByte)
	if e != nil {
		bs.logger.Debugf("Discard block response msg because unMarshalBlockMsgResponse error:%d", e.Error())
		return
	}

	block := blockResponse.Block
	isLastBlock := blockResponse.IsLastBlock

	var sync = false
	if block == nil {
		bs.logger.Debugf("Rcv block response nil from:%s", source)
	} else {
		bs.logger.Debugf("Rcv block response from:%s,hash:%v,height:%d,totalQn:%d,tx len:%d,isLastBlock:%t", source, block.Header.Hash.Hex(), block.Header.Height, block.Header.TotalQN, len(block.Transactions), isLastBlock)
		result := blockChainImpl.AddBlockOnChain(source, block, types.Sync)
		if result == types.AddBlockSucc {
			sync = true
		}
	}
	if isLastBlock {
		bs.logger.Debugf("Rcv last block! Set syncing false.Set candidate nil!")
		bs.Lock.Lock("blockResponseMsgHandler")
		bs.candidate = ""
		bs.syncing = false
		bs.reqTimeoutTimer.Stop()
		bs.Lock.Unlock("blockResponseMsgHandler")

		if sync {
			go bs.trySync()
		}
	}
}

func (bs *blockSyncer) addCandidatePool(id string, topBlockInfo TopBlockInfo) {
	if PeerManager.isEvil(id) {
		bs.logger.Debugf("Top block info notify id:%s is marked evil.Drop it!", id)
		return
	}
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
		if !bs.syncing {
			go bs.trySync()
		}
	}
}

func (bs *blockSyncer) isUsefulCandidate(localTotalQn uint64, localTopHash common.Hash, candidateTotalQn uint64, candidateTopHash common.Hash) bool {
	if candidateTotalQn < localTotalQn || (localTotalQn == candidateTotalQn && localTopHash == candidateTopHash) {
		return false
	}
	return true
}

func (bs *blockSyncer) candidatePoolDump() {
	bs.logger.Debugf("Candidate Pool Dump:")
	for id, topBlockInfo := range bs.candidatePool {
		bs.logger.Debugf("Candidate id:%s,totalQn:%d,height:%d,topHash:%s", id, topBlockInfo.TotalQn, topBlockInfo.Height, topBlockInfo.Hash.String())
	}
}

func marshalBlockInfo(bi TopBlockInfo) ([]byte, error) {
	blockInfo := middleware_pb.TopBlockInfo{Hash: bi.Hash.Bytes(), TotalQn: &bi.TotalQn, Height: &bi.Height, PreHash: bi.PreHash.Bytes()}
	return proto.Marshal(&blockInfo)
}

func (bs *blockSyncer) unMarshalTopBlockInfo(b []byte) (*TopBlockInfo, error) {
	message := new(middleware_pb.TopBlockInfo)
	e := proto.Unmarshal(b, message)
	if e != nil {
		bs.logger.Errorf("unMarshalBlockInfo error:%s", e.Error())
		return nil, e
	}
	blockInfo := TopBlockInfo{Hash: common.BytesToHash(message.Hash), TotalQn: *message.TotalQn, Height: *message.Height, PreHash: common.BytesToHash(message.PreHash)}
	return &blockInfo, nil
}

func (bs *blockSyncer) unMarshalBlockMsgResponse(b []byte) (*blockMsgResponse, error) {
	message := new(middleware_pb.BlockMsgResponse)
	e := proto.Unmarshal(b, message)
	if e != nil {
		bs.logger.Errorf("unMarshalBlockMsgResponse error:%s", e.Error())
		return nil, e
	}
	bmr := blockMsgResponse{IsLastBlock: *message.IsLast, Block: types.PbToBlock(message.Block)}
	return &bmr, nil
}
