package core

import (
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
)

type ftExecutor struct {
	baseFeeExecutor
}

func (this *ftExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
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
	case types.TransactionTypeUpdateNFT:
		success, msg = service.UpdateNFT(accountdb, transaction)
		break
	case types.TransactionTypeApproveNFT:
		success, msg = service.ApproveNFT(accountdb, transaction)
		break
	case types.TransactionTypeRevokeNFT:
		success, msg = service.RevokeNFT(accountdb, transaction)
		break
	}

	return success, msg
}
