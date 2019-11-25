package service

import (
	"errors"
	"sort"
	"x/src/common"
	"x/src/middleware"
	"x/src/middleware/types"
	"x/src/middleware/db"

	"github.com/hashicorp/golang-lru"
	"github.com/vmihailenco/msgpack"
)

const (
	txDataBasePrefix = "tx"
	gameDataPrefix   = "g"
	rcvTxPoolSize    = 50000
	minerTxCacheSize = 1000
	missTxCacheSize  = 60000

	txCountPerBlock = 5000
)

var (
	ErrNil = errors.New("nil transaction")

	ErrHash = errors.New("invalid transaction hash")

	ErrExist = errors.New("transaction already exist in pool")

	ErrEvicted = errors.New("error transaction already exist in pool")
)

type ExecutedTransaction struct {
	Receipt     *types.Receipt
	Transaction *types.Transaction
}

type TransactionPool interface {
	PackForCast() []*types.Transaction

	//add new transaction to the transaction pool
	AddTransaction(tx *types.Transaction) (bool, error)

	//rcv transactions broadcast from other nodes
	AddBroadcastTransactions(txs []*types.Transaction)

	//add  local miss transactions while verifying blocks to the transaction pool
	AddMissTransactions(txs []*types.Transaction)

	GetTransaction(hash common.Hash) (*types.Transaction, error)

	GetTransactionStatus(hash common.Hash) (uint, error)

	GetExecuted(hash common.Hash) *ExecutedTransaction

	GetReceived() []*types.Transaction

	TxNum() int

	MarkExecuted(receipts types.Receipts, txs []*types.Transaction, evictedTxs []common.Hash)

	UnMarkExecuted(txs []*types.Transaction)

	AddExecuted(tx *types.Transaction) error

	Clear()

	IsExisted(hash common.Hash) bool

	IsGameData(hash common.Hash) bool

	PutGameData(hash common.Hash)
}

type TxPool struct {
	minerTxs   *lru.Cache // miner and bonus tx
	missTxs    *lru.Cache
	evictedTxs *lru.Cache

	received *simpleContainer

	executed db.Database
	gameData db.Database
	batch    db.Batch

	txCount uint64
	lock    middleware.Loglock

	nodeType byte
}

var txpoolInstance TransactionPool

func initTransactionPool(nodeType byte) {
	if nil == txpoolInstance {
		txpoolInstance = newTransactionPool(nodeType)
	}
}

func GetTransactionPool() TransactionPool {
	return txpoolInstance
}

func newTransactionPool(nodeType byte) TransactionPool {
	pool := &TxPool{
		lock:     middleware.NewLoglock("txPool"),
		nodeType: nodeType,
	}
	pool.received = newSimpleContainer(rcvTxPoolSize)
	pool.minerTxs, _ = lru.New(minerTxCacheSize)
	pool.missTxs, _ = lru.New(missTxCacheSize)
	pool.evictedTxs, _ = lru.New(minerTxCacheSize)

	executed, err := db.NewDatabase(txDataBasePrefix)
	if err != nil {
		logger.Errorf("Init transaction pool error! Error:%s", err.Error())
		return nil
	}
	pool.executed = executed
	pool.batch = pool.executed.NewBatch()

	gameData, err := db.NewDatabase(gameDataPrefix)
	if err != nil {
		logger.Errorf("Init transaction pool error! Error:%s", err.Error())
		return nil
	}
	pool.gameData = gameData

	return pool
}

func (pool *TxPool) AddTransaction(tx *types.Transaction) (bool, error) {
	if err := pool.verifyTransaction(tx); err != nil {
		logger.Infof("Tx verify error:%s. Hash:%s, tx type:%d", err.Error(), tx.Hash.String(), tx.Type)
		return false, err
	}

	pool.lock.Lock("AddTransaction")
	defer pool.lock.Unlock("AddTransaction")
	b, err := pool.add(tx)
	//	logger.Debugf("Add tx %s to pool result:%t", tx.Hash.String(), b)
	return b, err
}

// deprecated
func (pool *TxPool) AddBroadcastTransactions(txs []*types.Transaction) {
	if nil == txs || 0 == len(txs) {
		return
	}
	pool.lock.Lock("AddBroadcastTransactions")
	defer pool.lock.Unlock("AddBroadcastTransactions")

	for _, tx := range txs {
		if err := pool.verifyTransaction(tx); err != nil {
			logger.Infof("Tx verify error:%s. Hash:%s, tx type:%d", err.Error(), tx.Hash.String(), tx.Type)
			continue
		}
		pool.add(tx)
	}
}

func (pool *TxPool) AddMissTransactions(txs []*types.Transaction) {
	if nil == txs || 0 == len(txs) {
		return
	}
	for _, tx := range txs {
		pool.missTxs.Add(tx.Hash, tx)
	}
	return
}

func (pool *TxPool) AddExecuted(tx *types.Transaction) error {
	if nil == tx {
		return nil
	}

	executedTx := &ExecutedTransaction{
		Transaction: tx,
	}
	executedTxBytes, err := msgpack.Marshal(executedTx)
	if nil != err {
		return err
	}

	return pool.executed.Put(tx.Hash.Bytes(), executedTxBytes)

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
			Receipt:     receipt,
			Transaction: findTxInList(txs, hash, i),
		}
		executedTxBytes, err := msgpack.Marshal(executedTx)
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

func (pool *TxPool) UnMarkExecuted(txs []*types.Transaction) {
	if nil == txs || 0 == len(txs) {
		return
	}
	pool.lock.RLock("UnMarkExecuted")
	defer pool.lock.RUnlock("UnMarkExecuted")

	for _, tx := range txs {
		pool.executed.Delete(tx.Hash.Bytes())
		pool.add(tx)
	}
}
func (pool *TxPool) IsExisted(hash common.Hash) bool {
	return pool.isTransactionExisted(hash)
}

func (pool *TxPool) GetTransaction(hash common.Hash) (*types.Transaction, error) {
	missTx, existInMissTxs := pool.missTxs.Get(hash)
	if existInMissTxs {
		return missTx.(*types.Transaction), nil
	}

	minerTx, existInMinerTxs := pool.minerTxs.Get(hash)
	if existInMinerTxs {
		return minerTx.(*types.Transaction), nil
	}

	receivedTx := pool.received.get(hash)
	if nil != receivedTx {
		return receivedTx, nil
	}

	executedTx := pool.GetExecuted(hash)
	if nil != executedTx {
		return executedTx.Transaction, nil
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
	pool.minerTxs, _ = lru.New(minerTxCacheSize)
	pool.missTxs, _ = lru.New(missTxCacheSize)
}

func (pool *TxPool) IsGameData(hash common.Hash) bool {
	result, _ := pool.gameData.Has(hash.Bytes())

	return result
}

func (pool *TxPool) PutGameData(hash common.Hash) {
	value := []byte{0}
	pool.gameData.Put(hash.Bytes(), value)
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
	err := msgpack.Unmarshal(txBytes, &executedTx)
	if err != nil || executedTx == nil {
		return nil
	}
	return executedTx
}

func (p *TxPool) TxNum() int {
	return p.received.Len()
}

func (pool *TxPool) PackForCast() []*types.Transaction {
	minerTxs := pool.packMinerTx()
	if len(minerTxs) >= txCountPerBlock {
		return minerTxs
	}
	result := pool.packTx(minerTxs)

	// transactions 已经根据RequestId排序
	return result
}

func (pool *TxPool) verifyTransaction(tx *types.Transaction) error {
	if pool.evictedTxs.Contains(tx.Hash) {
		return ErrEvicted
	}

	//expectHash := tx.GenHash()
	//if tx.Hash != expectHash {
	//	logger.Infof("Illegal tx hash! Hash:%s,except hash:%s", tx.Hash.String(), expectHash.String())
	//	return ErrHash
	//}
	//err := pool.verifySign(tx)
	//if err != nil {
	//	return err
	//}
	return nil
}

func (pool *TxPool) verifySign(tx *types.Transaction) error {
	//coiner 发送过来的充值消息不需要验证签名，因为在收到消息的时候验证过了
	if tx.Type == types.TransactionTypeCoinDepositAck || tx.Type == types.TransactionTypeFTDepositAck || tx.Type == types.TransactionTypeNFTDepositAck {
		return nil
	}
	//其他交易签名校验
	//hashByte := tx.Hash.Bytes()
	//pk, err := tx.Sign.RecoverPubkey(hashByte)
	//if err != nil {
	//	return err
	//}
	//if !pk.Verify(hashByte, tx.Sign) {
	//	return fmt.Errorf("verify sign fail, hash=%v", tx.Hash.Hex())
	//}
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

	if tx.Type == types.TransactionTypeMinerApply || tx.Type == types.TransactionTypeMinerAbort ||
		tx.Type == types.TransactionTypeBonus || tx.Type == types.TransactionTypeMinerRefund {

		if tx.Type == types.TransactionTypeMinerApply {
			logger.Debugf("Add TransactionTypeMinerApply,hash:%s,", tx.Hash.String())
		}
		pool.minerTxs.Add(tx.Hash, tx)
	} else {
		pool.received.push(tx, pool.nodeType)
	}

	return true, nil
}

func (pool *TxPool) remove(txHash common.Hash) {
	pool.minerTxs.Remove(txHash)
	pool.missTxs.Remove(txHash)
	pool.received.remove(txHash)
}

func (pool *TxPool) isTransactionExisted(hash common.Hash) bool {
	existInMinerTxs := pool.minerTxs.Contains(hash)
	if existInMinerTxs {
		return true
	}

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

func (pool *TxPool) packMinerTx() []*types.Transaction {
	minerTxs := make([]*types.Transaction, 0, txCountPerBlock)
	minerTxHashes := pool.minerTxs.Keys()
	for _, minerTxHash := range minerTxHashes {
		if v, ok := pool.minerTxs.Get(minerTxHash); ok {
			minerTxs = append(minerTxs, v.(*types.Transaction))
			if v.(*types.Transaction).Type == types.TransactionTypeMinerApply {
				logger.Debugf("pack miner apply tx hash:%s,", v.(*types.Transaction).Hash.String())
			}
		}
		if len(minerTxs) >= txCountPerBlock {
			return minerTxs
		}
	}
	return minerTxs
}

func (pool *TxPool) packTx(packedTxs []*types.Transaction) []*types.Transaction {
	txs := pool.received.asSlice()
	sort.Sort(types.Transactions(txs))
	for _, tx := range txs {
		packedTxs = append(packedTxs, tx)
		if len(packedTxs) >= txCountPerBlock {
			return packedTxs
		}
	}
	return packedTxs
}