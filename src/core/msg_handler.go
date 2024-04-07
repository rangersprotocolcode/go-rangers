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
	"com.tuntun.rangers/node/src/middleware/notify"
	middleware_pb "com.tuntun.rangers/node/src/middleware/pb"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/network"
	"com.tuntun.rangers/node/src/service"
	"com.tuntun.rangers/node/src/utility"
	"github.com/golang/protobuf/proto"
	"math/big"
)

type ChainHandler struct{}

func initChainHandler() {
	handler := ChainHandler{}

	notify.BUS.Subscribe(notify.NewBlock, handler)
	notify.BUS.Subscribe(notify.TransactionReq, handler)
}

func (c ChainHandler) HandleNetMessage(topic string, msg notify.Message) {
	switch topic {
	case notify.NewBlock:
		c.newBlockHandler(msg)
	case notify.TransactionReq:
		c.transactionReqHandler(msg)
	}
}

func (c ChainHandler) Handle(sourceId string, msg network.Message) error {
	return nil
}

func (c ChainHandler) transactionReqHandler(msg notify.Message) {
	trm, ok := msg.(*notify.TransactionReqMessage)
	if !ok {
		logger.Debugf("transactionReqHandler:Message assert not ok!")
		return
	}

	m, e := unMarshalTransactionRequestMessage(trm.TransactionReqByte)
	if e != nil {
		logger.Errorf("unmarshal transaction request message error:%s", e.Error())
		return
	}

	source := trm.Peer
	logger.Debugf("receive transaction req from %s,block height:%d,block hash:%s,tx_len:%d", source, m.BlockHeight, m.CurrentBlockHash.String(), len(m.TransactionHashes))
	if nil == blockChainImpl {
		logger.Errorf("no blockChainImpl, cannot find txs")
		return
	}

	transactions, need, e := blockChainImpl.queryTxsByBlockHash(m.CurrentBlockHash, m.TransactionHashes)
	if e == service.ErrNil {
		m.TransactionHashes = need
	}

	logger.Debugf("local find txs, length: %d, source: %s", len(transactions), source)
	if nil != transactions && 0 != len(transactions) {
		sendTransactions(transactions, source)
	}
}

func (c ChainHandler) newBlockHandler(msg notify.Message) {
	m, ok := msg.(*notify.NewBlockMessage)
	if !ok {
		return
	}
	source := m.Peer
	block, e := types.UnMarshalBlock(m.BlockByte)
	if e != nil {
		logger.Debugf("UnMarshal block error:%d", e.Error())
		return
	}

	middleware.PerfLogger.Infof("Rcv new block from %s,hash: %v,height: %d,totalQn: %d,tx: %d, cost: %v, size: %d", source, block.Header.Hash.Hex(), block.Header.Height, block.Header.TotalQN, len(block.Transactions), utility.GetTime().Sub(block.Header.CurTime), len(m.BlockByte))

	blockChainImpl.AddBlockOnChain(block)
}

func unMarshalTransactionRequestMessage(b []byte) (*transactionRequestMessage, error) {
	m := new(middleware_pb.TransactionRequestMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("UnMarshal transaction request message error:%s", e.Error())
		return nil, e
	}

	txHashes := make([]common.Hashes, 0)
	for _, txHash := range m.TransactionHashes {
		hashes := common.Hashes{}
		hashes[0] = common.BytesToHash(txHash.Hash)
		hashes[1] = common.BytesToHash(txHash.SubHash)

		txHashes = append(txHashes, hashes)
	}

	currentBlockHash := common.BytesToHash(m.CurrentBlockHash)
	blockPv := &big.Int{}
	blockPv.SetBytes(m.BlockPv)
	message := transactionRequestMessage{TransactionHashes: txHashes, CurrentBlockHash: currentBlockHash, BlockHeight: *m.BlockHeight, BlockPv: blockPv}
	return &message, nil
}
