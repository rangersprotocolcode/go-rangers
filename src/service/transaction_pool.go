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
	"sync"
	"time"
)

const (
	txDataBasePrefix = "tx"
	rcvTxPoolSize    = 50000
	txCacheSize      = 1000

	txCountPerBlock        = 200
	singleAccountQueueSize = 64
	expiredRing            = 2
	txCycleInterval        = time.Minute * 1
)

var (
	ErrNil = errors.New("nil transaction")

	ErrChainId = errors.New("illegal chain id")

	ErrHash = errors.New("illegal transaction hash")

	ErrSign = errors.New("illegal transaction signature")

	ErrExist = errors.New("transaction already exist in pool")

	ErrEvicted = errors.New("error transaction already exist in pool")

	ErrIllegal = errors.New("illegal transaction")

	ErrNonceTooLow = errors.New("nonce too low")

	ErrReplaceSameNonceFailed = errors.New("replace same nonce tx failed")

	ErrSameNonceExistPending = errors.New("same nonce pending tx exist")

	ErrAccountQueueTxLimitExceeded = errors.New("queue tx limit exceeded")

	ErrTxPoolOverflow = errors.New("txpool is full")
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

	GetPendingNonce(address string) uint64

	Stats() (int, int)

	GetPendingList(address string) []uint64

	GetQueueList(address string) []uint64
}

type TxPool struct {
	evictedTxs *lru.Cache
	received   *simpleContainer

	executed db.Database
	batch    db.Batch

	pending         map[string]*txList
	queue           map[string]*txList
	lock            middleware.Loglock
	annualRingMap   sync.Map //use to store queue tx for expire calculate
	lifeCycleTicker *time.Ticker
}

var (
	txpoolInstance TransactionPool
	delta, _       = utility.StrToBigInt("0.0001")
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

	pool.pending = make(map[string]*txList, 0)
	pool.queue = make(map[string]*txList, 0)
	pool.lock = middleware.NewLoglock("txPool")

	pool.annualRingMap = sync.Map{}
	pool.lifeCycleTicker = time.NewTicker(txCycleInterval)
	go pool.loop()
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

func (pool *TxPool) GetPendingNonce(address string) uint64 {
	pool.lock.RLock("GetPendingNonce")
	pendingList := pool.pending[address]

	if pendingList != nil && len(*pendingList.index) > 0 {
		pendingNonce := pendingList.GetTailNonce() + 1
		pool.lock.RUnlock("GetPendingNonce")
		return pendingNonce
	}
	pool.lock.RUnlock("GetPendingNonce")

	state := middleware.AccountDBManagerInstance.GetLatestStateDB()
	return state.GetNonce(common.HexToAddress(address))
}

func (pool *TxPool) AddTransaction(tx *types.Transaction) (bool, error) {
	//if pool.evictedTxs.Contains(tx.Hash) {
	//	txPoolLogger.Infof("Tx is marked evicted tx,do not add pool. Hash:%s", tx.Hash.String())
	//	return false, ErrEvicted
	//}

	if tx == nil {
		return false, ErrNil
	}
	if pool.IsFull() {
		return false, ErrTxPoolOverflow
	}
	b, err := pool.add(tx, tx.Type == types.TransactionTypeETHTX)
	if nil == err {
		pool.refreshGateNonce(tx)
	}
	return b, err
}

func (pool *TxPool) MarkExecuted(header *types.BlockHeader, receipts types.Receipts, txs []*types.Transaction, evictedTxs []common.Hash) {
	if receipts != nil && len(receipts) != 0 {
		go mysql.InsertLogs(header.Height, receipts, header.Hash)

		for i, receipt := range receipts {
			hash := receipt.TxHash
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

	pool.remove(txs, evictedTxs)
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
		pool.add(tx, false)
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
		txPoolLogger.Debugf("packed tx. height: %d. global nonce from %d to %d. size: %d", height, packedTxs[0].RequestId, packedTxs[len(packedTxs)-1].RequestId, len(packedTxs))
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

	if balance.Cmp(delta) < 0 {
		msg := fmt.Sprintf("not enough max, addr: %s, balance: %s", tx.Source, balance)
		return fmt.Errorf(msg)
	}
	accountDB.SubBalance(addr, delta)
	accountDB.AddBalance(common.FeeAccount, delta)
	return nil
}
func (pool *TxPool) GetPendingList(address string) []uint64 {
	pendingList := pool.pending[address]
	if pendingList == nil {
		return nil
	}
	return *pendingList.index
}

func (pool *TxPool) GetQueueList(address string) []uint64 {
	queueList := pool.queue[address]
	if queueList == nil {
		return nil
	}
	return *queueList.index
}

// stats retrieves the current pool stats, namely the number of pending and the
// number of queued (non-executable) transactions.
func (pool *TxPool) Stats() (int, int) {
	pending := 0
	for _, list := range pool.pending {
		pending += list.Size()
	}
	queued := 0
	for _, list := range pool.queue {
		queued += list.Size()
	}
	return pending, queued
}

func (pool *TxPool) add(tx *types.Transaction, checkPending bool) (bool, error) {
	if tx == nil {
		return false, ErrNil
	}

	if pool.isTransactionExisted(tx.Hash) {
		return false, ErrExist
	}
	if !checkPending {
		pool.received.push(tx)
		txPoolLogger.Debugf("[pool]Add tx without check pending:%s. global nonce:%d,source:%s,nonce:%d.Received size:%d", tx.Hash.String(), tx.RequestId, tx.Source, tx.Nonce, pool.received.Len())
		return true, nil
	}

	pendingNonce := pool.GetPendingNonce(tx.Source)
	pool.lock.Lock("addTx")
	defer pool.lock.Unlock("addTx")
	if tx.Nonce < pendingNonce {
		pendingList := pool.pending[tx.Source]
		if pendingList != nil && pendingList.Get(tx.Nonce) != nil {
			return false, ErrSameNonceExistPending
		}
		return false, ErrNonceTooLow
	} else if tx.Nonce == pendingNonce {
		if pool.pending[tx.Source] == nil {
			pool.pending[tx.Source] = newTxList()
		}
		pool.pending[tx.Source].Put(tx)
		pool.tryPopQueue(tx.Source)
	} else {
		err := pool.addQueue(tx)
		if err != nil {
			return false, err
		}
	}
	pool.received.push(tx)
	txPoolLogger.Debugf("[pool]Add tx:%s. global nonce:%d,source:%s,nonce:%d.Received size:%d", tx.Hash.String(), tx.RequestId, tx.Source, tx.Nonce, pool.received.Len())
	return true, nil
}

func (pool *TxPool) addQueue(tx *types.Transaction) error {
	if pool.queue[tx.Source] == nil {
		pool.queue[tx.Source] = newTxList()
	}
	if pool.queue[tx.Source].Size() >= singleAccountQueueSize {
		return ErrAccountQueueTxLimitExceeded
	}
	sameNonceTx := pool.queue[tx.Source].Get(tx.Nonce)
	if sameNonceTx == nil {
		pool.queue[tx.Source].Put(tx)
		pool.annualRingMap.Store(tx.Source, uint64(0))
		return nil
	}
	if tx.Less(sameNonceTx) {
		return ErrReplaceSameNonceFailed
	}
	pool.queue[tx.Source].Put(tx)
	pool.received.remove(sameNonceTx.Hash)
	txPoolLogger.Debugf("[pool]removed replaced queue tx:%s,source:%s,nonce:%d", sameNonceTx.Hash.String(), sameNonceTx.Source, sameNonceTx.Nonce)
	pool.annualRingMap.Store(tx.Source, uint64(0))
	return nil
}

func (pool *TxPool) tryPopQueue(sourceAddress string) {
	queue := pool.queue[sourceAddress]
	pending := pool.pending[sourceAddress]
	if queue == nil || pending == nil {
		return
	}
	for queue.Size() > 0 {
		pendingTail := pending.GetTailNonce()
		queueHead := queue.GetHeadNonce()
		if queueHead != pendingTail+1 {
			break
		}
		tx := queue.Pop()
		pending.Put(tx)
	}
}

func (pool *TxPool) remove(txs []*types.Transaction, evictedTxs []common.Hash) {
	txHashList := make([]interface{}, 0)
	pool.lock.Lock("removeTx")
	if txs != nil {
		for _, tx := range txs {
			if pool.pending[tx.Source] != nil {
				pool.pending[tx.Source].Forward(tx.Nonce)
			}
			if pool.queue[tx.Source] != nil {
				pool.queue[tx.Source].Forward(tx.Nonce)
			}
			txHashList = append(txHashList, tx.Hash)
			txPoolLogger.Debugf("[pool]removed tx:%s", tx.Hash.String())
		}
	}

	if evictedTxs != nil {
		for _, hash := range evictedTxs {
			tx := pool.received.get(hash)
			if tx == nil {
				continue
			}
			if pool.pending[tx.Source] != nil {
				pool.pending[tx.Source].Forward(tx.Nonce)
			}
			if pool.queue[tx.Source] != nil {
				pool.queue[tx.Source].Forward(tx.Nonce)
			}
			txHashList = append(txHashList, tx.Hash)
			txPoolLogger.Debugf("[pool]removed tx:%s", tx.Hash.String())
		}
	}
	pool.lock.Unlock("removeTx")
	pool.received.batchRemove(txHashList)
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
		if tx.RequestId > 0 {
			continue //only json rpc tx pre check nonce
		}
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
		packedTxs = append(packedTxs, tx)
		if len(packedTxs) >= txCountPerBlock {
			break
		}
	}
	return packedTxs
}

func (pool *TxPool) loop() {
	for {
		select {
		case <-pool.lifeCycleTicker.C:
			go pool.growRing()
		}
	}
}

func (pool *TxPool) growRing() {
	pool.annualRingMap.Range(func(key, value interface{}) bool {
		address := key.(string)
		txRing := value.(uint64)
		txRing = txRing + 1
		pool.annualRingMap.Store(address, txRing)
		if txRing >= expiredRing {
			pool.lock.Lock("queue expire")
			queueList := pool.queue[address]
			if queueList != nil && queueList.Size() > 0 {
				txPoolLogger.Debugf("address %s expired,clear the queue list.", address)
				for queueList.Size() > 0 {
					tx := queueList.Pop()
					txPoolLogger.Debugf("queue tx expired,drop it. hash:%s", tx.Hash.String())
					pool.received.remove(tx.Hash)
				}
			}
			pool.lock.Unlock("queue expire")
		}
		return true
	})
}
