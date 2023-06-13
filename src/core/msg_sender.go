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
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

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
