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

package service

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/eth_tx"
	"com.tuntun.rangers/node/src/middleware"
	"com.tuntun.rangers/node/src/middleware/db"
	"com.tuntun.rangers/node/src/middleware/mysql"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/storage/account"
	"com.tuntun.rangers/node/src/storage/rlp"
	"com.tuntun.rangers/node/src/utility"
	"encoding/json"
	"errors"
	"fmt"
	lru "github.com/hashicorp/golang-lru"
	"sort"
)

const (
	txDataBasePrefix = "tx"
	rcvTxPoolSize    = 50000
	txCacheSize      = 1000

	txCountPerBlock = 200
)

var (
	ErrNil = errors.New("nil transaction")

	ErrChainId = errors.New("illegal chain id")

	ErrHash = errors.New("illegal transaction hash")

	ErrSign = errors.New("illegal transaction signature")

	ErrExist = errors.New("transaction already exist in pool")

	ErrEvicted = errors.New("error transaction already exist in pool")

	ErrIllegal = errors.New("illegal transaction")
)

type ExecutedReceipt struct {
	types.Receipt
	BlockHash common.Hash `json:"blockHash"`
}

type ExecutedTransaction struct {
	Receipt     ExecutedReceipt
	Transaction []byte
}

type TransactionPool interface {
	PackForCast(height uint64, stateDB *account.AccountDB) []*types.Transaction

	//add new transaction to the transaction pool
	AddTransaction(tx *types.Transaction) (bool, error)

	GetTransaction(hash common.Hash) (*types.Transaction, error)

	GetTransactionStatus(hash common.Hash) (uint, error)

	GetExecuted(hash common.Hash) *ExecutedTransaction

	GetReceived() []*types.Transaction

	TxNum() int

	IsFull() bool

	MarkExecuted(header *types.BlockHeader, receipts types.Receipts, txs []*types.Transaction, evictedTxs []common.Hash)

	UnMarkExecuted(block *types.Block)

	Clear()

	IsExisted(hash common.Hash) bool

	VerifyTransaction(tx *types.Transaction, height uint64) error

	ProcessFee(tx types.Transaction, accountDB *account.AccountDB) error

	GetGateNonce() uint64

	Close()
}

type TxPool struct {
	evictedTxs *lru.Cache
	received   *simpleContainer

	executed db.Database
	batch    db.Batch
}

var (
	txpoolInstance TransactionPool
	delta, _       = utility.StrToBigInt("0.0001")
	delta026, _    = utility.StrToBigInt("0.001")
)

func initTransactionPool() {
	if nil == txpoolInstance {
		txpoolInstance = newTransactionPool()
	}
}

func GetTransactionPool() TransactionPool {
	return txpoolInstance
}

func newTransactionPool() TransactionPool {
	pool := &TxPool{}
	pool.received = newSimpleContainer(rcvTxPoolSize)
	pool.evictedTxs, _ = lru.New(txCacheSize)

	executed, err := db.NewLDBDatabase(txDataBasePrefix, 128, 128)
	if err != nil {
		txPoolLogger.Errorf("Init transaction pool error! Error:%s", err.Error())
		return nil
	}
	pool.executed = executed
	pool.batch = pool.executed.NewBatch()

	return pool
}

func (pool *TxPool) Close() {
	if nil != pool.executed {
		pool.executed.Close()
	}

	if nil != pool.received {
		pool.received.Close()
	}

}

func (pool *TxPool) refreshGateNonce(tx *types.Transaction) {
	sub := tx.SubTransactions
	if 1 == len(sub) && 0 != sub[0].Address {
		txPoolLogger.Debugf("refreshGateNonce. txhash: %s, gateNonce: %d", tx.Hash.String(), sub[0].Address)
		pool.batch.Put(utility.StrToBytes(txDataBasePrefix), utility.UInt64ToByte(sub[0].Address))
	}
}

func (pool *TxPool) GetGateNonce() uint64 {
	data, err := pool.executed.Get(utility.StrToBytes(txDataBasePrefix))
	if err != nil || 0 == len(data) {
		return 0
	}

	return utility.ByteToUInt64(data)
}

func (pool *TxPool) AddTransaction(tx *types.Transaction) (bool, error) {
	//if pool.evictedTxs.Contains(tx.Hash) {
	//	txPoolLogger.Infof("Tx is marked evicted tx,do not add pool. Hash:%s", tx.Hash.String())
	//	return false, ErrEvicted
	//}

	b, err := pool.add(tx)
	if nil == err {
		pool.refreshGateNonce(tx)
	}
	return b, err
}

func (pool *TxPool) MarkExecuted(header *types.BlockHeader, receipts types.Receipts, txs []*types.Transaction, evictedTxs []common.Hash) {
	txHashList := make([]interface{}, 0)

	if receipts != nil && len(receipts) != 0 {
		go mysql.InsertLogs(header.Height, receipts, header.Hash)

		for i, receipt := range receipts {
			hash := receipt.TxHash
			txHashList = append(txHashList, hash)
			var er ExecutedReceipt
			er.BlockHash = header.Hash
			er.Height = receipt.Height
			er.TxHash = receipt.TxHash
			er.Status = receipt.Status
			er.Logs = receipt.Logs
			er.ContractAddress = receipt.ContractAddress
			er.GasUsed = receipt.GasUsed
			if 0 != len(receipt.Result) {
				er.Result = receipt.Result
			}
			executedTx := &ExecutedTransaction{
				Receipt: er,
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
			pool.refreshGateNonce(tx)
		}
		if pool.batch.ValueSize() > 0 {
			pool.batch.Write()
			pool.batch.Reset()
		}
	}

	if evictedTxs != nil {
		for _, hash := range evictedTxs {
			pool.evictedTxs.Add(hash, 0)
			txHashList = append(txHashList, hash)
		}
	}

	if len(txHashList) > 0 {
		pool.remove(txHashList)
	}
}

func (pool *TxPool) UnMarkExecuted(block *types.Block) {
	txs := block.Transactions
	evictedTxs := block.Header.EvictedTxs
	if nil == txs || 0 == len(txs) {
		return
	}

	mysql.DeleteLogs(block.Header.Height, block.Header.Hash)

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
	middleware.LockBlockchain("Clear")
	defer middleware.UnLockBlockchain("Clear")

	executed, _ := db.NewDatabase(txDataBasePrefix)
	pool.executed = executed
	pool.batch.Reset()

	pool.received = newSimpleContainer(rcvTxPoolSize)
}

func (pool *TxPool) GetReceived() []*types.Transaction {
	items := pool.received.asSlice()
	result := make([]*types.Transaction, len(items))

	for i, item := range items {
		result[i] = item.(*types.Transaction)
	}
	return result
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
	//complete log's txHash and blockHash filed
	if len(executedTx.Receipt.Logs) > 0 {
		for _, log := range executedTx.Receipt.Logs {
			log.BlockHash = executedTx.Receipt.BlockHash
			log.TxHash = executedTx.Receipt.TxHash
		}
	}
	return executedTx
}

func (p *TxPool) TxNum() int {
	return p.received.Len()
}

func (p *TxPool) IsFull() bool {
	return p.received.isFull()
}

func (pool *TxPool) PackForCast(height uint64, stateDB *account.AccountDB) []*types.Transaction {
	packedTxs := make([]*types.Transaction, 0)

	txs := pool.received.asSlice()
	if 0 == len(txs) {
		txPoolLogger.Debugf("packed no tx. height: %d", height)
		return packedTxs
	}

	for _, tx := range txs {
		packedTxs = append(packedTxs, tx.(*types.Transaction))
	}
	if common.IsProposal018() {
		packedTxs = pool.checkNonce(packedTxs, stateDB)
	}
	if len(packedTxs) > txCountPerBlock {
		packedTxs = packedTxs[:txCountPerBlock]
	}
	if 0 == len(packedTxs) {
		txPoolLogger.Debugf("after check nonce ,packed no tx. height: %d", height)
	} else {
		txPoolLogger.Debugf("packed tx. height: %d. nonce from %d to %d. size: %d", height, packedTxs[0].RequestId, packedTxs[len(packedTxs)-1].RequestId, len(packedTxs))
	}
	return packedTxs
}

func (pool *TxPool) VerifyTransaction(tx *types.Transaction, height uint64) error {
	if tx.Type == types.TransactionTypeETHTX {
		return verifyETHTx(tx, height)
	}
	err := verifyTxChainId(tx, height)
	if nil != err {
		return err
	}
	err = verifyTransactionHash(tx)
	if nil != err {
		return err
	}
	err = verifyTransactionSign(tx)
	if nil != err {
		return err
	}

	txPoolLogger.Debugf("Verify tx success. hash: %s", tx.Hash.String())
	return nil
}

func (pool *TxPool) ProcessFee(tx types.Transaction, accountDB *account.AccountDB) error {
	addr := common.HexStringToAddress(tx.Source)
	balance := accountDB.GetBalance(addr)

	fee := delta
	if common.IsProposal026() {
		fee = delta026
	}

	if balance.Cmp(fee) < 0 {
		msg := fmt.Sprintf("not enough max, addr: %s, balance: %s", tx.Source, balance)
		return fmt.Errorf(msg)
	}
	accountDB.SubBalance(addr, fee)
	accountDB.AddBalance(common.FeeAccount, fee)
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
	txPoolLogger.Debugf("[pool]Add tx:%s. global nonce: %d,source:%s,nonce:%d, After add,received size:%d", tx.Hash.String(), tx.RequestId, tx.Source, tx.Nonce, pool.received.Len())
	return true, nil
}

func (pool *TxPool) remove(txHashList []interface{}) {
	pool.received.remove(txHashList)
	for _, txHash := range txHashList {
		txPoolLogger.Debugf("[pool]removed tx:%s", txHash.(common.Hash).String())
	}
	txPoolLogger.Debugf("[pool]removed tx, %d. After remove,received size:%d", len(txHashList), pool.received.Len())

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

	if receiptIndex < len(txs) && txs[receiptIndex].Hash == txHash {
		return txs[receiptIndex]
	}

	for _, tx := range txs {
		if tx.Hash == txHash {
			return tx
		}
	}
	return nil
}

func verifyTxChainId(tx *types.Transaction, height uint64) error {
	expectedChainId := common.ChainId(height)
	if tx.ChainId != expectedChainId {
		txPoolLogger.Errorf("Verify chain id error!Hash:%s,chainId:%s,expect chainId:%s", tx.Hash.String(), tx.ChainId, expectedChainId)
		return ErrChainId
	}
	return nil
}

func verifyTransactionHash(tx *types.Transaction) error {
	expectHash := tx.GenHash()
	if tx.Hash != expectHash {
		txPoolLogger.Errorf("Verify tx hash error!Hash:%s,expect hash:%s", tx.Hash.String(), expectHash.String())
		return ErrHash
	}
	return nil
}

func verifyTransactionSign(tx *types.Transaction) error {
	if tx.Sign == nil {
		txPoolLogger.Errorf("Verify tx sign error!Hash:%s,error:nil sign!", tx.Hash.String())
		return ErrSign
	}

	hashByte := tx.Hash.Bytes()
	pk, err := tx.Sign.RecoverPubkey(hashByte)
	if err != nil {
		txPoolLogger.Errorf("Verify tx sign error!Hash:%s,error:%s", tx.Hash.String(), err.Error())
		return ErrSign
	}
	if !pk.Verify(hashByte, tx.Sign) {
		txPoolLogger.Errorf("Verify tx sign error!Hash:%s, error: verify sign fail", tx.Hash.String())
		return ErrSign
	}
	expectAddr := pk.GetAddress().GetHexString()
	if tx.Source != expectAddr {
		txPoolLogger.Errorf("Verify tx sign error!Hash:%s,error:illegal signer! source:%s,expect source:%s", tx.Hash.String(), tx.Source, expectAddr)
		return ErrSign
	}
	return nil
}

func verifyETHTx(tx *types.Transaction, height uint64) error {
	if tx == nil {
		return ErrNil
	}
	ethTx := new(eth_tx.Transaction)
	var encodedTx utility.Bytes
	encodedTx = common.FromHex(tx.ExtraData)
	if err := rlp.DecodeBytes(encodedTx, ethTx); err != nil {
		txPoolLogger.Errorf("Verify eth tx rlp error!error:%v", err)
		return ErrIllegal
	}

	signer := eth_tx.NewEIP155Signer(common.GetChainId(height))
	sender, err := eth_tx.Sender(signer, ethTx)
	if err != nil {
		txPoolLogger.Errorf("Verify eth tx error!tx:%s,error:%v", ethTx.Hash().String(), err)
		return ErrIllegal
	}

	expectedTx := eth_tx.ConvertTx(ethTx, sender, encodedTx)
	if !compareTx(tx, expectedTx) {
		txPoolLogger.Errorf("Verify eth tx error:tx diff!tx:%s,expected tx:%s", tx.ToTxJson().ToString(), expectedTx.ToTxJson().ToString())
		return ErrIllegal
	}
	txPoolLogger.Debugf("Verify eth tx success. hash: %s", tx.Hash.String())
	return nil
}

func compareTx(tx *types.Transaction, expectedTx *types.Transaction) bool {
	if tx == nil || expectedTx == nil {
		return false
	}
	if tx.Source != expectedTx.Source || tx.Target != expectedTx.Target || tx.Type != expectedTx.Type || tx.ExtraData != expectedTx.ExtraData {
		return false
	}
	if tx.Nonce != expectedTx.Nonce || tx.ChainId != expectedTx.ChainId || tx.Data != expectedTx.Data || tx.Hash != expectedTx.Hash {
		return false
	}
	return true
}

func (pool *TxPool) checkNonce(txList []*types.Transaction, stateDB *account.AccountDB) []*types.Transaction {
	txs := types.Transactions(txList)
	sort.Sort(txs)

	packedTxs := make([]*types.Transaction, 0)
	nonceMap := make(map[string]uint64, 0)
	for _, tx := range txs {
		if tx.RequestId == 0 { //only json rpc tx pre check nonce
			expectedNonce, exist := nonceMap[tx.Source]
			if !exist {
				expectedNonce = stateDB.GetNonce(common.HexToAddress(tx.Source))
				nonceMap[tx.Source] = expectedNonce
			}

			//nonce too low tx and repeat nonce tx will be packed into block and execute failed
			if expectedNonce < tx.Nonce {
				txPoolLogger.Debugf("nonce too high tx,skip pack.tx:%s,expected:%d,but:%d", tx.Hash.String(), expectedNonce, tx.Nonce)
				continue
			}
			if expectedNonce == tx.Nonce {
				nonceMap[tx.Source] = expectedNonce + 1
			}
		}

		packedTxs = append(packedTxs, tx)
		if len(packedTxs) >= txCountPerBlock {
			break
		}
	}
	return packedTxs
}
