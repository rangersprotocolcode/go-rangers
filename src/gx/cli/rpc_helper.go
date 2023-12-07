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

package cli

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/middleware/types"
	"encoding/json"
)

func convertTransaction(tx *types.Transaction) *Transaction {
	trans := &Transaction{
		Hash:      tx.Hash,
		Source:    tx.Source,
		Target:    tx.Target,
		Type:      tx.Type,
		Data:      tx.Data,
		ExtraData: tx.ExtraData,
	}

	if tx.Sign != nil {
		trans.Signature = tx.Sign.GetHexString()
	}

	data, err := json.Marshal(tx.SubTransactions)
	if err == nil {
		trans.SubTransactions = string(data)
	}

	return trans
}

func convertBlockHeader(bh *types.BlockHeader) *Block {
	block := &Block{
		Version:     bh.Nonce,
		Height:      bh.Height,
		Hash:        bh.Hash,
		PreHash:     bh.PreHash,
		CurTime:     bh.CurTime.String(),
		PreTime:     bh.PreTime.String(),
		Castor:      groupsig.DeserializeID(bh.Castor),
		GroupID:     groupsig.DeserializeID(bh.GroupId),
		Signature:   common.ToHex(bh.Signature),
		Prove:       bh.ProveValue,
		EvictedTxs:  bh.EvictedTxs,
		TotalQN:     bh.TotalQN,
		StateRoot:   bh.StateTree,
		TxRoot:      bh.TxTree,
		ReceiptRoot: bh.ReceiptTree,
		Random:      common.ToHex(bh.Random),
		TxNum:       uint64(len(bh.Transactions)),
	}

	block.Txs = make([]common.Hash, len(bh.Transactions))
	if 0 != len(bh.Transactions) {
		for i, tx := range bh.Transactions {
			block.Txs[i] = tx[0]
		}
	}

	return block
}
