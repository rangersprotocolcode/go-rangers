package core

import (
	"x/src/middleware/types"
	"x/src/storage/account"
	"x/src/service"
)

type withdrawExecutor struct {
}

func (this *withdrawExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) bool {
	_, success := service.Withdraw(accountdb, transaction, true)
	return success
}
