// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/statemachine"
	"com.tuntun.rocket/node/src/storage/account"
)

type stmExecutor struct {
	baseFeeExecutor
}


func (this *stmExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
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
