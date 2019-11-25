package service

import (
	"sync"
	"x/src/storage/account"
	"x/src/common"
	"x/src/middleware/db"
	"x/src/storage/trie"
)

const stateDBPrefix = "state"

type AccountDBManager struct {
	accountDB *account.AccountDB
	lock      *sync.RWMutex //用于锁定BASE

	stateDB       account.AccountDatabase
	latestStateDB *account.AccountDB
}

var AccountDBManagerInstance *AccountDBManager

func initAccountDBManager() {
	AccountDBManagerInstance = &AccountDBManager{}
	AccountDBManagerInstance.lock = &sync.RWMutex{}

	db, err := db.NewDatabase(stateDBPrefix)
	if err != nil {
		logger.Errorf("Init accountDB error! Error:%s", err.Error())
		panic(err)
	}
	AccountDBManagerInstance.stateDB = account.NewDatabase(db)
}

func (manager *AccountDBManager) GetAccountDB(gameId string, isBase bool) *account.AccountDB {
	return manager.latestStateDB
}

func (manager *AccountDBManager) GetAccountDBByHash(hash common.Hash) (*account.AccountDB, error) {
	return account.NewAccountDB(hash, manager.stateDB)
}

func (manager *AccountDBManager) GetLatestStateDB() *account.AccountDB {
	manager.lock.RLock()
	defer manager.lock.RUnlock()

	return manager.latestStateDB
}

//
func (manager *AccountDBManager) SetLatestStateDB(latestStateDB *account.AccountDB) {
	manager.lock.Lock()
	defer manager.lock.Unlock()

	manager.latestStateDB = latestStateDB
}

func (manager *AccountDBManager) GetTrieDB() *trie.NodeDatabase {
	return manager.stateDB.TrieDB()
}