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

package types

import (
	"bytes"
	"strconv"

	"com.tuntun.rocket/node/src/common"
)

const (
	TransactionTypeMinerApply         = 2
	TransactionTypeMinerAbort         = 3
	TransactionTypeMinerRefund        = 4
	TransactionTypeMinerAdd           = 5
	TransactionTypeMinerChangeAccount = 6
	TransactionTypeOperatorNode       = 7 // 成为矿主

	//以下交易类型会被外部使用 禁止更改
	TransactionTypeOperatorBalance = 99
	TransactionTypeOperatorEvent   = 100 // 调用状态机/转账

	TransactionTypeETHTX = 188 //以太坊的交易改造而成的交易

	//合约交易
	TransactionTypeContract = 200

	//查询接口
	TransactionTypeGetNetworkId       = 600 //查询Network ID
	TransactionTypeGetChainId         = 601 //查询CHAIN ID
	TransactionTypeGetBlockNumber     = 602 //查询块高
	TransactionTypeGetBlock           = 603 //根据高度或者hash查询块
	TransactionTypeGetNonce           = 604 //查询NONCE
	TransactionTypeGetTx              = 605 //查询交易
	TransactionTypeGetReceipt         = 606 //查询收据
	TransactionTypeGetTxCount         = 607 //查询交易数量
	TransactionTypeGetTxFromBlock     = 608 //根据索引查询块中交易
	TransactionTypeGetContractStorage = 609 //查询合约存储信息
	TransactionTypeGetCode            = 610 //查询CODE

	TransactionTypeGetPastLogs = 611
	TransactionTypeCallVM      = 612
)

type Transaction struct {
	Source string // 用户id
	Target string // 游戏id
	Type   int32  // 场景id
	Time   string

	Data            string // 状态机入参
	ExtraData       string // 在rocketProtocol里，用于转账。包括余额转账、FT转账、NFT转账
	ExtraDataType   int32
	SubTransactions []UserData // 用于存储状态机rpc调用的交易数据
	SubHash         common.Hash

	Hash common.Hash
	Sign *common.Sign

	Nonce           uint64 // 用户级别nonce
	RequestId       uint64 // 消息编号 由网关添加
	SocketRequestId string // websocket id，用于客户端标示请求id，方便回调处理
	ChainId         string //用于区分不同的链
}

//source 在hash计算范围内
//RequestId 不列入hash计算范围
func (tx *Transaction) GenHash() common.Hash {
	if nil == tx {
		return common.Hash{}
	}
	buffer := bytes.Buffer{}

	buffer.Write([]byte(tx.Data))
	buffer.Write([]byte(strconv.FormatUint(tx.Nonce, 10)))
	buffer.Write([]byte(tx.Source))
	buffer.Write([]byte(tx.Target))
	buffer.Write([]byte(strconv.Itoa(int(tx.Type))))
	buffer.Write([]byte(tx.Time))
	buffer.Write([]byte(tx.ExtraData))
	buffer.Write([]byte(tx.ChainId))

	return common.BytesToHash(common.Sha256(buffer.Bytes()))
}

func (tx *Transaction) AppendSubTransaction(sub UserData) {
	tx.SubTransactions = append(tx.SubTransactions, sub)
	buffer := bytes.Buffer{}
	buffer.Write(sub.Hash())
	buffer.Write(tx.SubHash.Bytes())

	//todo: 性能优化点
	tx.SubHash = common.BytesToHash(common.Sha256(buffer.Bytes()))
}

func (tx *Transaction) GenHashes() common.Hashes {
	if nil == tx {
		return common.Hashes{}
	}

	result := common.Hashes{}
	result[0] = tx.Hash
	result[1] = tx.SubHash

	return result
}

type Transactions []*Transaction

func (c Transactions) Len() int {
	return len(c)
}
func (c Transactions) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
func (c Transactions) Less(i, j int) bool {
	if c[i].RequestId == 0 && c[j].RequestId == 0 {
		return c[i].Nonce < c[j].Nonce
	}

	return c[i].RequestId < c[j].RequestId
}

func IsContractTx(txType int32) bool {
	return txType == TransactionTypeETHTX || txType == TransactionTypeContract
}

func IsContractCreateTx(tx Transaction) bool {
	return (tx.Type == TransactionTypeETHTX || tx.Type == TransactionTypeContract) && tx.Target == ""
}
