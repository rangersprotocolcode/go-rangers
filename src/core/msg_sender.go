package core

import (
	"com.tuntun.rocket/node/src/utility"
	"math/big"

	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/pb"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/network"

	"github.com/gogo/protobuf/proto"
)

type transactionRequestMessage struct {
	TransactionHashes []common.Hashes
	CurrentBlockHash  common.Hash
	BlockHeight       uint64
	BlockPv           *big.Int
}

type chainPieceInfo struct {
	ChainPiece []*types.BlockHeader
	TopHeader  *types.BlockHeader
}

type blockMsgResponse struct {
	Block       *types.Block
	IsLastBlock bool
}

func requestTransaction(m transactionRequestMessage, castorId string) {
	if castorId == "" {
		return
	}

	body, e := marshalTransactionRequestMessage(&m)
	if e != nil {
		logger.Errorf("Discard MarshalTransactionRequestMessage because of marshal error:%s!", e.Error())
		return
	}
	logger.Debugf("send REQ_TRANSACTION_MSG to %s,height:%d,tx_len:%d,hash:%s,time at:%v", castorId, m.BlockHeight, len(m.TransactionHashes), m.CurrentBlockHash.String(), utility.GetTime())
	message := network.Message{Code: network.ReqTransactionMsg, Body: body}
	network.GetNetInstance().Broadcast(message)
}

func sendTransactions(txs []*types.Transaction, sourceId string) {
	body, e := types.MarshalTransactions(txs)
	if e != nil {
		logger.Errorf("Discard MarshalTransactions because of marshal error:%s!", e.Error())
		return
	}
	message := network.Message{Code: network.TransactionGotMsg, Body: body}
	go network.GetNetInstance().Send(sourceId, message)
}

func sendBlock(targetId string, block *types.Block, isLastBlock bool) {
	if block == nil {
		logger.Debugf("Send nil block to:%s", targetId)
	} else {
		logger.Debugf("Send local block:%d to:%s,isLastBlock:%t", block.Header.Height, targetId, isLastBlock)
	}
	body, e := marshalBlockMsgResponse(blockMsgResponse{Block: block, IsLastBlock: isLastBlock})
	if e != nil {
		logger.Errorf("Marshal block msg response error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.BlockResponseMsg, Body: body}
	network.GetNetInstance().Send(targetId, message)
}

func marshalTransactionRequestMessage(m *transactionRequestMessage) ([]byte, error) {
	txHashes := make([]*middleware_pb.TransactionHash, 0)
	for _, txHash := range m.TransactionHashes {
		hashes := &middleware_pb.TransactionHash{}
		hashes.Hash = txHash[0].Bytes()
		hashes.SubHash = txHash[1].Bytes()
		txHashes = append(txHashes, hashes)

	}

	currentBlockHash := m.CurrentBlockHash.Bytes()
	message := middleware_pb.TransactionRequestMessage{TransactionHashes: txHashes, CurrentBlockHash: currentBlockHash, BlockHeight: &m.BlockHeight, BlockPv: m.BlockPv.Bytes()}
	return proto.Marshal(&message)
}

func marshalChainPieceInfo(chainPieceInfo chainPieceInfo) ([]byte, error) {
	headers := make([]*middleware_pb.BlockHeader, 0)
	for _, header := range chainPieceInfo.ChainPiece {
		h := types.BlockHeaderToPb(header)
		headers = append(headers, h)
	}
	topHeader := types.BlockHeaderToPb(chainPieceInfo.TopHeader)
	message := middleware_pb.ChainPieceInfo{TopHeader: topHeader, BlockHeaders: headers}
	return proto.Marshal(&message)
}

func marshalBlockMsgResponse(bmr blockMsgResponse) ([]byte, error) {
	message := middleware_pb.BlockMsgResponse{IsLast: &bmr.IsLastBlock, Block: types.BlockToPb(bmr.Block)}
	return proto.Marshal(&message)
}
