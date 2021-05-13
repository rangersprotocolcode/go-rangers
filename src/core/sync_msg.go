package core

import (
	"bytes"
	"com.tuntun.rocket/node/src/common"
	middleware_pb "com.tuntun.rocket/node/src/middleware/pb"
	"com.tuntun.rocket/node/src/middleware/types"
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

type blockChainPieceReq struct {
	Height   uint64
	SignInfo common.SignData
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

type blockChainPiece struct {
	ChainPiece []*types.BlockHeader
	TopHeader  *types.BlockHeader
	SignInfo   common.SignData
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

type blockSyncReq struct {
	Height   uint64
	SignInfo common.SignData
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

type blockMsgResponse struct {
	Block       *types.Block
	IsLastBlock bool
	SignInfo    common.SignData
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

type groupChainPieceReq struct {
	Height   uint64
	SignInfo common.SignData
}

func (chainPieceReq *groupChainPieceReq) GenHash() common.Hash {
	buffer := bytes.Buffer{}
	buffer.Write([]byte(strconv.FormatUint(chainPieceReq.Height, 10)))
	return common.BytesToHash(common.Sha256(buffer.Bytes()))
}

func marshalGroupChainPieceReq(req groupChainPieceReq) ([]byte, error) {
	chainPieceReq := middleware_pb.GroupChainPieceReq{Height: &req.Height}
	chainPieceReq.SignInfo = signDataToPb(req.SignInfo)
	return proto.Marshal(&chainPieceReq)
}

func unMarshalGroupChainPieceReq(b []byte) (*groupChainPieceReq, error) {
	message := new(middleware_pb.GroupChainPieceReq)
	e := proto.Unmarshal(b, message)
	if e != nil {
		return nil, e
	}
	chainPieceReq := groupChainPieceReq{Height: *message.Height}
	chainPieceReq.SignInfo = pbToSignData(*message.SignInfo)
	return &chainPieceReq, nil
}

type groupChainPiece struct {
	GroupChainPiece []*types.Group
	SignInfo        common.SignData
}

func (groupChainPiece *groupChainPiece) GenHash() common.Hash {
	buffer := bytes.Buffer{}
	for _, group := range groupChainPiece.GroupChainPiece {
		buffer.Write(group.Header.Hash.Bytes())
	}
	return common.BytesToHash(common.Sha256(buffer.Bytes()))
}

func marshalGroupChainPiece(chainPieceInfo groupChainPiece) ([]byte, error) {
	groups := make([]*middleware_pb.Group, 0)
	for _, group := range chainPieceInfo.GroupChainPiece {
		g := types.GroupToPb(group)
		groups = append(groups, g)
	}
	message := middleware_pb.GroupChainPiece{Groups: groups}
	message.SignInfo = signDataToPb(chainPieceInfo.SignInfo)
	return proto.Marshal(&message)
}

func unMarshalGroupChainPiece(b []byte) (*groupChainPiece, error) {
	message := new(middleware_pb.GroupChainPiece)
	e := proto.Unmarshal(b, message)
	if e != nil {
		return nil, e
	}

	chainPiece := make([]*types.Group, 0)
	for _, group := range message.Groups {
		g := types.PbToGroup(group)
		chainPiece = append(chainPiece, g)
	}
	chainPieceInfo := groupChainPiece{GroupChainPiece: chainPiece}
	chainPieceInfo.SignInfo = pbToSignData(*message.SignInfo)
	return &chainPieceInfo, nil
}

type groupSyncReq struct {
	Height   uint64
	SignInfo common.SignData
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

type groupMsgResponse struct {
	Group       *types.Group
	IsLastGroup bool
	SignInfo    common.SignData
}

func (groupMsgResponse *groupMsgResponse) GenHash() common.Hash {
	buffer := bytes.Buffer{}
	if groupMsgResponse != nil {
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
