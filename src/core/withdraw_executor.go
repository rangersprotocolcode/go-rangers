package core

import (
	"x/src/middleware/types"
	"x/src/service"
	"x/src/storage/account"
)

type withdrawExecutor struct {
}

func (this *withdrawExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	if err := service.GetTransactionPool().ProcessFee(*transaction, accountdb); err != nil {
		return false, "not enough fee"
	}
	msg, success := service.Withdraw(accountdb, transaction, true)
	return success, msg
}
