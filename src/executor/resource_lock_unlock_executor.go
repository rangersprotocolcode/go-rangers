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

package executor

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"encoding/json"
)

type resourceLockUnLockExecutor struct {
	baseFeeExecutor
	logger log.Logger
}

func (this *resourceLockUnLockExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	success := false
	msg := ""

	data := transaction.Data
	if 0 == len(data) {
		return false, "no data"
	}
	resource := types.LockResource{}
	if err := json.Unmarshal(utility.StrToBytes(data), &resource); nil != err {
		return false, "data format err: " + data
	}

	switch transaction.Type {
	case types.TransactionTypeLockResource:
		success = accountdb.LockResource(common.HexToAddress(transaction.Source), common.GenerateNFTSetAddress(transaction.Target), resource)
		if success {
			msg = "resource locked successful"
		} else {
			msg = "resource locked failed"
		}
		break
	case types.TransactionTypeUnLockResource:
		success = accountdb.UnLockResource(common.HexToAddress(transaction.Source), common.GenerateNFTSetAddress(transaction.Target), resource)
		if success {
			msg = "resource unlocked successful"
		} else {
			msg = "resource unlocked failed"
		}
		break
	case types.TransactionTypeComboNFT:
		success, msg = service.ComboNFT(accountdb, transaction)
		break
	}

	return success, msg
}
