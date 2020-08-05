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
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/db"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/storage/trie"
	"sync"
	"sync/atomic"
)

const stateDBPrefix = "state"

type AccountDBManager struct {
	lock          *sync.RWMutex //用于锁定BASE
	conds         sync.Map
	stateDB       account.AccountDatabase
	latestStateDB *account.AccountDB
	requestId     uint64
	debug         bool // debug 为true，则不开启requestId校验
	logger        log.Logger
}

var (
	AccountDBManagerInstance AccountDBManager
)

func initAccountDBManager() {
	AccountDBManagerInstance = AccountDBManager{}
	AccountDBManagerInstance.lock = &sync.RWMutex{}
	AccountDBManagerInstance.conds = sync.Map{}
	AccountDBManagerInstance.logger = log.GetLoggerByIndex(log.AccountDBLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	//if nil != common.GlobalConf {
	//	AccountDBManagerInstance.debug = common.GlobalConf.GetBool("gx", "debug", true)
	//} else {
	//	AccountDBManagerInstance.debug = true
	//}

	AccountDBManagerInstance.debug = false
	db, err := db.NewDatabase(stateDBPrefix)
	if err != nil {
		AccountDBManagerInstance.logger.Errorf("Init accountDB error! Error:%s", err.Error())
		panic(err)
	}
	AccountDBManagerInstance.stateDB = account.NewDatabase(db)
}

func (manager *AccountDBManager) GetAccountDBByGameExecutor(nonce uint64) *account.AccountDB {
	waited := false
	// 校验 nonce
	if !manager.debug {
		// requestId 按序执行
		manager.getCond().L.Lock()

		for nonce != (manager.getRequestId() + 1) {
			if nonce <= manager.getRequestId() {
				// 已经执行过的消息，忽略
				manager.logger.Errorf("requestId :%d skipped, current requestId: %d", nonce, manager.getRequestId())
				manager.getCond().L.Unlock()
				return nil
			}

			// waiting until the right requestId
			manager.logger.Infof("requestId :%d is waiting, current requestId: %d", nonce, manager.getRequestId())
			waited = true
			manager.getCond().Wait()
		}
	}

	manager.lock.RLock()
	defer manager.lock.RUnlock()

	// waiting until the right requestId
	if waited {
		manager.logger.Infof("requestId :%d waited, current requestId: %d", nonce, manager.getRequestId())
	}

	return manager.latestStateDB
}

//todo: 功能增强
func (manager *AccountDBManager) GetAccountDB(gameId string, isBase bool) *account.AccountDB {
	return manager.GetLatestStateDB()
}

func (manager *AccountDBManager) GetAccountDBByHash(hash common.Hash) (*account.AccountDB, error) {
	//todo: cache
	return account.NewAccountDB(hash, manager.stateDB)
}

func (manager *AccountDBManager) GetLatestStateDB() *account.AccountDB {
	manager.lock.RLock()
	defer manager.lock.RUnlock()

	return manager.latestStateDB
}

//
func (manager *AccountDBManager) SetLatestStateDBWithNonce(latestStateDB *account.AccountDB, nonce uint64, msg string) {
	if !manager.debug && msg == "gameExecutor" {
		manager.getCond().L.Unlock()
	}

	manager.lock.Lock()
	defer manager.lock.Unlock()

	if nil == manager.latestStateDB || nonce >= manager.getRequestId() {
		manager.logger.Warnf("accountDB set success. requestId: %d, current: %d, msg: %s", nonce, manager.getRequestId(), msg)

		manager.latestStateDB = latestStateDB
		manager.setRequestId(nonce)

		if !manager.debug && nonce >= manager.getRequestId() {
			manager.getCond().Broadcast()
		}
	} else {
		manager.logger.Warnf("accountDB not set. requestId: %d, current: %d, msg: %s", nonce, manager.getRequestId(), msg)
	}
}

func (manager *AccountDBManager) SetLatestStateDB(latestStateDB *account.AccountDB, requestIds map[string]uint64) {
	key := "fixed"
	value := requestIds[key]
	manager.SetLatestStateDBWithNonce(latestStateDB, value, "add block")
}

func (manager *AccountDBManager) GetTrieDB() *trie.NodeDatabase {
	return manager.stateDB.TrieDB()
}

func (manager *AccountDBManager) getCond() *sync.Cond {
	gameId := "fixed"
	defaultValue := sync.NewCond(new(sync.Mutex))
	value, _ := manager.conds.LoadOrStore(gameId, defaultValue)

	return value.(*sync.Cond)
}

func (manager *AccountDBManager) getRequestId() uint64 {
	return atomic.LoadUint64(&manager.requestId)
}

func (manager *AccountDBManager) setRequestId(value uint64) {
	atomic.StoreUint64(&manager.requestId, value)
}
