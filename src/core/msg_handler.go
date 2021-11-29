// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"math/big"

	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/notify"
	"com.tuntun.rocket/node/src/middleware/pb"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/network"
	"com.tuntun.rocket/node/src/service"
	"github.com/golang/protobuf/proto"
	"time"
)

const blockResponseSize = 1

type ChainHandler struct{}

func initChainHandler() {
	handler := ChainHandler{}

	notify.BUS.Subscribe(notify.NewBlock, handler.newBlockHandler)
	notify.BUS.Subscribe(notify.TransactionReq, handler.transactionReqHandler)
	notify.BUS.Subscribe(notify.TransactionGot, handler.transactionGotHandler)
}

func (c *ChainHandler) Handle(sourceId string, msg network.Message) error {
	return nil
}

func (ch ChainHandler) transactionReqHandler(msg notify.Message) {
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
	logger.Debugf("receive transaction req from %s,block height:%d,block hash:%s,tx_len%d", source, m.BlockHeight, m.CurrentBlockHash.String(), len(m.TransactionHashes))
	if nil == blockChainImpl {
		return
	}
	transactions, need, e := blockChainImpl.queryTxsByBlockHash(m.CurrentBlockHash, m.TransactionHashes)
	if e == service.ErrNil {
		m.TransactionHashes = need
	}

	for _, tx := range transactions {
		if tx == nil {
			logger.Debugf("local find nil tx")
			continue
		}

		logger.Debugf("local find tx :%s,%v", tx.Hash.String(), tx)
	}
	if nil != transactions && 0 != len(transactions) {
		sendTransactions(transactions, source)
	}
	return
}

func (ch ChainHandler) transactionGotHandler(msg notify.Message) {
	tgm, ok := msg.(*notify.TransactionGotMessage)
	if !ok {
		logger.Debugf("transactionGotHandler:Message assert not ok!")
		return
	}

	txs, e := types.UnMarshalTransactions(tgm.TransactionGotByte)
	if e != nil {
		logger.Errorf("Unmarshal got transactions error:%s", e.Error())
		return
	}
	for _, tx := range txs {
		service.GetTransactionPool().AddTransaction(tx)
	}

	m := notify.TransactionGotAddSuccMessage{Transactions: txs, Peer: tgm.Peer}
	notify.BUS.Publish(notify.TransactionGotAddSucc, &m)
	return
}

func (ch ChainHandler) newBlockHandler(msg notify.Message) {
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

	middleware.PerfLogger.Debugf("Rcv new block from %s,hash:%v,height:%d,totalQn:%d,tx len:%d, total cost: %v", source, block.Header.Hash.Hex(), block.Header.Height, block.Header.TotalQN, len(block.Transactions), time.Since(block.Header.CurTime))

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
