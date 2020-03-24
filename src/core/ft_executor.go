package core

import (
	"x/src/middleware/types"
	"x/src/service"
	"x/src/storage/account"
)

type ftExecutor struct {
}

func (this *ftExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	if err := service.GetTransactionPool().ProcessFee(*transaction, accountdb); err != nil {
		return false, "not enough fee"
	}

	success := false
	msg := ""

	switch transaction.Type {
	case types.TransactionTypePublishFT:
		msg, success = service.PublishFT(accountdb, transaction)
		break
	case types.TransactionTypePublishNFTSet:
		success, msg = service.PublishNFTSet(accountdb, transaction)
		break
	case types.TransactionTypeMintFT:
		success, msg = service.MintFT(accountdb, transaction)
		break
	case types.TransactionTypeMintNFT:
		success, msg = service.MintNFT(accountdb, transaction)
		break
	case types.TransactionTypeShuttleNFT:
		success, msg = service.ShuttleNFT(accountdb, transaction)
		break
	}

	return success, msg
}
