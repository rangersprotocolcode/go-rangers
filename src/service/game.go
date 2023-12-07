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

package service

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/storage/account"
	"com.tuntun.rangers/node/src/utility"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"strconv"
)

func GetRawBalance(source common.Address, accountDB *account.AccountDB) string {
	if accountDB == nil {
		return ""
	}
	balance := accountDB.GetBalance(source)
	if balance == nil {
		return "0"
	} else {
		return balance.String()
	}
}

func GetNetWorkId() string {
	return common.NetworkId()
}

func GetChainId() string {
	return common.ChainId(utility.MaxUint64)
}

func GetNonce(address common.Address, accountDB *account.AccountDB) string {
	if accountDB == nil {
		return ""
	}
	nonce := accountDB.GetNonce(address)
	return strconv.FormatUint(nonce, 10)
}

func GetMarshalReceipt(txHash common.Hash) string {
	transaction := GetTransactionPool().GetExecuted(txHash)
	if transaction == nil {
		return ""
	}
	if transaction.Receipt.Logs != nil && len(transaction.Receipt.Logs) != 0 {
		logs := make([]*types.Log, 0)
		for _, log := range transaction.Receipt.Logs {
			log.BlockHash = transaction.Receipt.BlockHash
			log.TxHash = transaction.Receipt.TxHash
			logs = append(logs, log)
		}
		transaction.Receipt.Logs = logs
	}
	result, _ := json.Marshal(transaction.Receipt)
	return string(result)
}

func GetReceipt(txHash common.Hash) *types.Receipt {
	transaction := GetTransactionPool().GetExecuted(txHash)
	if transaction == nil {
		return nil
	}
	if transaction.Receipt.Logs != nil && len(transaction.Receipt.Logs) != 0 {
		logs := make([]*types.Log, 0)
		for _, log := range transaction.Receipt.Logs {
			log.BlockHash = transaction.Receipt.BlockHash
			log.TxHash = transaction.Receipt.TxHash
			logs = append(logs, log)
		}
		transaction.Receipt.Logs = logs
	}
	return &transaction.Receipt.Receipt
}

func GetContractStorageAt(address string, key string, accountDB *account.AccountDB) string {
	if accountDB == nil {
		return ""
	}
	value := accountDB.GetData(common.HexToAddress(address), common.HexToHash(key).Bytes())
	if value == nil {
		value = common.Hash{}.Bytes()
	}
	return common.ToHex(value)
}

func GetCode(address string, accountDB *account.AccountDB) string {
	if accountDB == nil {
		return ""
	}
	value := accountDB.GetCode(common.HexToAddress(address))
	return base64.StdEncoding.EncodeToString(value)
}

func ChangeAssets(source string, targets map[string]types.TransferData, accountdb *account.AccountDB) (string, bool) {
	sourceAddr := common.HexToAddress(source)

	responseBalance := ""
	responseCoin := types.NewJSONObject()
	responseFT := types.NewJSONObject()

	for address, transferData := range targets {
		targetAddr := common.HexToAddress(address)

		// 转钱
		ok, leftBalance := transferBalance(transferData.Balance, sourceAddr, targetAddr, accountdb)
		if !ok {
			logger.Debugf("Transfer balance failed!")
			return "Transfer Balance Failed", false
		} else {
			logger.Debugf("%s to %s, target: %s", source, address, utility.BigIntToStr(accountdb.GetBalance(targetAddr)))
			responseBalance = utility.BigIntToStr(leftBalance)
		}
	}

	response := types.NewJSONObject()
	if responseBalance != "" {
		response.Put("balance", responseBalance)
	}
	if !responseCoin.IsEmpty() {
		response.Put("coin", responseCoin.GetData())
	}
	if !responseFT.IsEmpty() {
		response.Put("ft", responseFT.GetData())
	}

	return response.TOJSONString(), true
}

func transferBalance(value string, source common.Address, target common.Address, accountDB *account.AccountDB) (bool, *big.Int) {
	balance, err := utility.StrToBigInt(value)
	if err != nil {
		return false, nil
	}

	if balance.Sign() == -1 {
		return false, nil
	}

	sourceBalance := accountDB.GetBalance(source)

	if sourceBalance.Cmp(balance) == -1 {
		logger.Debugf("transfer bnt:bnt not enough!")
		return false, nil
	}

	accountDB.AddBalance(target, balance)
	left := accountDB.SubBalance(source, balance)

	return true, left
}
