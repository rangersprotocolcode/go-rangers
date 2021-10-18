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

type coinDepositExecutor struct {
}

type ftDepositExecutor struct {
}

type nftDepositExecutor struct {
}

type erc20BindingExecutor struct {
}

func (this *coinDepositExecutor) BeforeExecute(tx *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	return true, ""
}

//主链币充值确认
func (this *coinDepositExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	return service.CoinDeposit(accountdb, transaction)
}

func (this *ftDepositExecutor) BeforeExecute(tx *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	return true, ""
}

//FT充值确认
func (this *ftDepositExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	return service.FTDeposit(accountdb, transaction)
}

func (this *nftDepositExecutor) BeforeExecute(tx *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	return true, ""
}

//NFT充值确认
func (this *nftDepositExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	return service.NFTDeposit(accountdb, transaction)
}


func (this *erc20BindingExecutor) BeforeExecute(tx *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	return true, ""
}

//NFT充值确认
func (this *erc20BindingExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	return service.ERC20Binding(accountdb, transaction)
}
