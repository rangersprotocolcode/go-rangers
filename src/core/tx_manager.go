package core

import (
	"x/src/storage/account"
	"x/src/middleware/types"
	"sync"
	"fmt"
)

// 基于gameId的事务管理器
type TxManager struct {
	context map[string]*TxContext
	lock    *sync.Mutex
}

type TxContext struct {
	AccountDB *account.AccountDB
	Tx        *types.Transaction
	snapshot  int
	lock      *sync.Mutex
	count     int
}

var TxManagerInstance *TxManager

func initTxManager() {
	TxManagerInstance = &TxManager{}

	TxManagerInstance.context = make(map[string]*TxContext)
	TxManagerInstance.lock = &sync.Mutex{}
}

func (manager *TxManager) BeginTransaction(gameId string, accountDB *account.AccountDB, tx *types.Transaction) error {
	if nil == accountDB || nil == tx || 0 == len(gameId) {
		return fmt.Errorf("no value")
	}

	// 已经执行过了
	if GetBlockChain().GetTransactionPool().IsExisted(tx.Hash) {
		return fmt.Errorf("isExisted")
	}

	context := manager.context[gameId]
	if nil == context {
		manager.lock.Lock()
		context = manager.context[gameId]
		if nil == context {
			context = &TxContext{lock: &sync.Mutex{}}
			manager.context[gameId] = context
		}
		manager.lock.Unlock()
	}

	context.lock.Lock()
	context.snapshot = accountDB.Snapshot()
	context.AccountDB = accountDB
	context.Tx = tx
	context.count = context.count + 1

	logger.Errorf("BeginTransaction. %s %s %d", gameId, tx.Hash, context.count)

	return nil
}

func (manager *TxManager) GetContext(gameId string) *TxContext {
	return manager.context[gameId]
}

func (manager *TxManager) Commit(gameId string) {
	manager.clean(false, gameId)
}

func (manager *TxManager) RollBack(gameId string) {
	manager.clean(true, gameId)
}

func (manager *TxManager) clean(isRollback bool, gameId string) {
	context := manager.GetContext(gameId)
	if nil == context {
		return
	}

	logger.Errorf("endTransaction. %s %s %d %s", gameId, context.Tx.Hash, context.count-1, isRollback)

	if isRollback {
		context.AccountDB.RevertToSnapshot(context.snapshot)
	}

	if context.count == 0 {

	}
	context.AccountDB = nil
	context.Tx = nil
	context.count = context.count - 1
	context.lock.Unlock()
}
