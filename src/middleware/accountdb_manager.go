// Copyright 2020 The RangersProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RangersProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RangersProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package middleware

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/middleware/db"
	"com.tuntun.rangers/node/src/middleware/log"
	"com.tuntun.rangers/node/src/storage/account"
	"com.tuntun.rangers/node/src/storage/trie"
	"strconv"
)

const stateDBPrefix = "state"

type AccountDBManager struct {
	stateDB       account.AccountDatabase
	LatestStateDB *account.AccountDB
	db            *db.LDBDatabase

	Height uint64
	logger log.Logger

	waitingTxs *PriorityQueue
}

var AccountDBManagerInstance AccountDBManager

func initAccountDBManager() {
	AccountDBManagerInstance = AccountDBManager{}

	AccountDBManagerInstance.logger = log.GetLoggerByIndex(log.AccountDBLogConfig, strconv.Itoa(common.InstanceIndex))

	db, err := db.NewLDBDatabase(stateDBPrefix, 2048, 2048)
	if err != nil {
		AccountDBManagerInstance.logger.Errorf("Init accountDB error! Error:%s", err.Error())
		panic(err)
	}
	AccountDBManagerInstance.db = db
	AccountDBManagerInstance.stateDB = account.NewDatabase(db)

	AccountDBManagerInstance.waitingTxs = NewPriorityQueue()

	AccountDBManagerInstance.loop()
}

func (manager *AccountDBManager) Close() {
	if nil != manager.db {
		manager.db.Close()
	}
}

func (manager *AccountDBManager) GetAccountDBByHash(hash common.Hash) (*account.AccountDB, error) {
	return account.NewAccountDB(hash, manager.stateDB)
}

func (manager *AccountDBManager) GetLatestStateDB() *account.AccountDB {
	return manager.LatestStateDB
}

func (manager *AccountDBManager) GetTrieDB() *trie.NodeDatabase {
	return manager.stateDB.TrieDB()
}

func (manager *AccountDBManager) SetLatestStateDB(latestStateDB *account.AccountDB, nonces map[string]uint64, height uint64) {
	key := "fixed"
	nonce := nonces[key]

	testNonce := manager.LatestStateDB.GetNonce(common.HexToAddress("0x2f4F09b722a6e5b77bE17c9A99c785Fa7035a09f"))
	common.DefaultLogger.Debugf("before SetLatestStateDB,height:%d,0x2f4F09b722a6e5b77bE17c9A99c785Fa7035a09f nonce:%d", height, testNonce)
	common.DefaultLogger.Debugf("nonce:%d,Threshold:%d", nonce, manager.waitingTxs.GetThreshold)
	//manager.SetLatestStateDBWithNonce(latestStateDB, nonce, "add block", height)
	manager.Height = height
	if nil == manager.LatestStateDB || nonce >= manager.waitingTxs.GetThreshold() {
		if nil != latestStateDB {
			common.DefaultLogger.Debugf("latest db is not nil")
			manager.LatestStateDB = latestStateDB
		}

		if nonce > 0 {
			manager.waitingTxs.SetThreshold(nonce)
		}
	}
	testNonce = manager.LatestStateDB.GetNonce(common.HexToAddress("0x2f4F09b722a6e5b77bE17c9A99c785Fa7035a09f"))
	common.DefaultLogger.Debugf("after SetLatestStateDB,height:%d,0x2f4F09b722a6e5b77bE17c9A99c785Fa7035a09f nonce:%d", height, testNonce)
}

func (manager *AccountDBManager) loop() {
	go func() {
		for {
			select {
			case message := <-DataChannel.GetRcvedTx():
				manager.logger.Debugf("write rcv message. hash: %s, nonce: %d", message.Tx.Hash.String(), message.Nonce)
				manager.waitingTxs.heapPush(message)

				//txRaw := message.Tx
				//txRaw.RequestId = message.Nonce
				//if txRaw.Type == 0 || 0 == txRaw.RequestId {
				//	msg := notify.ClientTransactionMessage{Tx: txRaw}
				//	notify.BUS.Publish(notify.ClientTransactionWrite, &msg)
				//} else {
				//	manager.waitingTxs.heapPush(message)
				//}

			}
		}
	}()
}

func (manager *AccountDBManager) SetHandler(handler func(message *Item)) {
	manager.waitingTxs.SetHandle(handler)
}

func (manager *AccountDBManager) GetThreshold() uint64 {
	return manager.waitingTxs.threshold
}
