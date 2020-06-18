package service

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/db"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/storage/trie"
	"sync"
)

const stateDBPrefix = "state"

type AccountDBManager struct {
	lock          *sync.RWMutex //用于锁定BASE
	conds         sync.Map
	stateDB       account.AccountDatabase
	latestStateDB *account.AccountDB
	requestId     uint64
	debug         bool // debug 为true，则不开启requestId校验
}

var AccountDBManagerInstance AccountDBManager

func initAccountDBManager() {
	AccountDBManagerInstance = AccountDBManager{}
	AccountDBManagerInstance.lock = &sync.RWMutex{}
	AccountDBManagerInstance.conds = sync.Map{}
	//if nil != common.GlobalConf {
	//	AccountDBManagerInstance.debug = common.GlobalConf.GetBool("gx", "debug", true)
	//} else {
	//	AccountDBManagerInstance.debug = true
	//}

	AccountDBManagerInstance.debug = false
	db, err := db.NewDatabase(stateDBPrefix)
	if err != nil {
		logger.Errorf("Init accountDB error! Error:%s", err.Error())
		panic(err)
	}
	AccountDBManagerInstance.stateDB = account.NewDatabase(db)
}

func (manager *AccountDBManager) GetAccountDBByGameExecutor(nonce uint64) *account.AccountDB {
	waited := false

	// 校验 nonce
	if !manager.debug {
		if nonce <= manager.requestId {
			// 已经执行过的消息，忽略
			logger.Errorf("%s requestId :%d skipped, current requestId: %d", "", nonce, manager.requestId)
			return nil
		}

		// requestId 按序执行
		manager.getCond().L.Lock()
		for ; nonce != (manager.requestId + 1); {
			if nonce <= manager.requestId {
				// 已经执行过的消息，忽略
				logger.Errorf("%s requestId :%d skipped, current requestId: %d", "", nonce, manager.requestId)
				manager.getCond().L.Unlock()
				return nil
			}

			// waiting until the right requestId
			logger.Infof("requestId :%d is waiting, current requestId: %d", nonce, manager.requestId)
			waited = true

			// todo 超时放弃
			manager.getCond().Wait()

		}
	}

	manager.lock.RLock()
	defer manager.lock.RUnlock()

	// waiting until the right requestId
	if waited {
		logger.Infof("requestId :%d waited, current requestId: %d", nonce, manager.requestId)
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

	if nil == manager.latestStateDB || nonce > manager.requestId {
		logger.Warnf("accountDB set success. requestId: %d, current: %d, msg: %s", nonce, manager.requestId, msg)

		manager.latestStateDB = latestStateDB
		manager.requestId = nonce

		if !manager.debug {
			manager.getCond().Broadcast()
		}
	} else {
		logger.Warnf("accountDB not set. requestId: %d, current: %d, msg: %s", nonce, manager.requestId, msg)
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
