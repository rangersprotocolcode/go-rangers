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

package service

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/db"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"errors"
	"github.com/hashicorp/golang-lru"
	"sort"

	"encoding/json"
	"fmt"
)

const (
	txDataBasePrefix = "tx"
	rcvTxPoolSize    = 50000
	txCacheSize      = 1000

	txCountPerBlock = 3000
)

var (
	ErrNil = errors.New("nil transaction")

	ErrChainId = errors.New("illegal chain id")

	ErrHash = errors.New("illegal transaction hash")

	ErrSign = errors.New("illegal transaction signature")

	ErrExist = errors.New("transaction already exist in pool")

	ErrEvicted = errors.New("error transaction already exist in pool")
)

type ExecutedTransaction struct {
	Receipt     *types.Receipt
	Transaction []byte
}

type TransactionPool interface {
	PackForCast() []*types.Transaction

	//add new transaction to the transaction pool
	AddTransaction(tx *types.Transaction) (bool, error)

	GetTransaction(hash common.Hash) (*types.Transaction, error)

	GetTransactionStatus(hash common.Hash) (uint, error)

	GetExecuted(hash common.Hash) *ExecutedTransaction

	GetReceived() []*types.Transaction

	TxNum() int

	MarkExecuted(receipts types.Receipts, txs []*types.Transaction, evictedTxs []common.Hash)

	UnMarkExecuted(txs []*types.Transaction, evictedTxs []common.Hash)

	Clear()

	IsExisted(hash common.Hash) bool

	VerifyTransaction(tx *types.Transaction) error

	ProcessFee(tx types.Transaction, accountDB *account.AccountDB) error
}

type TxPool struct {
	evictedTxs *lru.Cache
	received   *simpleContainer

	executed db.Database
	batch    db.Batch

	lock middleware.Loglock
}

var txpoolInstance TransactionPool

func initTransactionPool() {
	if nil == txpoolInstance {
		txpoolInstance = newTransactionPool()
	}
}

func GetTransactionPool() TransactionPool {
	return txpoolInstance
}

func newTransactionPool() TransactionPool {
	pool := &TxPool{
		lock: middleware.NewLoglock("txPool"),
	}
	pool.received = newSimpleContainer(rcvTxPoolSize)
	pool.evictedTxs, _ = lru.New(txCacheSize)

	executed, err := db.NewLDBDatabase(txDataBasePrefix, 16, 128)
	if err != nil {
		txPoolLogger.Errorf("Init transaction pool error! Error:%s", err.Error())
		return nil
	}
	pool.executed = executed
	pool.batch = pool.executed.NewBatch()

	return pool
}

func (pool *TxPool) AddTransaction(tx *types.Transaction) (bool, error) {
	if pool.evictedTxs.Contains(tx.Hash) {
		txPoolLogger.Infof("Tx is marked evicted tx,do not add pool. Hash:%s", tx.Hash.String())
		return false, ErrEvicted
	}

	pool.lock.Lock("AddTransaction")
	defer pool.lock.Unlock("AddTransaction")
	b, err := pool.add(tx)
	return b, err
}

func (pool *TxPool) MarkExecuted(receipts types.Receipts, txs []*types.Transaction, evictedTxs []common.Hash) {
	if nil == receipts || 0 == len(receipts) {
		return
	}
	pool.lock.RLock("MarkExecuted")
	defer pool.lock.RUnlock("MarkExecuted")

	for i, receipt := range receipts {
		hash := receipt.TxHash
		executedTx := &ExecutedTransaction{
			Receipt: receipt,
		}
		tx := findTxInList(txs, hash, i)
		txData, _ := types.MarshalTransaction(tx)
		executedTx.Transaction = txData
		executedTxBytes, err := json.Marshal(executedTx)
		if nil != err {
			continue
		}
		pool.batch.Put(hash.Bytes(), executedTxBytes)
		if pool.batch.ValueSize() > 100*1024 {
			pool.batch.Write()
			pool.batch.Reset()
		}
	}
	if pool.batch.ValueSize() > 0 {
		pool.batch.Write()
		pool.batch.Reset()
	}

	for _, tx := range txs {
		pool.remove(tx.Hash)
	}
	if evictedTxs != nil {
		for _, hash := range evictedTxs {
			pool.remove(hash)
			pool.evictedTxs.Add(hash, 0)
		}
	}
}

func (pool *TxPool) UnMarkExecuted(txs []*types.Transaction, evictedTxs []common.Hash) {
	if nil == txs || 0 == len(txs) {
		return
	}
	pool.lock.RLock("UnMarkExecuted")
	defer pool.lock.RUnlock("UnMarkExecuted")

	if evictedTxs != nil {
		for _, hash := range evictedTxs {
			pool.evictedTxs.Remove(hash)
		}
	}

	for _, tx := range txs {
		pool.executed.Delete(tx.Hash.Bytes())
		pool.add(tx)
	}
}

func (pool *TxPool) IsExisted(hash common.Hash) bool {
	return pool.isTransactionExisted(hash)
}

func (pool *TxPool) GetTransaction(hash common.Hash) (*types.Transaction, error) {
	receivedTx := pool.received.get(hash)
	if nil != receivedTx {
		return receivedTx, nil
	}

	executedTx := pool.GetExecuted(hash)
	if nil != executedTx {
		tx, _ := types.UnMarshalTransaction(executedTx.Transaction)
		return &tx, nil
	}
	return nil, ErrNil
}

func (pool *TxPool) GetTransactionStatus(hash common.Hash) (uint, error) {
	executedTx := pool.GetExecuted(hash)
	if executedTx == nil {
		return 0, ErrNil
	}
	return executedTx.Receipt.Status, nil
}

func (pool *TxPool) Clear() {
	pool.lock.Lock("Clear")
	defer pool.lock.Unlock("Clear")

	executed, _ := db.NewDatabase(txDataBasePrefix)
	pool.executed = executed
	pool.batch.Reset()

	pool.received = newSimpleContainer(rcvTxPoolSize)
}

func (pool *TxPool) GetReceived() []*types.Transaction {
	return pool.received.txs
}

func (pool *TxPool) GetExecuted(hash common.Hash) *ExecutedTransaction {
	txBytes, _ := pool.executed.Get(hash.Bytes())
	if txBytes == nil {
		return nil
	}

	var executedTx *ExecutedTransaction
	err := json.Unmarshal(txBytes, &executedTx)
	if err != nil || executedTx == nil {
		return nil
	}
	return executedTx
}

func (p *TxPool) TxNum() int {
	return p.received.Len()
}

func (pool *TxPool) PackForCast() []*types.Transaction {
	packedTxs := make([]*types.Transaction, 0)
	txs := pool.received.asSlice()
	for _, tx := range txs {
		packedTxs = append(packedTxs, tx)
		txPoolLogger.Debugf("Pack tx:%s", tx.Hash.String())
		if len(packedTxs) >= txCountPerBlock {
			break
		}
	}
	sort.Sort(types.Transactions(packedTxs))
	// transactions 已经根据RequestId排序
	return packedTxs
}

func (pool *TxPool) VerifyTransaction(tx *types.Transaction) error {
	err := pool.verifyTxChainId(tx)
	if nil != err {
		return err
	}
	err = pool.verifyTransactionHash(tx)
	if nil != err {
		return err
	}
	err = pool.verifyTransactionSign(tx)
	if nil != err {
		return err
	}

	txLogger.Debugf("Verify tx success. hash: %s", tx.Hash.String())
	return nil
}

func (pool *TxPool) ProcessFee(tx types.Transaction, accountDB *account.AccountDB) error {
	addr := common.HexStringToAddress(tx.Source)
	balance := accountDB.GetBalance(addr)

	delta, _ := utility.StrToBigInt("0.0001")
	if balance.Cmp(delta) < 0 {
		msg := fmt.Sprintf("not enough max, addr: %s, balance: %s", tx.Source, balance)
		return fmt.Errorf(msg)
	}
	accountDB.SubBalance(addr, delta)
	return nil
}

func (pool *TxPool) add(tx *types.Transaction) (bool, error) {
	if tx == nil {
		return false, ErrNil
	}

	hash := tx.Hash
	if pool.isTransactionExisted(hash) {
		return false, ErrExist
	}
	pool.received.push(tx)
	txPoolLogger.Debugf("[pool]Add tx:%s. After add,received size:%d", tx.Hash.String(), pool.received.Len())
	return true, nil
}

func (pool *TxPool) remove(txHash common.Hash) {
	pool.received.remove(txHash)
	txPoolLogger.Debugf("[pool]Remove tx:%s. After remove, received size:%d", txHash.String(), pool.received.Len())
}

func (pool *TxPool) isTransactionExisted(hash common.Hash) bool {
	existInReceivedTxs := pool.received.contains(hash)
	if existInReceivedTxs {
		return true
	}

	isExecutedTx, _ := pool.executed.Has(hash.Bytes())
	return isExecutedTx
}

func findTxInList(txs []*types.Transaction, txHash common.Hash, receiptIndex int) *types.Transaction {
	if nil == txs || 0 == len(txs) {
		return nil
	}
	if txs[receiptIndex].Hash == txHash {
		return txs[receiptIndex]
	}

	for _, tx := range txs {
		if tx.Hash == txHash {
			return tx
		}
	}
	return nil
}

func (pool *TxPool) verifyTxChainId(tx *types.Transaction) error {
	if tx.ChainId != common.ChainId() {
		txLogger.Errorf("Verify chain id error!Hash:%s,chainId:%s,expect chainId:%s", tx.Hash.String(), tx.ChainId, common.ChainId())
		return ErrChainId
	}
	return nil
}

func (pool *TxPool) verifyTransactionHash(tx *types.Transaction) error {
	expectHash := tx.GenHash()
	if tx.Hash != expectHash {
		txLogger.Errorf("Verify tx hash error!Hash:%s,expect hash:%s", tx.Hash.String(), expectHash.String())
		return ErrHash
	}
	return nil
}

func (pool *TxPool) verifyTransactionSign(tx *types.Transaction) error {
	if tx.Sign == nil {
		txLogger.Errorf("Verify tx sign error!Hash:%s,error:nil sign!", tx.Hash.String())
		return ErrSign
	}

	hashByte := tx.Hash.Bytes()
	pk, err := tx.Sign.RecoverPubkey(hashByte)
	if err != nil {
		txLogger.Errorf("Verify tx sign error!Hash:%s,error:%s", tx.Hash.String(), err.Error())
		return ErrSign
	}
	if !pk.Verify(hashByte, tx.Sign) {
		txLogger.Errorf("Verify tx sign error!Hash:%s, error: verify sign fail", tx.Hash.String())
		return ErrSign
	}
	expectAddr := pk.GetAddress().GetHexString()
	if tx.Source != expectAddr {
		txLogger.Errorf("Verify tx sign error!Hash:%s,error:illegal signer! source:%s,expect source:%s", tx.Hash.String(), tx.Source, expectAddr)
		return ErrSign
	}
	return nil
}
