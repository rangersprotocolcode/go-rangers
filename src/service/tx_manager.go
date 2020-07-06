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
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/account"
	"fmt"
	"sync"
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

// 只有执行状态机时，才会开启事务
// 此处的gameId不能为空
func (manager *TxManager) BeginTransaction(gameId string, accountDB *account.AccountDB, tx *types.Transaction) error {
	if nil == accountDB || nil == tx || 0 == len(gameId) {
		return fmt.Errorf("no value")
	}

	// 已经执行过了
	if GetTransactionPool().IsGameData(tx.Hash) {
		return fmt.Errorf("isExisted")
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
	if GetTransactionPool().IsGameData(tx.Hash) {
		context.lock.Unlock()
		return fmt.Errorf("gameData is Existed")
	}

	context.snapshot = accountDB.Snapshot()
	context.AccountDB = accountDB
	context.Tx = tx

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
	logger.Debugf("is rollback %t, game id:%s", isRollback, gameId)
	context := manager.GetContext(gameId)
	if nil == context {
		logger.Error("clean context is nil")
		return
	}

	if isRollback {
		context.AccountDB.RevertToSnapshot(context.snapshot)
	}

	context.AccountDB = nil
	context.Tx = nil
	context.lock.Unlock()
}
