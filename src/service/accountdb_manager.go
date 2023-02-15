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
	"com.tuntun.rocket/node/src/middleware/notify"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/storage/trie"
)

const stateDBPrefix = "state"

type AccountDBManager struct {
	stateDB       account.AccountDatabase
	LatestStateDB *account.AccountDB
	requestId     uint64
	Height        uint64
	logger        log.Logger

	waitingTxs *PriorityQueue
	writeChan  chan *notify.ClientTransactionMessage
}

const maxWriteSize = 100000

var AccountDBManagerInstance AccountDBManager

func initAccountDBManager() {
	AccountDBManagerInstance = AccountDBManager{}

	db, err := db.NewLDBDatabase(stateDBPrefix, 128, 2048)
	if err != nil {
		AccountDBManagerInstance.logger.Errorf("Init accountDB error! Error:%s", err.Error())
		panic(err)
	}
	AccountDBManagerInstance.stateDB = account.NewDatabase(db)

	AccountDBManagerInstance.logger = log.GetLoggerByIndex(log.AccountDBLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	AccountDBManagerInstance.waitingTxs = NewPriorityQueue()
	AccountDBManagerInstance.writeChan = make(chan *notify.ClientTransactionMessage, maxWriteSize)
	AccountDBManagerInstance.loop()

	notify.BUS.Subscribe(notify.ClientTransaction, AccountDBManagerInstance.write)
}

func (manager *AccountDBManager) GetAccountDBByHash(hash common.Hash) (*account.AccountDB, error) {
	//todo: cache
	return account.NewAccountDB(hash, manager.stateDB)
}

func (manager *AccountDBManager) GetLatestStateDB() *account.AccountDB {
	return manager.LatestStateDB
}

func (manager *AccountDBManager) GetTrieDB() *trie.NodeDatabase {
	return manager.stateDB.TrieDB()
}

func (manager *AccountDBManager) SetLatestStateDB(latestStateDB *account.AccountDB, requestIds map[string]uint64, height uint64) {
	// 这里无需加锁，因为外面加过了
	key := "fixed"
	nonce := requestIds[key]
	//manager.SetLatestStateDBWithNonce(latestStateDB, nonce, "add block", height)
	manager.Height = height
	if nil == manager.LatestStateDB || nonce >= manager.waitingTxs.GetThreshold() {
		if nil != latestStateDB {
			manager.LatestStateDB = latestStateDB
		}
		manager.waitingTxs.SetThreshold(nonce)
	}
}

func (manager *AccountDBManager) write(msg notify.Message) {
	message, ok := msg.(*notify.ClientTransactionMessage)
	if !ok {
		manager.logger.Errorf("AccountDBManager: Write assert not ok!")
		return
	}

	if len(manager.writeChan) == maxWriteSize {
		manager.logger.Errorf("write rcv message error: %v", msg)
		return
	}

	manager.logger.Debugf("write rcv message. hash: %s, nonce: %d", message.Tx.Hash.String(), message.Nonce)
	manager.writeChan <- message
}

func (manager *AccountDBManager) loop() {
	go func() {
		for {
			select {
			case message := <-manager.writeChan:
				manager.waitingTxs.heapPush(message)
			}
		}
	}()
}

func (manager *AccountDBManager) SetHandler(handler func(message *Item)) {
	manager.waitingTxs.SetHandle(handler)
}
