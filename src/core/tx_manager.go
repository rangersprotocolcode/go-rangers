package core

import (
	"x/src/storage/account"
	"x/src/middleware/types"
)

// 基于gameId的事务管理器
type TxManager struct {
	context map[string]*TxContext
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
}

func (manager *TxManager) BeginTransaction(gameId string, accountDB *account.AccountDB, tx *types.Transaction) {
	if nil == accountDB || nil == tx || 0 == len(gameId) {
		return
	}

	tx.SubTransactions = make([]string, 0)
	snapshot := accountDB.Snapshot()
	manager.context[gameId] = &TxContext{AccountDB: accountDB, Tx: tx, snapshot: snapshot}
}

func (manager *TxManager) GetContext(gameId string) *TxContext {
	return manager.context[gameId]
}

func (manager *TxManager) Commit(gameId string) {
	manager.remove(gameId)
}

func (manager *TxManager) RollBack(gameId string) {
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
