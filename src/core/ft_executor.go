package core

import (
	"x/src/middleware/types"
	"x/src/storage/account"
	"x/src/service"
)

type ftExecutor struct {
}

func (this *ftExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) bool {
	if err := service.GetTransactionPool().ProcessFee(*transaction, accountdb); err != nil {
		return false
	}

	success := false
	switch transaction.Type {
	case types.TransactionTypePublishFT:
		_, success = service.PublishFT(accountdb, transaction)
		break
	case types.TransactionTypePublishNFTSet:
		success, _ = service.PublishNFTSet(accountdb, transaction)
		break
	case types.TransactionTypeMintFT:
		success, _ = service.MintFT(accountdb, transaction)
		break
	case types.TransactionTypeMintNFT:
		success, _ = service.MintNFT(accountdb, transaction)
		break
	case types.TransactionTypeShuttleNFT:
		success, _ = service.ShuttleNFT(accountdb, transaction)
		break
	}

	return success
}
