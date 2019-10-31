package core

import (
	"time"
	"sync"

	"x/src/utility"
	"x/src/network"
	"x/src/common"
	"x/src/middleware/log"
	"x/src/middleware/notify"
	"x/src/middleware/types"
	"x/src/middleware/pb"

	"github.com/gogo/protobuf/proto"
	"math"
)

const forkTimeOut = 3 * time.Second

type forkProcessor struct {
	candidate string
	reqTimer  *time.Timer

	lock   sync.Mutex
	logger log.Logger

	chain *blockChain
}

type ChainPieceBlockMsg struct {
	Blocks    []*types.Block
	TopHeader *types.BlockHeader
}

func initForkProcessor(chain *blockChain) *forkProcessor {
	fh := forkProcessor{lock: sync.Mutex{}, reqTimer: time.NewTimer(forkTimeOut), chain: chain}
	fh.logger = log.GetLoggerByIndex(log.ForkLogConfig, common.GlobalConf.GetString("instance", "index", ""))

	notify.BUS.Subscribe(notify.ChainPieceInfoReq, fh.chainPieceInfoReqHandler)
	notify.BUS.Subscribe(notify.ChainPieceInfo, fh.chainPieceInfoHandler)
	notify.BUS.Subscribe(notify.ChainPieceBlockReq, fh.chainPieceBlockReqHandler)
	notify.BUS.Subscribe(notify.ChainPieceBlock, fh.chainPieceBlockHandler)

	go fh.loop()
	return &fh
}

func (fh *forkProcessor) requestChainPieceInfo(targetNode string, height uint64) {
	if BlockSyncer == nil {
		return
	}
	if targetNode == "" {
		return
	}
	if fh.candidate != "" {
		fh.logger.Debugf("Processing fork to %s! Do not req chain piece info anymore", fh.candidate)
		return
	}

	if PeerManager.isEvil(targetNode) {
		fh.logger.Debugf("Req id:%s is marked evil.Do not req!", targetNode)
		return
	}

	fh.lock.Lock()
	fh.candidate = targetNode
	fh.reqTimer.Reset(forkTimeOut)
	fh.lock.Unlock()
	fh.logger.Debugf("Req chain piece info to:%s,local height:%d", targetNode, height)
	body := utility.UInt64ToByte(height)
	message := network.Message{Code: network.ChainPieceInfoReq, Body: body}
	network.GetNetInstance().Send(targetNode, message)
}

func (fh *forkProcessor) chainPieceInfoReqHandler(msg notify.Message) {
	chainPieceReqMessage, ok := msg.GetData().(*notify.ChainPieceInfoReqMessage)
	if !ok {
		return
	}
	reqHeight := utility.ByteToUInt64(chainPieceReqMessage.HeightByte)
	id := chainPieceReqMessage.Peer

	fh.logger.Debugf("Rcv chain piece info req from:%s,req height:%d", id, reqHeight)
	chainPiece := fh.getChainPieceInfo(reqHeight)
	fh.sendChainPieceInfo(id, chainPieceInfo{ChainPiece: chainPiece, TopHeader: blockChainImpl.TopBlock()})
}

func (fh *forkProcessor) sendChainPieceInfo(targetNode string, chainPieceInfo chainPieceInfo) {
	chainPiece := chainPieceInfo.ChainPiece
	if len(chainPiece) == 0 {
		return
	}
	fh.logger.Debugf("Send chain piece %d-%d to:%s", chainPiece[len(chainPiece)-1].Height, chainPiece[0].Height, targetNode)
	body, e := marshalChainPieceInfo(chainPieceInfo)
	if e != nil {
		fh.logger.Errorf("Marshal chain piece info error:%s!", e.Error())
		return
	}
	message := network.Message{Code: network.ChainPieceInfo, Body: body}
	network.GetNetInstance().Send(targetNode, message)
}

func (fh *forkProcessor) chainPieceInfoHandler(msg notify.Message) {
	chainPieceInfoMessage, ok := msg.GetData().(*notify.ChainPieceInfoMessage)
	if !ok {
		return
	}
	chainPieceInfo, err := fh.unMarshalChainPieceInfo(chainPieceInfoMessage.ChainPieceInfoByte)
	if err != nil {
		fh.logger.Errorf("Unmarshal chain piece info error:%s", err.Error())
		return
	}
	source := chainPieceInfoMessage.Peer
	if source != fh.candidate {
		fh.logger.Debugf("Unexpected chain piece info from %s, expect from %s!", source, chainPieceInfoMessage.Peer)
		PeerManager.markEvil(source)
		return
	}
	if !fh.verifyChainPieceInfo(chainPieceInfo.ChainPiece, chainPieceInfo.TopHeader) {
		fh.logger.Debugf("Bad chain piece info from %s", source)
		PeerManager.markEvil(source)
		return
	}
	status, reqHeight := fh.processChainPieceInfo(chainPieceInfo.ChainPiece, chainPieceInfo.TopHeader)
	if status == 0 {
		fh.reset()
		return
	}
	if status == 1 {
		fh.requestChainPieceBlock(source, reqHeight)
		return
	}

	if status == 2 {
		fh.reset()
		fh.requestChainPieceInfo(source, reqHeight)
		return
	}
}

func (fh *forkProcessor) requestChainPieceBlock(id string, height uint64) {
	fh.logger.Debugf("Req chain piece block to:%s,height:%d", id, height)
	body := utility.UInt64ToByte(height)
	message := network.Message{Code: network.ReqChainPieceBlock, Body: body}
	go network.GetNetInstance().Send(id, message)
}

func (fh *forkProcessor) chainPieceBlockReqHandler(msg notify.Message) {
	m, ok := msg.GetData().(*notify.ChainPieceBlockReqMessage)
	if !ok {
		return
	}
	source := m.Peer
	reqHeight := utility.ByteToUInt64(m.ReqHeightByte)
	fh.logger.Debugf("Rcv chain piece block req from:%s,req height:%d", source, reqHeight)

	blocks := fh.getChainPieceBlocks(reqHeight)
	topHeader := blockChainImpl.TopBlock()
	fh.sendChainPieceBlock(source, blocks, topHeader)
}

func (fh *forkProcessor) sendChainPieceBlock(targetId string, blocks []*types.Block, topHeader *types.BlockHeader) {
	fh.logger.Debugf("Send chain piece blocks %d-%d to:%s", blocks[len(blocks)-1].Header.Height, blocks[0].Header.Height, targetId)
	body, e := fh.marshalChainPieceBlockMsg(ChainPieceBlockMsg{Blocks: blocks, TopHeader: topHeader})
	if e != nil {
		fh.logger.Errorf("Marshal chain piece block msg error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.ChainPieceBlock, Body: body}
	go network.GetNetInstance().Send(targetId, message)
}

func (fh *forkProcessor) chainPieceBlockHandler(msg notify.Message) {
	m, ok := msg.GetData().(*notify.ChainPieceBlockMessage)
	if !ok {
		return
	}
	source := m.Peer
	if source != fh.candidate {
		fh.logger.Debugf("Unexpected chain piece block from %s, expect from %s!", source, fh.candidate)
		PeerManager.markEvil(source)
		return
	}

	chainPieceBlockMsg, e := fh.unmarshalChainPieceBlockMsg(m.ChainPieceBlockMsgByte)
	if e != nil {
		fh.logger.Debugf("Unmarshal chain piece block msg error:%d", e.Error())
		return
	}

	blocks := chainPieceBlockMsg.Blocks
	topHeader := chainPieceBlockMsg.TopHeader

	if topHeader == nil {
		return
	}
	fh.logger.Debugf("Rcv chain piece block chain piece blocks %d-%d from %s", blocks[len(blocks)-1].Header.Height, blocks[0].Header.Height, source)

	if !fh.verifyChainPieceBlocks(blocks, topHeader) {
		fh.logger.Debugf("Bad chain piece blocks from %s", source)
		PeerManager.markEvil(source)
		return
	}
	fh.mergeFork(blocks, topHeader)
	fh.reset()

}

func (fh *forkProcessor) reset() {
	fh.lock.Lock()
	defer fh.lock.Unlock()
	fh.logger.Debugf("Fork processor reset!")
	fh.candidate = ""
	fh.reqTimer.Stop()
}

func (fh *forkProcessor) verifyChainPieceInfo(chainPiece []*types.BlockHeader, topHeader *types.BlockHeader) bool {
	if len(chainPiece) == 0 {
		return false
	}
	if topHeader.Hash != topHeader.GenHash() {
		logger.Infof("Invalid topHeader! Hash:%s", topHeader.Hash.String())
		return false
	}

	for i := 0; i < len(chainPiece)-1; i++ {
		bh := chainPiece[i]
		if bh.Hash != bh.GenHash() {
			logger.Infof("Invalid chainPiece element,hash:%s", bh.Hash.String())
			return false
		}
		if bh.PreHash != chainPiece[i+1].Hash {
			logger.Infof("Invalid preHash,expect prehash:%s,real hash:%s", bh.PreHash.String(), chainPiece[i+1].Hash.String())
			return false
		}
	}
	return true
}

func (fh *forkProcessor) verifyChainPieceBlocks(chainPiece []*types.Block, topHeader *types.BlockHeader) bool {
	if len(chainPiece) == 0 {
		return false
	}
	if topHeader.Hash != topHeader.GenHash() {
		fh.logger.Infof("Invalid topHeader! Hash:%s", topHeader.Hash.String())
		return false
	}

	for i := len(chainPiece) - 1; i > 0; i-- {
		block := chainPiece[i]
		if block == nil {
			return false
		}
		if block.Header.Hash != block.Header.GenHash() {
			fh.logger.Infof("Invalid chainPiece element,hash:%s", block.Header.Hash.String())
			return false
		}
		if block.Header.PreHash != chainPiece[i-1].Header.Hash {
			fh.logger.Infof("Invalid preHash,expect preHash:%s,real hash:%s", block.Header.PreHash.String(), chainPiece[i+1].Header.Hash.String())
			return false
		}
	}
	return true
}

func (fh *forkProcessor) unMarshalChainPieceInfo(b []byte) (*chainPieceInfo, error) {
	message := new(middleware_pb.ChainPieceInfo)
	e := proto.Unmarshal(b, message)
	if e != nil {
		fh.logger.Errorf("UnMarshal chain piece info error:%s", e.Error())
		return nil, e
	}

	chainPiece := make([]*types.BlockHeader, 0)
	for _, header := range message.BlockHeaders {
		h := types.PbToBlockHeader(header)
		chainPiece = append(chainPiece, h)
	}
	topHeader := types.PbToBlockHeader(message.TopHeader)
	chainPieceInfo := chainPieceInfo{ChainPiece: chainPiece, TopHeader: topHeader}
	return &chainPieceInfo, nil
}

func (fh *forkProcessor) marshalChainPieceBlockMsg(cpb ChainPieceBlockMsg) ([]byte, error) {
	topHeader := types.BlockHeaderToPb(cpb.TopHeader)
	blocks := make([]*middleware_pb.Block, 0)
	for _, b := range cpb.Blocks {
		blocks = append(blocks, types.BlockToPb(b))
	}
	message := middleware_pb.ChainPieceBlockMsg{TopHeader: topHeader, Blocks: blocks}
	return proto.Marshal(&message)
}

func (fh *forkProcessor) unmarshalChainPieceBlockMsg(b []byte) (*ChainPieceBlockMsg, error) {
	message := new(middleware_pb.ChainPieceBlockMsg)
	e := proto.Unmarshal(b, message)
	if e != nil {
		fh.logger.Errorf("Unmarshal chain piece block msg error:%s", e.Error())
		return nil, e
	}
	topHeader := types.PbToBlockHeader(message.TopHeader)
	blocks := make([]*types.Block, 0)
	for _, b := range message.Blocks {
		blocks = append(blocks, types.PbToBlock(b))
	}
	cpb := ChainPieceBlockMsg{TopHeader: topHeader, Blocks: blocks}
	return &cpb, nil
}

func (fh *forkProcessor) loop() {
	for {
		select {
		case <-fh.reqTimer.C:
			if fh.candidate != "" {
				fh.logger.Debugf("Fork req time out to  %s", fh.candidate)
				PeerManager.markEvil(fh.candidate)
				fh.reset()
			}
		}
	}
}

func (fh *forkProcessor) getChainPieceInfo(reqHeight uint64) []*types.BlockHeader {
	fh.chain.lock.Lock("GetChainPieceInfo")
	defer fh.chain.lock.Unlock("GetChainPieceInfo")
	localHeight := fh.chain.latestBlock.Height

	fh.logger.Debugf("Req chain piece info height:%d,local height:%d", reqHeight, localHeight)

	var height uint64
	if reqHeight > localHeight {
		height = localHeight
	} else {
		height = reqHeight
	}

	chainPiece := make([]*types.BlockHeader, 0)

	var lastChainPieceBlock *types.BlockHeader
	for i := height; i <= fh.chain.Height(); i++ {
		bh := fh.chain.queryBlockHeaderByHeight(i, true)
		if nil == bh {
			continue
		}
		lastChainPieceBlock = bh
		break
	}
	if lastChainPieceBlock == nil {
		fh.logger.Errorf("Last chain piece block should not be nil!")
		return chainPiece
	}

	chainPiece = append(chainPiece, lastChainPieceBlock)

	hash := lastChainPieceBlock.PreHash
	for i := 0; i < chainPieceLength; i++ {
		header := fh.chain.queryBlockHeaderByHash(hash)
		if header == nil {
			//创世块 pre hash 不存在
			break
		}
		chainPiece = append(chainPiece, header)
		hash = header.PreHash
	}
	return chainPiece
}

func (fh *forkProcessor) getChainPieceBlocks(reqHeight uint64) []*types.Block {
	fh.chain.lock.Lock("GetChainPieceBlocks")
	defer fh.chain.lock.Unlock("GetChainPieceBlocks")
	localHeight := fh.chain.latestBlock.Height
	fh.logger.Debugf("Req chain piece block height:%d,local height:%d", reqHeight, localHeight)

	var height uint64
	if reqHeight > localHeight {
		height = localHeight
	} else {
		height = reqHeight
	}

	var firstChainPieceBlock *types.BlockHeader
	for i := height; i <= fh.chain.Height(); i++ {
		bh := fh.chain.queryBlockHeaderByHeight(i, true)
		if nil == bh {
			continue
		}
		firstChainPieceBlock = bh
		break
	}
	if firstChainPieceBlock == nil {
		panic("last chain piece block should not be nil!")
	}

	chainPieceBlocks := make([]*types.Block, 0)
	for i := firstChainPieceBlock.Height; i <= fh.chain.Height(); i++ {
		bh := fh.chain.queryBlockHeaderByHeight(i, true)
		if nil == bh {
			continue
		}
		b := fh.chain.queryBlockByHash(bh.Hash)
		if nil == b {
			continue
		}
		chainPieceBlocks = append(chainPieceBlocks, b)
		if len(chainPieceBlocks) > chainPieceBlockLength {
			break
		}
	}
	return chainPieceBlocks
}

//status 0 忽略该消息  不需要同步
//status 1 需要同步ChainPieceBlock
//status 2 需要继续同步ChainPieceInfo
func (fh *forkProcessor) processChainPieceInfo(chainPiece []*types.BlockHeader, topHeader *types.BlockHeader) (status int, reqHeight uint64) {
	fh.chain.lock.Lock("ProcessChainPieceInfo")
	defer fh.chain.lock.Unlock("ProcessChainPieceInfo")

	localTopHeader := fh.chain.latestBlock
	if topHeader.TotalQN < localTopHeader.TotalQN {
		return 0, math.MaxUint64
	}
	fh.logger.Debugf("ProcessChainPiece %d-%d,topHeader height:%d,totalQn:%d,hash:%v", chainPiece[len(chainPiece)-1].Height, chainPiece[0].Height, topHeader.Height, topHeader.TotalQN, topHeader.Hash.Hex())
	commonAncestor, hasCommonAncestor, index := fh.findCommonAncestor(chainPiece, 0, len(chainPiece)-1)
	if hasCommonAncestor {
		fh.logger.Debugf("Got common ancestor! Height:%d,localHeight:%d", commonAncestor.Height, localTopHeader.Height)
		if topHeader.TotalQN > localTopHeader.TotalQN {
			return 1, commonAncestor.Height + 1
		}

		if topHeader.TotalQN == fh.chain.latestBlock.TotalQN {
			var remoteNext *types.BlockHeader
			for i := index - 1; i >= 0; i-- {
				if chainPiece[i].ProveValue != nil {
					remoteNext = chainPiece[i]
					break
				}
			}
			if remoteNext == nil {
				return 0, math.MaxUint64
			}
			if fh.chain.compareValue(commonAncestor, remoteNext) {
				fh.logger.Debugf("Local value is great than coming value!")
				return 0, math.MaxUint64
			}
			fh.logger.Debugf("Coming value is great than local value!")
			return 1, commonAncestor.Height + 1
		}
		return 0, math.MaxUint64
	}
	//Has no common ancestor
	if index == 0 {
		fh.logger.Debugf("Local chain is same with coming chain piece.")
		return 1, chainPiece[0].Height + 1
	} else {
		var preHeight uint64
		preBlock := fh.chain.queryBlockByHash(fh.chain.latestBlock.PreHash)
		if preBlock != nil {
			preHeight = preBlock.Header.Height
		} else {
			preHeight = 0
		}
		lastPieceHeight := chainPiece[len(chainPiece)-1].Height

		var minHeight uint64
		if preHeight < lastPieceHeight {
			minHeight = preHeight
		} else {
			minHeight = lastPieceHeight
		}
		var baseHeight uint64
		if minHeight != 0 {
			baseHeight = minHeight - 1
		} else {
			baseHeight = 0
		}
		fh.logger.Debugf("Do not find common ancestor in chain piece info:%d-%d!Continue to request chain piece info,base height:%d", chainPiece[len(chainPiece)-1].Height, chainPiece[0].Height, baseHeight, )
		return 2, baseHeight
	}

}

func (fh *forkProcessor) mergeFork(blockChainPiece []*types.Block, topHeader *types.BlockHeader) {
	if topHeader == nil || len(blockChainPiece) == 0 {
		return
	}
	fh.chain.lock.Lock("MergeFork")
	defer fh.chain.lock.Unlock("MergeFork")

	localTopHeader := fh.chain.latestBlock
	if blockChainPiece[len(blockChainPiece)-1].Header.TotalQN < localTopHeader.TotalQN {
		return
	}

	if blockChainPiece[len(blockChainPiece)-1].Header.TotalQN == localTopHeader.TotalQN {
		if !fh.compareNextBlockPv(blockChainPiece[0].Header) {
			return
		}
	}

	originCommonAncestorHash := (*blockChainPiece[0]).Header.PreHash
	originCommonAncestor := fh.chain.queryBlockByHash(originCommonAncestorHash)
	if originCommonAncestor == nil {
		return
	}

	var index = -100
	for i := 0; i < len(blockChainPiece); i++ {
		block := blockChainPiece[i]
		if fh.chain.queryBlockByHash(block.Header.Hash) == nil {
			index = i - 1
			break
		}
	}

	if index == -100 {
		return
	}

	var realCommonAncestor *types.BlockHeader
	if index == -1 {
		realCommonAncestor = originCommonAncestor.Header
	} else {
		realCommonAncestor = blockChainPiece[index].Header
	}
	fh.chain.removeFromCommonAncestor(realCommonAncestor)

	for i := index + 1; i < len(blockChainPiece); i++ {
		block := blockChainPiece[i]
		var result types.AddBlockResult
		result = fh.chain.addBlockOnChain("", block, types.MergeFork)
		if result != types.AddBlockSucc {
			return
		}
	}
}

func (fh *forkProcessor) compareNextBlockPv(remoteNextHeader *types.BlockHeader) bool {
	if remoteNextHeader == nil {
		return false
	}
	remoteNextBlockPv := remoteNextHeader.ProveValue
	if remoteNextBlockPv == nil {
		return false
	}
	commonAncestor := fh.chain.queryBlockByHash(remoteNextHeader.PreHash)
	if commonAncestor == nil {
		fh.logger.Debugf("MergeFork common ancestor should not be nil!")
		return false
	}

	var localNextBlock *types.BlockHeader
	for i := commonAncestor.Header.Height + 1; i <= fh.chain.Height(); i++ {
		bh := fh.chain.queryBlockHeaderByHeight(i, true)
		if nil == bh {
			continue
		}
		localNextBlock = bh
		break
	}
	if localNextBlock == nil {
		return true
	}
	if remoteNextBlockPv.Cmp(localNextBlock.ProveValue) > 0 {
		return true
	}
	return false
}

func (fh *forkProcessor) findCommonAncestor(chainPiece []*types.BlockHeader, l int, r int) (*types.BlockHeader, bool, int) {
	if l > r {
		return nil, false, -1
	}

	m := (l + r) / 2
	result := fh.isCommonAncestor(chainPiece, m)
	if result == 0 {
		return chainPiece[m], true, m
	}

	if result == 1 {
		return fh.findCommonAncestor(chainPiece, l, m-1)
	}

	if result == -1 {
		return fh.findCommonAncestor(chainPiece, m+1, r)
	}
	if result == 100 {
		return nil, false, 0
	}
	return nil, false, -1
}

//bhs 中没有空值
//返回值
// 0  当前HASH相等，后面一块HASH不相等 是共同祖先
//1   当前HASH相等，后面一块HASH相等
//100  当前HASH相等，但是到达数组边界，找不到后面一块 无法判断同祖先
//-1  当前HASH不相等
//-100 参数不合法
func (fh *forkProcessor) isCommonAncestor(chainPiece []*types.BlockHeader, index int) int {
	if index < 0 || index >= len(chainPiece) {
		return -100
	}
	he := chainPiece[index]

	bh := fh.chain.queryBlockHeaderByHeight(he.Height, true)
	if bh == nil {
		fh.logger.Debugf("isCommonAncestor:Height:%d,local hash:%x,coming hash:%x\n", he.Height, nil, he.Hash)
		return -1
	}
	fh.logger.Debugf("isCommonAncestor:Height:%d,local hash:%x,coming hash:%x\n", he.Height, bh.Hash, he.Hash)
	if index == 0 && bh.Hash == he.Hash {
		return 100
	}
	if index == 0 {
		return -1
	}
	//判断链更后面的一块
	afterHe := chainPiece[index-1]
	afterBh := fh.chain.queryBlockHeaderByHeight(afterHe.Height, true)
	if afterBh == nil {
		fh.logger.Debugf("isCommonAncestor:after block height:%d,local hash:%s,coming hash:%x\n", afterHe.Height, "null", afterHe.Hash)
		if afterHe != nil && bh.Hash == he.Hash {
			return 0
		}
		return -1
	}
	fh.logger.Debugf("isCommonAncestor:after block height:%d,local hash:%x,coming hash:%x\n", afterHe.Height, afterBh.Hash, afterHe.Hash)
	if afterHe.Hash != afterBh.Hash && bh.Hash == he.Hash {
		return 0
	}
	if afterHe.Hash == afterBh.Hash && bh.Hash == he.Hash {
		return 1
	}
	return -1
}
