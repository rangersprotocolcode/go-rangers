//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package core

import (
	"math/big"

	"x/src/network"
	"x/src/common"
	"x/src/utility"
	"x/src/middleware/types"
	"x/src/middleware/notify"

	"github.com/gogo/protobuf/proto"
	"taschain/src/middleware/pb"
)

const blockResponseSize = 1

type ChainHandler struct{}

func initChainHandler(){
	handler := ChainHandler{}

	notify.BUS.Subscribe(notify.BlockReq, handler.blockReqHandler)
	notify.BUS.Subscribe(notify.NewBlock, handler.newBlockHandler)
	notify.BUS.Subscribe(notify.TransactionBroadcast, handler.transactionBroadcastHandler)
	notify.BUS.Subscribe(notify.TransactionReq, handler.transactionReqHandler)
	notify.BUS.Subscribe(notify.TransactionGot, handler.transactionGotHandler)
}

func (c *ChainHandler) Handle(sourceId string, msg network.Message) error {
	return nil
}

func (ch ChainHandler) transactionBroadcastHandler(msg notify.Message) {
	mtm, ok := msg.(*notify.TransactionBroadcastMessage)
	if !ok {
		logger.Debugf("transactionBroadcastHandler:Message assert not ok!")
		return
	}
	txs, e := types.UnMarshalTransactions(mtm.TransactionsByte)
	if e != nil {
		logger.Errorf("Unmarshal transactions error:%s", e.Error())
		return
	}
	BlockChainImpl.GetTransactionPool().AddBroadcastTransactions(txs)
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
	logger.Debugf("receive transaction req from %s,%d-%D,tx_len", source, m.BlockHeight, m.CurrentBlockHash.String(), len(m.TransactionHashes))
	if nil == BlockChainImpl {
		return
	}
	transactions, need, e := BlockChainImpl.GetTransactions(m.CurrentBlockHash, m.TransactionHashes)
	if e == ErrNil {
		m.TransactionHashes = need
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
	BlockChainImpl.GetTransactionPool().AddMissTransactions(txs)

	m := notify.TransactionGotAddSuccMessage{Transactions: txs, Peer: tgm.Peer}
	notify.BUS.Publish(notify.TransactionGotAddSucc, &m)
	return
}

func (ch ChainHandler) blockReqHandler(msg notify.Message) {
	if BlockChainImpl.IsLightMiner() {
		logger.Debugf("Is light miner!")
		return
	}

	m, ok := msg.(*notify.BlockReqMessage)
	if !ok {
		logger.Debugf("blockReqHandler:Message assert not ok!")
		return
	}
	reqHeight := utility.ByteToUInt64(m.HeightByte)
	localHeight := BlockChainImpl.Height()

	logger.Debugf("Rcv block request:reqHeight:%d,localHeight:%d", reqHeight, localHeight)
	var count = 0
	for i := reqHeight; i <= localHeight; i++ {
		block := BlockChainImpl.QueryBlock(i)
		if block == nil {
			continue
		}
		count++
		if count == blockResponseSize || i == localHeight {
			sendBlock(m.Peer, block, true)
		} else {
			sendBlock(m.Peer, block, false)
		}
		if count >= blockResponseSize {
			break
		}
	}
	if count == 0 {
		sendBlock(m.Peer, nil, true)
	}
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

	logger.Debugf("Rcv new block from %s,hash:%v,height:%d,totalQn:%d,tx len:%d", source, block.Header.Hash.Hex(), block.Header.Height, block.Header.TotalQN, len(block.Transactions))
	BlockChainImpl.AddBlockOnChain(source, block, types.NewBlock)
}

func unMarshalTransactionRequestMessage(b []byte) (*transactionRequestMessage, error) {
	m := new(tas_middleware_pb.TransactionRequestMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		network.Logger.Errorf("UnMarshal transaction request message error:%s", e.Error())
		return nil, e
	}

	txHashes := make([]common.Hash, 0)
	for _, txHash := range m.TransactionHashes {
		txHashes = append(txHashes, common.BytesToHash(txHash))
	}

	currentBlockHash := common.BytesToHash(m.CurrentBlockHash)
	blockPv := &big.Int{}
	blockPv.SetBytes(m.BlockPv)
	message := transactionRequestMessage{TransactionHashes: txHashes, CurrentBlockHash: currentBlockHash, BlockHeight: *m.BlockHeight, BlockPv: blockPv}
	return &message, nil
}
