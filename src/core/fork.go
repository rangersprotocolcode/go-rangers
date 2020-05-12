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

	"github.com/golang/protobuf/proto"
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
	chainPiece := fh.chain.getChainPieceInfo(reqHeight)
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
	status, reqHeight := fh.chain.processChainPieceInfo(chainPieceInfo.ChainPiece, chainPieceInfo.TopHeader)
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

	blocks := fh.chain.getChainPieceBlocks(reqHeight)
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
	fh.chain.mergeFork(blocks, topHeader)
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
