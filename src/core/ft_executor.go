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
