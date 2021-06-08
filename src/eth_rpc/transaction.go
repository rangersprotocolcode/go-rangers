// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package eth_rpc

import (
	"com.tuntun.rocket/node/src/common"
	crypto "com.tuntun.rocket/node/src/eth_crypto"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/rlp"
	"com.tuntun.rocket/node/src/utility"
	"encoding/json"
	"errors"
	"golang.org/x/crypto/sha3"
	"io"
	"math/big"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrInvalidSig = errors.New("invalid transaction v, r, s values")
)

type Transaction struct {
	data txdata    // Consensus contents of a transaction
	time time.Time // Time first seen locally (spam avoidance)

	// caches
	hash atomic.Value
	size atomic.Value
	from atomic.Value
}

type txdata struct {
	AccountNonce uint64          `json:"nonce"    gencodec:"required"`
	Price        *big.Int        `json:"gasPrice" gencodec:"required"`
	GasLimit     uint64          `json:"gas"      gencodec:"required"`
	Recipient    *common.Address `json:"to"       rlp:"nil"` // nil means contract creation
	Amount       *big.Int        `json:"value"    gencodec:"required"`
	Payload      []byte          `json:"input"    gencodec:"required"`

	// Signature values
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`

	// This is only used when marshaling to JSON.
	Hash *common.Hash `json:"hash" rlp:"-"`
}

func NewTransaction(nonce uint64, to common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	return newTransaction(nonce, &to, amount, gasLimit, gasPrice, data)
}

func NewContractCreation(nonce uint64, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	return newTransaction(nonce, nil, amount, gasLimit, gasPrice, data)
}

func newTransaction(nonce uint64, to *common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	if len(data) > 0 {
		data = common.CopyBytes(data)
	}
	d := txdata{
		AccountNonce: nonce,
		Recipient:    to,
		Payload:      data,
		Amount:       new(big.Int),
		GasLimit:     gasLimit,
		Price:        new(big.Int),
		V:            new(big.Int),
		R:            new(big.Int),
		S:            new(big.Int),
	}
	if amount != nil {
		d.Amount.Set(amount)
	}
	if gasPrice != nil {
		d.Price.Set(gasPrice)
	}
	return &Transaction{
		data: d,
		time: time.Now(),
	}
}

// ChainId returns which chain id this transaction was signed for (if at all)
func (tx *Transaction) ChainId() *big.Int {
	return deriveChainId(tx.data.V)
}

// Protected returns whether the transaction is protected from replay protection.
func (tx *Transaction) Protected() bool {
	return isProtectedV(tx.data.V)
}

func isProtectedV(V *big.Int) bool {
	if V.BitLen() <= 8 {
		v := V.Uint64()
		return v != 27 && v != 28
	}
	// anything not 27 or 28 is considered protected
	return true
}

// EncodeRLP implements rlp.Encoder
func (tx *Transaction) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &tx.data)
}

// DecodeRLP implements rlp.Decoder
func (tx *Transaction) DecodeRLP(s *rlp.Stream) error {
	_, size, _ := s.Kind()
	err := s.Decode(&tx.data)
	if err == nil {
		tx.size.Store(common.StorageSize(rlp.ListSize(size)))
		tx.time = time.Now()
	}
	return err
}

func (tx *Transaction) Data() []byte       { return common.CopyBytes(tx.data.Payload) }
func (tx *Transaction) Gas() uint64        { return tx.data.GasLimit }
func (tx *Transaction) GasPrice() *big.Int { return new(big.Int).Set(tx.data.Price) }
func (tx *Transaction) GasPriceCmp(other *Transaction) int {
	return tx.data.Price.Cmp(other.data.Price)
}
func (tx *Transaction) GasPriceIntCmp(other *big.Int) int {
	return tx.data.Price.Cmp(other)
}
func (tx *Transaction) Value() *big.Int  { return new(big.Int).Set(tx.data.Amount) }
func (tx *Transaction) Nonce() uint64    { return tx.data.AccountNonce }
func (tx *Transaction) CheckNonce() bool { return true }

// To returns the recipient address of the transaction.
// It returns nil if the transaction is a contract creation.
func (tx *Transaction) To() *common.Address {
	if tx.data.Recipient == nil {
		return nil
	}
	to := *tx.data.Recipient
	return &to
}

// Hash hashes the RLP encoding of tx.
// It uniquely identifies the transaction.
func (tx *Transaction) Hash() common.Hash {
	if hash := tx.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	v := rlpHash(tx)
	tx.hash.Store(v)
	return v
}

// WithSignature returns a new transaction with the given signature.
// This signature needs to be in the [R || S || V] format where V is 0 or 1.
func (tx *Transaction) WithSignature(signer Signer, sig []byte) (*Transaction, error) {
	r, s, v, err := signer.SignatureValues(tx, sig)
	if err != nil {
		return nil, err
	}
	cpy := &Transaction{
		data: tx.data,
		time: tx.time,
	}
	cpy.data.R, cpy.data.S, cpy.data.V = r, s, v
	return cpy, nil
}

// Cost returns amount + gasprice * gaslimit.
func (tx *Transaction) Cost() *big.Int {
	total := new(big.Int).Mul(tx.data.Price, new(big.Int).SetUint64(tx.data.GasLimit))
	total.Add(total, tx.data.Amount)
	return total
}

// RawSignatureValues returns the V, R, S signature values of the transaction.
// The return values should not be modified by the caller.
func (tx *Transaction) RawSignatureValues() (v, r, s *big.Int) {
	return tx.data.V, tx.data.R, tx.data.S
}

// Transactions is a Transaction slice type for basic sorting.
type Transactions []*Transaction

// Len returns the length of s.
func (s Transactions) Len() int { return len(s) }

// Swap swaps the i'th and the j'th element in s.
func (s Transactions) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// GetRlp implements Rlpable and returns the i'th element of s in rlp.
func (s Transactions) GetRlp(i int) []byte {
	enc, _ := rlp.EncodeToBytes(s[i])
	return enc
}

// hasherPool holds LegacyKeccak hashers.
var hasherPool = sync.Pool{
	New: func() interface{} {
		return sha3.NewLegacyKeccak256()
	},
}

func rlpHash(x interface{}) (h common.Hash) {
	sha := hasherPool.Get().(crypto.KeccakState)
	defer hasherPool.Put(sha)
	sha.Reset()
	rlp.Encode(sha, x)
	sha.Read(h[:])
	return h
}

// TxDifference returns a new set which is the difference between a and b.
func TxDifference(a, b Transactions) Transactions {
	keep := make(Transactions, 0, len(a))

	remove := make(map[common.Hash]struct{})
	for _, tx := range b {
		remove[tx.Hash()] = struct{}{}
	}

	for _, tx := range a {
		if _, ok := remove[tx.Hash()]; !ok {
			keep = append(keep, tx)
		}
	}

	return keep
}

func convertTx(txRaw *Transaction, sender common.Address) *types.Transaction {
	result := &types.Transaction{}
	result.Source = sender.String()
	result.Target = txRaw.To().String()
	result.Type = types.TransactionTypeETHTX

	data := service.ContractData{}
	data.AbiData = common.ToHex(txRaw.Data())
	transferValue := txRaw.Value()
	if transferValue != nil {
		data.TransferValue = utility.BigIntToStr(transferValue)
	}
	dataByes, _ := json.Marshal(data)
	result.Data = string(dataByes)
	result.Hash = result.GenHash()
	return result
}

//
//// AsMessage returns the transaction as a core.Message.
////
//// AsMessage requires a signer to derive the sender.
////
//// XXX Rename message to something less arbitrary?
//func (tx *Transaction) AsMessage(s Signer) (Message, error) {
//	msg := Message{
//		nonce:      tx.data.AccountNonce,
//		gasLimit:   tx.data.GasLimit,
//		gasPrice:   new(big.Int).Set(tx.data.Price),
//		to:         tx.data.Recipient,
//		amount:     tx.data.Amount,
//		data:       tx.data.Payload,
//		checkNonce: true,
//	}
//
//	var err error
//	msg.from, err = Sender(s, tx)
//	return msg, err
//}

// Size returns the true RLP encoded storage size of the transaction, either by
// encoding and returning it, or returning a previsouly cached value.
//func (tx *Transaction) Size() common.StorageSize {
//	if size := tx.size.Load(); size != nil {
//		return size.(common.StorageSize)
//	}
//	c := writeCounter(0)
//	rlp.Encode(&c, &tx.data)
//	tx.size.Store(common.StorageSize(c))
//	return common.StorageSize(c)
//}

//type txdataMarshaling struct {
//	AccountNonce hexutil.Uint64
//	Price        *hexutil.Big
//	GasLimit     hexutil.Uint64
//	Amount       *hexutil.Big
//	Payload      hexutil.Bytes
//	V            *hexutil.Big
//	R            *hexutil.Big
//	S            *hexutil.Big
//}
//
//
//// MarshalJSON encodes the web3 RPC transaction format.
//func (tx *Transaction) MarshalJSON() ([]byte, error) {
//	hash := tx.Hash()
//	data := tx.data
//	data.Hash = &hash
//	return data.MarshalJSON()
//}
//
//// UnmarshalJSON decodes the web3 RPC transaction format.
//func (tx *Transaction) UnmarshalJSON(input []byte) error {
//	var dec txdata
//	if err := dec.UnmarshalJSON(input); err != nil {
//		return err
//	}
//	withSignature := dec.V.Sign() != 0 || dec.R.Sign() != 0 || dec.S.Sign() != 0
//	if withSignature {
//		var V byte
//		if isProtectedV(dec.V) {
//			chainID := deriveChainId(dec.V).Uint64()
//			V = byte(dec.V.Uint64() - 35 - 2*chainID)
//		} else {
//			V = byte(dec.V.Uint64() - 27)
//		}
//		if !crypto.ValidateSignatureValues(V, dec.R, dec.S, false) {
//			return ErrInvalidSig
//		}
//	}
//	*tx = Transaction{
//		data: dec,
//		time: time.Now(),
//	}
//	return nil
//}
