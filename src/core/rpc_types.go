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
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/service"
	"com.tuntun.rangers/node/src/utility"
	"math/big"
	"strconv"
)

var (
	nonce      = []byte{1, 2, 3, 4, 5, 6, 7, 8}
	difficulty = utility.Big(*big.NewInt(32))
)

type BlockDetail struct {
	RPCBlock
	Trans []RPCTransaction `json:"txDetails"`
}

type RPCBlock struct {
	Version     uint64                    `json:"version"`
	Height      uint64                    `json:"height"`
	Hash        common.Hash               `json:"hash"`
	PreHash     common.Hash               `json:"preHash"`
	CurTime     string                    `json:"curTime"`
	PreTime     string                    `json:"preTime"`
	Castor      groupsig.ID               `json:"proposer"`
	GroupID     groupsig.ID               `json:"groupId"`
	Signature   string                    `json:"sigature"`
	Prove       *big.Int                  `json:"prove"`
	TotalQN     uint64                    `json:"totalQn"`
	Qn          uint64                    `json:"qn"`
	Txs         []interface{}             `json:"transactions"`
	EvictedTxs  []common.Hash             `json:"wrongTxs"`
	TxNum       uint64                    `json:"txCount"`
	StateRoot   common.Hash               `json:"stateRoot"`
	TxRoot      common.Hash               `json:"txRoot"`
	ReceiptRoot common.Hash               `json:"receiptRoot"`
	Random      string                    `json:"random"`
	Reward      map[common.Address]string `json:"reward"`
	HashType    string                    `json:"hashType,omitempty"`
	//adapt eth block
	Number           utility.Uint64 `json:"number,omitempty"`
	GasLimit         utility.Uint64 `json:"gasLimit,omitempty"`
	GasUsed          utility.Uint64 `json:"gasUsed,omitempty"`
	ParentHash       common.Hash    `json:"parentHash,omitempty"`
	Timestamp        utility.Uint64 `json:"timestamp,omitempty"`
	TransactionsRoot common.Hash    `json:"transactionsRoot,omitempty"`
	ReceiptsRoot     common.Hash    `json:"receiptsRoot,omitempty"`
	Miner            common.Address `json:"miner,omitempty"`
	Nonce            utility.Bytes  `json:"nonce,omitempty"`
	Difficulty       *utility.Big   `json:"difficulty,omitempty"`
	ExtraData        utility.Bytes  `json:"extraData,omitempty"`
	TotalReward      string         `json:"totalReward,omitempty"`
	UncleHash        common.Hash    `json:"sha3Uncles,omitempty"`
	Bloom            string         `json:"logsBloom"`
}

type RPCTransaction struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   int32  `json:"type"`

	Signature string `json:"signature"`

	SubTransactions string `json:"subTransactions"`

	Hash common.Hash `json:"hash"`

	Data      string `json:"data"`
	ExtraData string `json:"extraData"`
}

func ConvertBlockWithTxDetail(b *types.Block) *BlockDetail {
	bh := b.Header
	block := ConvertBlockByHeader(bh)

	trans := make([]RPCTransaction, 0)
	for _, tx := range b.Transactions {
		trans = append(trans, *ConvertTransaction(tx))
	}

	return &BlockDetail{
		RPCBlock: *block,
		Trans:    trans,
	}
}

func ConvertBlockByHeader(bh *types.BlockHeader) *RPCBlock {
	block := &RPCBlock{
		Version:      bh.Nonce,
		Height:       bh.Height,
		Hash:         bh.Hash,
		PreHash:      bh.PreHash,
		CurTime:      bh.CurTime.String(),
		PreTime:      bh.PreTime.String(),
		Castor:       groupsig.DeserializeID(bh.Castor),
		GroupID:      groupsig.DeserializeID(bh.GroupId),
		Signature:    common.ToHex(bh.Signature),
		Prove:        bh.ProveValue,
		EvictedTxs:   bh.EvictedTxs,
		TotalQN:      bh.TotalQN,
		StateRoot:    bh.StateTree,
		TxRoot:       bh.TxTree,
		ReceiptRoot:  bh.ReceiptTree,
		Random:       common.ToHex(bh.Random),
		TxNum:        uint64(len(bh.Transactions)),
		Number:       utility.Uint64(bh.Height),
		GasLimit:     utility.Uint64(200000000000),
		GasUsed:      utility.Uint64(200000),
		ParentHash:   bh.PreHash,
		ReceiptsRoot: bh.ReceiptTree,
		Timestamp:    utility.Uint64(bh.CurTime.Unix()),
		Miner:        common.BytesToAddress(bh.Castor),
		Nonce:        utility.Bytes(nonce[:]),
		Difficulty:   &difficulty,
		ExtraData:    utility.Bytes(nonce[:]),
		Bloom:        "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
	}

	totalReward := service.GetTotalReward(bh.Height)
	block.TotalReward = strconv.FormatFloat(totalReward, 'f', -1, 64)

	//adapt eth rpc,return []common.Hash while fullTx is false,[]RPCTransaction while fullTx is true
	block.Txs = make([]interface{}, 0)
	//uncle has to be this value(rlpHash([]*Header(nil))) for pass go ethereum client verify because tx uncles is empty
	block.UncleHash = common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347")
	//transactionsRoot  has to be this value(EmptyRootHash) for pass go ethereum client verify because tx uncles is empty
	block.TransactionsRoot = common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")

	preBlock := GetBlockChain().QueryBlockByHash(bh.PreHash)
	if preBlock != nil {
		block.Qn = bh.TotalQN - preBlock.Header.TotalQN
	} else {
		block.Qn = bh.TotalQN
	}

	return block
}

func ConvertTransaction(tx *types.Transaction) *RPCTransaction {
	trans := &RPCTransaction{
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

	return trans
}
