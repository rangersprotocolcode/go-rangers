package core

import (
	"x/src/middleware/notify"
	"sync"
	"x/src/storage/account"
)

type AccountDBManager struct {
	accountDB *account.AccountDB
	lock      *sync.RWMutex //用于锁定BASE
}

var AccountDBManagerInstance *AccountDBManager

func initAccountDBManager() {
	AccountDBManagerInstance = &AccountDBManager{}
	AccountDBManagerInstance.lock = &sync.RWMutex{}

	// chain初始化在前，所以链上数据已经有了
	AccountDBManagerInstance.onBlockAddSuccess(nil)
	notify.BUS.Subscribe(notify.BlockAddSucc, AccountDBManagerInstance.onBlockAddSuccess)
}

//事件onBlockAddSuccess，表示着上链已经完成，链上已经是最新状态了
func (manager *AccountDBManager) onBlockAddSuccess(message notify.Message) {
	manager.lock.Lock()
	defer manager.lock.Unlock()
	manager.accountDB = GetBlockChain().GetAccountDB()
}

func (manager *AccountDBManager) GetAccountDB(gameId string, isBase bool) *account.AccountDB {
	return manager.accountDB
}
