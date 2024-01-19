package executor

import (
	"com.tuntun.rangers/node/src/middleware/log"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/service"
	"com.tuntun.rangers/node/src/storage/account"
)

type jsonrpcExecutor struct {
	contractExecutor
	logger log.Logger
}

func (this *jsonrpcExecutor) BeforeExecute(tx *types.Transaction, header *types.BlockHeader, accountDB *account.AccountDB, context map[string]interface{}) (bool, bool, string) {
	if err := validateNonce(tx, accountDB); err != nil {
		return false, false, err.Error()
	}

	if err := service.GetTransactionPool().ProcessFee(*tx, accountDB); err != nil {
		return false, false, err.Error()
	}

	raw, errMessage := this.decodeContractData(tx.Data)
	if errMessage != "" {
		return false, false, errMessage
	}
	context["contractData"] = raw

	if err := preCheckContractFee(tx, accountDB, *raw); err != nil {
		return false, false, err.Error()
	}
	return true, true, ""
}

func (this *jsonrpcExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	return this.contractExecutor.Execute(transaction, header, accountdb, context)
}
