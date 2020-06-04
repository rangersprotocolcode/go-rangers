package core

import (
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
)

type withdrawExecutor struct {
	baseFeeExecutor
}

func (this *withdrawExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	msg, success := service.Withdraw(accountdb, transaction, true)
	return success, msg
}
