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
}

var TxManagerInstance *TxManager

func initTxManager() {
	TxManagerInstance = &TxManager{}

	TxManagerInstance.context = make(map[string]*TxContext)
	TxManagerInstance.lock = &sync.Mutex{}
}

func (manager *TxManager) BeginTransaction(gameId string, accountDB *account.AccountDB, tx *types.Transaction) error {
	if nil == accountDB || nil == tx {
		return fmt.Errorf("no value")
	}

	// 已经执行过了
	if GetBlockChain().GetTransactionPool().IsExisted(tx.Hash) {
		return fmt.Errorf("isExisted")
	}

	// 如果gameId为空，则为纯转账场景，无须在此开事务
	if 0 == len(gameId) {
		return nil
	}

	context := manager.context[gameId]
	if nil == context {
		logger.Debugf("context is nil")
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

	return nil
}

func (manager *TxManager) GetContext(gameId string) *TxContext {
	return manager.context[gameId]
}

func (manager *TxManager) Commit(gameId string) {
	if 0==len(gameId){
		return
	}
	manager.clean(false, gameId)
}

func (manager *TxManager) RollBack(gameId string) {
	if 0==len(gameId){
		return
	}
	manager.clean(true, gameId)
}

func (manager *TxManager) clean(isRollback bool, gameId string) {
	logger.Debugf("is rollback %t,game id:%s", isRollback, gameId)
	context := manager.GetContext(gameId)
	if nil == context {
		logger.Debugf("clean context is nil")
		return
	}

	if isRollback {
		context.AccountDB.RevertToSnapshot(context.snapshot)
	}

	context.AccountDB = nil
	context.Tx = nil
	context.lock.Unlock()
}
