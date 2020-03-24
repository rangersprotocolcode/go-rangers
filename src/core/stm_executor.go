package core

import (
	"x/src/middleware/types"
	"x/src/service"
	"x/src/statemachine"
	"x/src/storage/account"
)

type stmExecutor struct {
}

func (this *stmExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	if err := service.GetTransactionPool().ProcessFee(*transaction, accountdb); err != nil {
		return false, "not enough fee"
	}

	switch transaction.Type {
	case types.TransactionTypeAddStateMachine:
		// todo: 经济模型，新增状态机应该要付费
		go statemachine.STMManger.AddStatemachine(transaction.Source, transaction.Data)
		break
	case types.TransactionTypeUpdateStorage:
		// todo: 经济模型，更新状态机应该要付费
		go statemachine.STMManger.UpdateSTMStorage(transaction.Source, transaction.Data)
		break
	case types.TransactionTypeStartSTM:
		// todo: 经济模型，重启状态机应该要付费
		go statemachine.STMManger.StartSTM(transaction.Source)
		break
	case types.TransactionTypeStopSTM:
		// todo: 经济模型，重启状态机应该要付费
		go statemachine.STMManger.StopSTM(transaction.Source)
		break
	case types.TransactionTypeUpgradeSTM:
		// todo: 经济模型，重启状态机应该要付费
		go statemachine.STMManger.UpgradeSTM(transaction.Source, transaction.Data)
		break
	case types.TransactionTypeQuitSTM:
		// todo: 经济模型，重启状态机应该要付费
		go statemachine.STMManger.QuitSTM(transaction.Source)
		break
	case types.TransactionTypeImportNFT:
		appId := transaction.Source
		return statemachine.STMManger.IsAppId(appId), ""
	}

	return true, ""
}
