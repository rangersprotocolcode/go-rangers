package core

import (
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
)

type exchangeRateExecutor struct {
}

// 不扣手续费
func (this *exchangeRateExecutor) BeforeExecute(tx *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	return true, ""
}

func (this *exchangeRateExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	return service.SetExchangeRate(accountdb, transaction)
}
