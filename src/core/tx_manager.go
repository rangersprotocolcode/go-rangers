package core

import (
	"x/src/storage/account"
	"x/src/middleware/types"
	"sync"
	"fmt"
	"x/src/common"
	"x/src/middleware"
)

// 基于gameId的事务管理器
type TxManager struct {
	context     map[string]*TxContext
	contextLock map[common.Hash]*middleware.Loglock
	lock        *sync.Mutex
}

type TxContext struct {
	AccountDB *account.AccountDB
	Tx        *types.Transaction
	snapshot  int
}

var TxManagerInstance *TxManager

func initTxManager() {
	TxManagerInstance = &TxManager{}

	TxManagerInstance.context = make(map[string]*TxContext)
	TxManagerInstance.contextLock = make(map[common.Hash]*middleware.Loglock)
	TxManagerInstance.lock = &sync.Mutex{}
}

func (manager *TxManager) BeginTransaction(gameId string, accountDB *account.AccountDB, isCopy bool, tx *types.Transaction) error {
	if nil == accountDB || nil == tx || 0 == len(gameId) {
		return nil
	}

	// 对tx.Hash加锁
	// 同个时间内，只允许一个相同的交易被执行
	txLock := manager.getTxLock(tx.Hash)
	txLock.Lock(fmt.Sprintf("txM: %s", tx.Hash.String()))

	// 已经执行过了
	if GetBlockChain().GetTransactionPool().IsExisted(tx.Hash) {
		manager.unlock(tx.Hash)
		return fmt.Errorf("isExisted")
	}

	tx.SubTransactions = make([]string, 0)
	copy := accountDB
	if isCopy {
		copy = accountDB.Copy()
	}

	snapshot := copy.Snapshot()
	manager.context[gameId] = &TxContext{AccountDB: copy, Tx: tx, snapshot: snapshot}
	return nil
}

func (manager *TxManager) GetContext(gameId string) *TxContext {
	return manager.context[gameId]
}

func (manager *TxManager) Commit(gameId string, hash common.Hash) {
	manager.remove(gameId)
	manager.unlock(hash)
}

func (manager *TxManager) RollBack(gameId string, hash common.Hash) {
	defer manager.unlock(hash)

	context := manager.GetContext(gameId)
	if nil == context {
		return
	}

	context.AccountDB.RevertToSnapshot(context.snapshot)
	manager.remove(gameId)
}

func (manager *TxManager) remove(gameId string) {
	delete(manager.context, gameId)
}

func (manager *TxManager) unlock(hash common.Hash) {
	manager.contextLock[hash].Unlock(fmt.Sprintf("txM: %s", hash.String()))

	manager.lock.Lock()
	defer manager.lock.Unlock()

	delete(manager.contextLock, hash)
}

func (manager *TxManager) getTxLock(hash common.Hash) *middleware.Loglock {
	txLock := manager.contextLock[hash]
	if nil == txLock {
		manager.lock.Lock()
		txLock = manager.contextLock[hash]
		if nil == txLock {
			lock:=middleware.NewLoglock("tx_manager")
			txLock = &lock
			manager.contextLock[hash] = txLock
		}
		manager.lock.Unlock()
	}

	return txLock
}
