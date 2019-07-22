package core

import (
	"x/src/storage/account"
	"x/src/middleware/types"
	"sync"
	"fmt"
	"x/src/common"
)

// 基于gameId的事务管理器
type TxManager struct {
	context     map[string]*TxContext
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
	TxManagerInstance.lock = &sync.Mutex{}
}

func (manager *TxManager) BeginTransaction(gameId string, accountDB *account.AccountDB, tx *types.Transaction, isCopy bool) error {
	if nil == accountDB || nil == tx || 0 == len(gameId) {
		return nil
	}

	// 已经执行过了
	if GetBlockChain().GetTransactionPool().IsExisted(tx.Hash) {
		return fmt.Errorf("isExisted")
	}

	tx.SubTransactions = make([]string, 0)

	copy := accountDB
	if isCopy {
		copy = copy.Copy()
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
}

func (manager *TxManager) RollBack(gameId string, hash common.Hash) {
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

