// Copyright 2020 The RangersProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RangersProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RangersProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package executor

import (
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
)

type executor interface {
	BeforeExecute(tx *types.Transaction, header *types.BlockHeader, accountDB *account.AccountDB, context map[string]interface{}) (bool, string)
	Execute(tx *types.Transaction, header *types.BlockHeader, accountDB *account.AccountDB, context map[string]interface{}) (bool, string)
}

type baseFeeExecutor struct {
}

func (this *baseFeeExecutor) BeforeExecute(tx *types.Transaction, header *types.BlockHeader, accountDB *account.AccountDB, context map[string]interface{}) (bool, string) {
	err := service.GetTransactionPool().ProcessFee(*tx, accountDB)
	if err == nil {
		return true, ""
	}
	return false, err.Error()
}
