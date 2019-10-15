package core

import (
	"x/src/middleware/notify"
	"sync"
	"x/src/storage/account"
)

const BASE = "base"

// 基于gameId的accountdb管理器
// 用于实时计算
type AccountDBManager struct {
	context map[string]*accountContext
	lock    *sync.RWMutex //用于锁定BASE
}

type accountContext struct {
	accountDB *account.AccountDB
	height    uint64
	lock      *sync.Mutex // 锁自己
}

var AccountDBManagerInstance *AccountDBManager

func initAccountDBManager() {
	AccountDBManagerInstance = &AccountDBManager{}
	AccountDBManagerInstance.lock = &sync.RWMutex{}
	AccountDBManagerInstance.context = make(map[string]*accountContext)

	// chain初始化在前，所以链上数据已经有了
	//AccountDBManagerInstance.onBlockAddSuccess(nil)
	//notify.BUS.Subscribe(notify.BlockAddSucc, AccountDBManagerInstance.onBlockAddSuccess)
}

//事件onBlockAddSuccess，表示着上链已经完成，链上已经是最新状态了
func (manager *AccountDBManager) onBlockAddSuccess(message notify.Message) {
	context := &accountContext{height: GetBlockChain().Height(), accountDB: GetBlockChain().GetAccountDB()}

	manager.lock.Lock()
	defer manager.lock.Unlock()
	manager.context[BASE] = context
}

func (manager *AccountDBManager) GetAccountDB(gameId string, isBase bool) *account.AccountDB {
	return blockChainImpl.GetAccountDB()
}
