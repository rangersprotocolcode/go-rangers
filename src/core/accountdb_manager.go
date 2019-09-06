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
	AccountDBManagerInstance.onBlockAddSuccess(nil)
	notify.BUS.Subscribe(notify.BlockAddSucc, AccountDBManagerInstance.onBlockAddSuccess)
}

//事件onBlockAddSuccess，表示着上链已经完成，链上已经是最新状态了
func (manager *AccountDBManager) onBlockAddSuccess(message notify.Message) {
	context := &accountContext{height: GetBlockChain().Height(), accountDB: GetBlockChain().GetAccountDB()}

	manager.lock.Lock()
	defer manager.lock.Unlock()
	manager.context[BASE] = context
}

func (manager *AccountDBManager) GetAccountDB(gameId string, isBase bool) *account.AccountDB {
	manager.lock.RLock()
	defer manager.lock.RUnlock()

	base := manager.context[BASE]
	if isBase {
		return base.accountDB
	}
	context := manager.context[gameId]
	if nil == context {
		// new
		manager.lock.Lock()
		context = manager.context[gameId]
		if nil == context {
			context = &accountContext{accountDB: base.accountDB.Copy(), height: base.height, lock: &sync.Mutex{}}
			manager.context[gameId] = context
		}
		manager.lock.Unlock()
	} else if base.height > context.height {
		// 这里要小心分叉判断
		// update
		context.lock.Lock()
		// 再次判断，防止并发多次copy
		if base.height > context.height {
			context.height = base.height
			context.accountDB = base.accountDB.Copy()
		}
		context.lock.Unlock()
	}

	// 返回的是指针，所以修改值的动作会自动生效
	return context.accountDB
}
