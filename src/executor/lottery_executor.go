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
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
)

type lotteryExecutor struct {
	baseFeeExecutor
}

func (this *lotteryExecutor) Execute(tx *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	msg := ""
	switch tx.Type {
	case types.TransactionTypeLotteryCreate:
		msg, _ = service.CreateLottery(tx.Source, tx.Data, accountdb)
		break
	case types.TransactionTypeJackpot:
		msg, _ = service.Jackpot(tx.Target, tx.Source, tx.RequestId, header.Height, accountdb)
		break
	}

	return 0 != len(msg), msg
}
