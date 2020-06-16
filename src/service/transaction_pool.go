package service

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/db"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"errors"
	"sort"

	"github.com/hashicorp/golang-lru"

	"encoding/json"
	"fmt"
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
	Transaction []byte
}

type TransactionPool interface {
	PackForCast() []*types.Transaction

	//add new transaction to the transaction pool
	AddTransaction(tx *types.Transaction) (bool, error)

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

	VerifyTransaction(tx *types.Transaction) error

	ProcessFee(tx types.Transaction, accountDB *account.AccountDB) error
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
	pool.minerTxs, _ = lru.New(minerTxCacheSize)
	pool.missTxs, _ = lru.New(missTxCacheSize)
	pool.evictedTxs, _ = lru.New(minerTxCacheSize)

	executed, err := db.NewDatabase(txDataBasePrefix)
	if err != nil {
		txPoolLogger.Errorf("Init transaction pool error! Error:%s", err.Error())
		return nil
	}
	pool.executed = executed
	pool.batch = pool.executed.NewBatch()

	gameData, err := db.NewDatabase(gameDataPrefix)
	if err != nil {
		txPoolLogger.Errorf("Init transaction pool error! Error:%s", err.Error())
		return nil
	}
	pool.gameData = gameData

	return pool
}

func (pool *TxPool) AddTransaction(tx *types.Transaction) (bool, error) {
	if err := pool.verifyTransaction(tx); err != nil {
		txPoolLogger.Infof("Tx verify error:%s. Hash:%s, tx type:%d", err.Error(), tx.Hash.String(), tx.Type)
		return false, err
	}

	pool.lock.Lock("AddTransaction")
	defer pool.lock.Unlock("AddTransaction")
	b, err := pool.add(tx)
	return b, err
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

	executedTx := &ExecutedTransaction{}

	txData, _ := types.MarshalTransaction(tx)
	executedTx.Transaction = txData
	executedTxBytes, err := json.Marshal(executedTx)
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
	return nil
}

func (pool *TxPool) VerifyTransaction(tx *types.Transaction) error {
	err := pool.verifyTransactionHash(tx)
	if nil != err {
		return err
	}

	err = pool.verifyTransactionSign(tx)
	if nil != err {
		return err
	}

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

func (pool *TxPool) verifyTransactionHash(tx *types.Transaction) error {
	expectHash := tx.GenHash()
	if tx.Hash != expectHash {
		err := fmt.Errorf("illegal tx hash! Hash:%s,expect hash:%s", tx.Hash.String(), expectHash.String())
		txLogger.Errorf("Verify tx hash error!Hash:%s,error:%s", tx.Hash.String(), err.Error())
		return err
	}

	return nil
}

func (pool *TxPool) verifyTransactionSign(tx *types.Transaction) error {
	if tx.Sign == nil {
		return fmt.Errorf("nil sign")
	}

	hashByte := tx.Hash.Bytes()
	pk, err := tx.Sign.RecoverPubkey(hashByte)
	if err != nil {
		txLogger.Errorf("Verify tx sign error!Hash:%s,error:%s", tx.Hash.String(), err.Error())
		return err
	}
	if !pk.Verify(hashByte, tx.Sign) {
		txLogger.Errorf("Verify tx sign error!Hash:%s, error: verify sign fail", tx.Hash.String())
		return fmt.Errorf("verify sign fail")
	}
	expectAddr := pk.GetAddress().GetHexString()
	if tx.Source != expectAddr {
		err := fmt.Errorf("illegal signer! Source:%s,expect source:%s", tx.Source, expectAddr)
		txLogger.Errorf("Verify tx sign error!Hash:%s,error:%s", tx.Hash.String(), err.Error())
		return err
	}
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
		tx.Type == types.TransactionTypeBonus || tx.Type == types.TransactionTypeMinerRefund ||
		tx.Type == types.TransactionTypeMinerAdd {

		if tx.Type == types.TransactionTypeMinerApply {
			txPoolLogger.Debugf("Add TransactionTypeMinerApply,hash:%s,", tx.Hash.String())
		}
		pool.minerTxs.Add(tx.Hash, tx)
	} else {
		pool.received.push(tx)
	}
	txPoolLogger.Debugf("Add tx:%s.After add,received  size:%d,miner txs size:%d", tx.Hash.String(), pool.received.Len(), pool.minerTxs.Len())
	return true, nil
}

func (pool *TxPool) remove(txHash common.Hash) {
	pool.minerTxs.Remove(txHash)
	pool.missTxs.Remove(txHash)
	pool.received.remove(txHash)
	txPoolLogger.Debugf("Remove tx:%s.After remove,received size:%d,miner txs size:%d", txHash.String(), pool.received.Len(), pool.minerTxs.Len())
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
				txPoolLogger.Debugf("Pack miner apply tx hash:%s,", v.(*types.Transaction).Hash.String())
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
		txPoolLogger.Debugf("Pack tx:%s", tx.Hash.String())
		if len(packedTxs) >= txCountPerBlock {
			return packedTxs
		}
	}
	return packedTxs
}
