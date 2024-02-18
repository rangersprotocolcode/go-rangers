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
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package executor

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/service"
	"com.tuntun.rangers/node/src/storage/account"
	"errors"
)

var (
	ErrNonceTooLow  = errors.New("nonce too low")
	ErrNonceTooHigh = errors.New("nonce too high")
)

type executor interface {
	BeforeExecute(tx *types.Transaction, header *types.BlockHeader, accountDB *account.AccountDB, context map[string]interface{}) (bool, bool, string)
	Execute(tx *types.Transaction, header *types.BlockHeader, accountDB *account.AccountDB, context map[string]interface{}) (bool, string)
}

type baseFeeExecutor struct {
}

func (this *baseFeeExecutor) BeforeExecute(tx *types.Transaction, header *types.BlockHeader, accountDB *account.AccountDB, context map[string]interface{}) (bool, bool, string) {
	if err := validateNonce(tx, accountDB); err != nil {
		return false, true, err.Error()
	}

	if err := service.GetTransactionPool().ProcessFee(*tx, accountDB); err != nil {
		return false, true, err.Error()
	}
	return true, true, ""
}

func validateNonce(tx *types.Transaction, accountDB *account.AccountDB) error {
	if common.IsProposal018() {
		expectedNonce := accountDB.GetNonce(common.HexToAddress(tx.Source))
		if expectedNonce > tx.Nonce {
			logger.Debugf("Tx nonce too low.tx:%s,expected:%d,but:%d", tx.Hash.String(), expectedNonce, tx.Nonce)
			return ErrNonceTooLow
		} else if expectedNonce < tx.Nonce {
			logger.Debugf("Tx nonce too high.tx:%s,expected:%d,but:%d", tx.Hash.String(), expectedNonce, tx.Nonce)
			return ErrNonceTooHigh
		}
	}
	return nil
}
