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

package types

import (
	"bytes"
	"com.tuntun.rangers/node/src/common"
	"math/big"
	"strconv"
)

const (
	TransactionTypeMinerApply         = 2
	TransactionTypeMinerAbort         = 3
	TransactionTypeMinerRefund        = 4
	TransactionTypeMinerAdd           = 5
	TransactionTypeMinerChangeAccount = 6
	TransactionTypeOperatorNode       = 7 // 成为矿主

	TransactionTypeOperatorBalance = 99
	TransactionTypeOperatorEvent   = 100

	TransactionTypeETHTX = 188

	TransactionTypeContract = 200

	TransactionTypeGetNetworkId       = 600
	TransactionTypeGetChainId         = 601
	TransactionTypeGetBlockNumber     = 602
	TransactionTypeGetBlock           = 603
	TransactionTypeGetNonce           = 604
	TransactionTypeGetTx              = 605
	TransactionTypeGetReceipt         = 606
	TransactionTypeGetTxCount         = 607
	TransactionTypeGetTxFromBlock     = 608
	TransactionTypeGetContractStorage = 609
	TransactionTypeGetCode            = 610

	TransactionTypeGetPastLogs = 611
	TransactionTypeCallVM      = 612
)

type Transaction struct {
	Source string
	Target string
	Type   int32
	Time   string

	Data            string
	ExtraData       string
	ExtraDataType   int32
	SubTransactions []UserData
	SubHash         common.Hash

	Hash common.Hash
	Sign *common.Sign

	Nonce           uint64
	RequestId       uint64
	SocketRequestId string

	ChainId string
}

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
		if c[i].Source == c[j].Source {
			return c[i].Nonce < c[j].Nonce
		}

		num1 := new(big.Int).SetBytes(c[i].Hash.Bytes())
		num2 := new(big.Int).SetBytes(c[j].Hash.Bytes())
		return num1.Cmp(num2) > 0
	}
	return c[i].RequestId < c[j].RequestId
}

func IsContractTx(txType int32) bool {
	return txType == TransactionTypeETHTX || txType == TransactionTypeContract
}
