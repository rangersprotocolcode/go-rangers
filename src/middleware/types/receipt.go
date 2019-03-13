
package types

import (
	"bytes"
	"fmt"
	"unsafe"

	"x/src/common"
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
	PostState         []byte `json:"-"`
	Status            uint   `json:"status"`
	CumulativeGasUsed uint64 `json:"cumulativeGasUsed"`
	Bloom             Bloom  `json:"-"`
	Logs              []*Log `json:"logs"`

	TxHash          common.Hash    `json:"transactionHash" gencodec:"required"`
	ContractAddress common.Address `json:"contractAddress"`
}

func NewReceipt(root []byte, failed bool, cumulativeGasUsed uint64) *Receipt {
	r := &Receipt{PostState: common.CopyBytes(root), CumulativeGasUsed: cumulativeGasUsed}
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
		return fmt.Sprintf("receipt{status=%d cgas=%v bloom=%x}", r.Status, r.CumulativeGasUsed, r.Bloom)
	}
	return fmt.Sprintf("receipt{med=%x cgas=%v bloom=%x}", r.PostState, r.CumulativeGasUsed, r.Bloom)
}

type Receipts []*Receipt

func (r Receipts) Len() int { return len(r) }
