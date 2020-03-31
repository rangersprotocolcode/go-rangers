package core

import (
	"x/src/middleware/types"
	"x/src/service"
	"x/src/storage/account"
)

type withdrawExecutor struct {
	baseFeeExecutor
}

func (this *withdrawExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	msg, success := service.Withdraw(accountdb, transaction, true)
	return success, msg
}
