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
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/db"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/mysql"
	"com.tuntun.rocket/node/src/middleware/notify"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/storage/trie"
	"com.tuntun.rocket/node/src/utility"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

const stateDBPrefix = "state"

type AccountDBManager struct {
	conds         sync.Map
	stateDB       account.AccountDatabase
	LatestStateDB *account.AccountDB
	requestId     uint64
	Height        uint64
	debug         bool // debug 为true，则不开启requestId校验
	logger        log.Logger

	WaitingTxs *PriorityQueue
	NewTxs     chan byte
}

var AccountDBManagerInstance AccountDBManager

func initAccountDBManager() {
	AccountDBManagerInstance = AccountDBManager{}
	AccountDBManagerInstance.conds = sync.Map{}
	AccountDBManagerInstance.logger = log.GetLoggerByIndex(log.AccountDBLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	AccountDBManagerInstance.debug = false
	AccountDBManagerInstance.WaitingTxs = NewPriorityQueue()
	AccountDBManagerInstance.NewTxs = make(chan byte, 1)

	db, err := db.NewLDBDatabase(stateDBPrefix, 128, 2048)
	if err != nil {
		AccountDBManagerInstance.logger.Errorf("Init accountDB error! Error:%s", err.Error())
		panic(err)
	}
	AccountDBManagerInstance.stateDB = account.NewDatabase(db)

	//AccountDBManagerInstance.getTxList()
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
	middleware.RLockAccountDB("GetLatestStateDB")
	defer middleware.RUnLockAccountDB("GetLatestStateDB")

	return manager.LatestStateDB
}

func (manager *AccountDBManager) GetTrieDB() *trie.NodeDatabase {
	return manager.stateDB.TrieDB()
}

// GetAccountDBByGameExecutor deprecated
func (manager *AccountDBManager) GetAccountDBByGameExecutor(nonce uint64) (*account.AccountDB, uint64) {
	waited := false
	req := manager.requestId

	// 校验 nonce
	if !manager.debug {
		// requestId 按序执行
		manager.getCond().L.Lock()

		for nonce != (manager.requestId + 1) {
			if nonce <= manager.requestId {
				// 已经执行过的消息，忽略
				manager.logger.Errorf("%s requestId :%d skipped, current requestId: %d", "", nonce, manager.requestId)
				manager.getCond().L.Unlock()
				return nil, 0
			}

			// waiting until the right requestId
			manager.logger.Infof("requestId :%d is waiting, current requestId: %d", nonce, manager.requestId)
			waited = true

			// todo 超时放弃
			manager.getCond().Wait()
		}
	}

	// waiting until the right requestId
	if waited {
		manager.logger.Infof("requestId: %d waited, since: %d", nonce, req)
	}
	return manager.LatestStateDB, manager.Height
}

// SetLatestStateDBWithNonce 设置nonce deprecated
func (manager *AccountDBManager) SetLatestStateDBWithNonce(latestStateDB *account.AccountDB, nonce uint64, msg string, height uint64) {
	defer manager.getCond().L.Unlock()
	if !manager.debug && msg != "gameExecutor" {
		manager.getCond().L.Lock()
	}

	manager.Height = height
	if nil == manager.LatestStateDB || nonce >= manager.requestId {
		if nil != latestStateDB {
			manager.LatestStateDB = latestStateDB
		}

		manager.requestId = nonce
		manager.logger.Warnf("accountDB set success. requestId: %d, current: %d, msg: %s", nonce, manager.requestId, msg)

		if !manager.debug && nonce >= manager.requestId {
			manager.getCond().Broadcast()
		}

		return
	}

	manager.logger.Warnf("accountDB not set. requestId: %d, current: %d, msg: %s", nonce, manager.requestId, msg)
}

func (manager *AccountDBManager) SetLatestStateDB(latestStateDB *account.AccountDB, requestIds map[string]uint64, height uint64) {
	middleware.LockAccountDB("SetLatestStateDB")
	defer middleware.UnLockAccountDB("SetLatestStateDB")

	key := "fixed"
	nonce := requestIds[key]
	//manager.SetLatestStateDBWithNonce(latestStateDB, nonce, "add block", height)
	manager.Height = height
	if nil == manager.LatestStateDB || nonce >= manager.WaitingTxs.GetThreshold() {
		if nil != latestStateDB {
			manager.LatestStateDB = latestStateDB
		}
		manager.WaitingTxs.SetThreshold(nonce)

		manager.NewTxs <- 1
	}
}

// deprecated
func (manager *AccountDBManager) getCond() *sync.Cond {
	gameId := "fixed"
	defaultValue := sync.NewCond(new(sync.Mutex))
	value, _ := manager.conds.LoadOrStore(gameId, defaultValue)

	return value.(*sync.Cond)
}

// GetLatestNonce deprecated
func (manager *AccountDBManager) GetLatestNonce() uint64 {
	manager.getCond().L.Lock()
	defer manager.getCond().L.Unlock()

	return manager.requestId
}

// deprecated
func (manager *AccountDBManager) getTxList() {
	go func() {
		for {
			txs := mysql.GetTxRaws(manager.GetLatestNonce())
			if nil != txs {
				for _, tx := range txs {
					var txJson types.TxJson
					err := json.Unmarshal(utility.StrToBytes(tx.Data), &txJson)
					if nil != err {
						msg := fmt.Sprintf("handleClientMessage json unmarshal client message error:%s", err.Error())
						manager.logger.Errorf(msg)
					}
					transaction := txJson.ToTransaction()

					msg := notify.ClientTransactionMessage{Tx: transaction, UserId: tx.UserId, Nonce: tx.Nonce, GateNonce: tx.GateNonce}
					notify.BUS.Publish(notify.ClientTransaction, &msg)
				}
			}
			time.Sleep(time.Millisecond * 10)
		}
	}()
}
