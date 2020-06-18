package core

import (
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
)

type coinDepositExecutor struct {
}

type ftDepositExecutor struct {
}

type nftDepositExecutor struct {
}

func (this *coinDepositExecutor) BeforeExecute(tx *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	return true, ""
}

//主链币充值确认
func (this *coinDepositExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	return service.CoinDeposit(accountdb, transaction)
}

func (this *ftDepositExecutor) BeforeExecute(tx *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	return true, ""
}

//FT充值确认
func (this *ftDepositExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	return service.FTDeposit(accountdb, transaction)
}

func (this *nftDepositExecutor) BeforeExecute(tx *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	return true, ""
}

//NFT充值确认
func (this *nftDepositExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	return service.NFTDeposit(accountdb, transaction)
}
