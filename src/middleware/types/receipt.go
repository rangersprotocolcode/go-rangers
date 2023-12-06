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

package types

import (
	"bytes"
	"fmt"
	"unsafe"

	"com.tuntun.rocket/node/src/common"
)

//go:generate gencodec -type Receipt -field-override receiptMarshaling -out gen_receipt_json.go

var (
	receiptStatusFailed     = []byte{}
	receiptStatusSuccessful = []byte{0x01}
)

const (
	ReceiptStatusFailed = uint(0)

	ReceiptStatusSuccessful = uint(1)
)

type Receipt struct {
	PostState         []byte         `json:"-"`
	Status            uint           `json:"status"`
	CumulativeGasUsed uint64         `json:"cumulativeGasUsed"`
	Height            uint64         `json:"height"`
	TxHash            common.Hash    `json:"transactionHash" gencodec:"required"`
	Msg               string         `json:"-"`
	Source            string         `json:"-"`
	ContractAddress   common.Address `json:"contractAddress"`
	Logs              []*Log         `json:"logs" gencodec:"required"`
	Result            string         `json:"result,omitempty"`
	GasUsed           uint64         `json:"gasUsed,omitempty"`
}

func NewReceipt(root []byte, failed bool, cumulativeGasUsed uint64, height uint64, msg, source, result string) *Receipt {
	r := &Receipt{PostState: common.CopyBytes(root), CumulativeGasUsed: cumulativeGasUsed, Height: height, Msg: msg, Source: source, Result: result}
	if failed {
		r.Status = ReceiptStatusFailed
	} else {
		r.Status = ReceiptStatusSuccessful
	}
	return r
}

func (r *Receipt) setStatus(postStateOrStatus []byte) error {
	switch {
	case bytes.Equal(postStateOrStatus, receiptStatusSuccessful):
		r.Status = ReceiptStatusSuccessful
	case bytes.Equal(postStateOrStatus, receiptStatusFailed):
		r.Status = ReceiptStatusFailed
	case len(postStateOrStatus) == len(common.Hash{}):
		r.PostState = postStateOrStatus
	default:
		return fmt.Errorf("invalid receipt status %x", postStateOrStatus)
	}
	return nil
}

func (r *Receipt) Size() common.StorageSize {
	size := common.StorageSize(unsafe.Sizeof(*r)) + common.StorageSize(len(r.PostState))

	//size += common.StorageSize(len(r.Logs)) * common.StorageSize(unsafe.Sizeof(Log{}))
	//for _, log := range r.Logs {
	//	size += common.StorageSize(len(log.Topics)*common.HashLength + len(log.Data))
	//}
	return size
}

func (r *Receipt) String() string {
	if len(r.PostState) == 0 {
		return fmt.Sprintf("receipt{status=%d cgas=%v}", r.Status, r.CumulativeGasUsed)
	}
	return fmt.Sprintf("receipt{med=%x cgas=%v}", r.PostState, r.CumulativeGasUsed)
}

type Receipts []*Receipt

func (r Receipts) Len() int { return len(r) }
