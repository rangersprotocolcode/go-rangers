package core

import (
	"bytes"
	"com.tuntun.rocket/node/src/common"
	middleware_pb "com.tuntun.rocket/node/src/middleware/pb"
	"com.tuntun.rocket/node/src/middleware/types"
	"github.com/golang/protobuf/proto"
	"strconv"
)

type TopBlockInfo struct {
	TotalQn uint64
	Hash    common.Hash
	Height  uint64
	PreHash common.Hash

	SignInfo common.SignData
}

func (topBlockInfo *TopBlockInfo) GenHash() common.Hash {
	buffer := bytes.Buffer{}

	buffer.Write([]byte(strconv.FormatUint(topBlockInfo.TotalQn, 10)))
	buffer.Write(topBlockInfo.Hash.Bytes())
	buffer.Write([]byte(strconv.FormatUint(topBlockInfo.Height, 10)))
	buffer.Write(topBlockInfo.PreHash.Bytes())
	return common.BytesToHash(common.Sha256(buffer.Bytes()))
}

func marshalTopBlockInfo(bi TopBlockInfo) ([]byte, error) {
	blockInfo := middleware_pb.TopBlockInfo{Hash: bi.Hash.Bytes(), TotalQn: &bi.TotalQn, Height: &bi.Height, PreHash: bi.PreHash.Bytes()}
	blockInfo.SignInfo = signDataToPb(bi.SignInfo)
	return proto.Marshal(&blockInfo)
}

func unMarshalTopBlockInfo(b []byte) (*TopBlockInfo, error) {
	message := new(middleware_pb.TopBlockInfo)
	e := proto.Unmarshal(b, message)
	if e != nil {
		return nil, e
	}
	blockInfo := TopBlockInfo{Hash: common.BytesToHash(message.Hash), TotalQn: *message.TotalQn, Height: *message.Height, PreHash: common.BytesToHash(message.PreHash)}
	blockInfo.SignInfo = pbToSignData(*message.SignInfo)
	return &blockInfo, nil
}

type ChainPieceReq struct {
	Height   uint64
	SignInfo common.SignData
}

func (chainPieceReq *ChainPieceReq) GenHash() common.Hash {
	buffer := bytes.Buffer{}
	buffer.Write([]byte(strconv.FormatUint(chainPieceReq.Height, 10)))
	return common.BytesToHash(common.Sha256(buffer.Bytes()))
}

func marshalChainPieceReq(req ChainPieceReq) ([]byte, error) {
	chainPieceReq := middleware_pb.ChainPieceReq{Height: &req.Height}
	chainPieceReq.SignInfo = signDataToPb(req.SignInfo)
	return proto.Marshal(&chainPieceReq)
}

func unMarshalChainPieceReq(b []byte) (*ChainPieceReq, error) {
	message := new(middleware_pb.ChainPieceReq)
	e := proto.Unmarshal(b, message)
	if e != nil {
		return nil, e
	}
	chainPieceReq := ChainPieceReq{Height: *message.Height}
	chainPieceReq.SignInfo = pbToSignData(*message.SignInfo)
	return &chainPieceReq, nil
}

type chainPieceInfo struct {
	ChainPiece []*types.BlockHeader
	TopHeader  *types.BlockHeader
	SignInfo   common.SignData
}

func (chainPieceInfo *chainPieceInfo) GenHash() common.Hash {
	buffer := bytes.Buffer{}
	for _, bh := range chainPieceInfo.ChainPiece {
		buffer.Write(bh.Hash.Bytes())
	}
	buffer.Write(chainPieceInfo.TopHeader.Hash.Bytes())
	return common.BytesToHash(common.Sha256(buffer.Bytes()))
}

func marshalChainPieceInfo(chainPieceInfo chainPieceInfo) ([]byte, error) {
	headers := make([]*middleware_pb.BlockHeader, 0)
	for _, header := range chainPieceInfo.ChainPiece {
		h := types.BlockHeaderToPb(header)
		headers = append(headers, h)
	}
	topHeader := types.BlockHeaderToPb(chainPieceInfo.TopHeader)
	message := middleware_pb.ChainPieceInfo{TopHeader: topHeader, BlockHeaders: headers}
	message.SignInfo = signDataToPb(chainPieceInfo.SignInfo)
	return proto.Marshal(&message)
}

func unMarshalChainPieceInfo(b []byte) (*chainPieceInfo, error) {
	message := new(middleware_pb.ChainPieceInfo)
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
	chainPieceInfo := chainPieceInfo{ChainPiece: chainPiece, TopHeader: topHeader}
	chainPieceInfo.SignInfo = pbToSignData(*message.SignInfo)
	return &chainPieceInfo, nil
}

type BlockSyncReq struct {
	Height   uint64
	SignInfo common.SignData
}

func (blockReq *BlockSyncReq) GenHash() common.Hash {
	buffer := bytes.Buffer{}
	buffer.Write([]byte(strconv.FormatUint(blockReq.Height, 10)))
	return common.BytesToHash(common.Sha256(buffer.Bytes()))
}

func marshalBlockSyncReq(req BlockSyncReq) ([]byte, error) {
	blockReq := middleware_pb.BlockReq{Height: &req.Height}
	blockReq.SignInfo = signDataToPb(req.SignInfo)
	return proto.Marshal(&blockReq)
}

func unMarshalBlockSyncReq(b []byte) (*BlockSyncReq, error) {
	message := new(middleware_pb.BlockReq)
	e := proto.Unmarshal(b, message)
	if e != nil {
		return nil, e
	}
	blockReq := BlockSyncReq{Height: *message.Height}
	blockReq.SignInfo = pbToSignData(*message.SignInfo)
	return &blockReq, nil
}

type BlockMsgResponse struct {
	Block       *types.Block
	IsLastBlock bool
	SignInfo    common.SignData
}

func (syncedBlockMessage *BlockMsgResponse) GenHash() common.Hash {
	buffer := bytes.Buffer{}
	buffer.Write(syncedBlockMessage.Block.Header.Hash.Bytes())
	if syncedBlockMessage.IsLastBlock {
		buffer.Write([]byte{0})
	} else {
		buffer.Write([]byte{1})
	}
	return common.BytesToHash(common.Sha256(buffer.Bytes()))
}

func marshalBlockMsgResponse(bmr BlockMsgResponse) ([]byte, error) {
	message := middleware_pb.BlockMsgResponse{IsLast: &bmr.IsLastBlock, Block: types.BlockToPb(bmr.Block)}
	message.SignInfo = signDataToPb(bmr.SignInfo)
	return proto.Marshal(&message)
}

func (bs *blockSyncer) unMarshalBlockMsgResponse(b []byte) (*BlockMsgResponse, error) {
	message := new(middleware_pb.BlockMsgResponse)
	e := proto.Unmarshal(b, message)
	if e != nil {
		bs.logger.Errorf("unMarshalBlockMsgResponse error:%s", e.Error())
		return nil, e
	}
	bmr := BlockMsgResponse{IsLastBlock: *message.IsLast, Block: types.PbToBlock(message.Block)}
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
