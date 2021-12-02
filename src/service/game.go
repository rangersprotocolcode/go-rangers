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

package service

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"strconv"
)

func GetBalance(source common.Address, accountDB *account.AccountDB) string {
	if accountDB == nil {
		return ""
	}
	balance := accountDB.GetBalance(source)

	return utility.BigIntToStr(balance)
}

func GetCoinBalance(source common.Address, bnt string) string {
	accountDB := AccountDBManagerInstance.GetAccountDB("", true)
	balance := accountDB.GetBNT(source, bnt)

	return utility.BigIntToStr(balance)
}

func GetAllCoinInfo(source common.Address) string {
	accountDB := AccountDBManagerInstance.GetAccountDB("", true)
	ftMap := accountDB.GetAllBNT(source)
	data := make(map[string]string, 0)
	for key, value := range ftMap {
		data[key] = utility.BigIntToStr(value)
	}
	bytes, _ := json.Marshal(data)
	return string(bytes)
}

func GetNetWorkId() string {
	return common.NetworkId()
}

func GetChainId() string {
	return common.ChainId()
}

func GetNonce(address common.Address, accountDB *account.AccountDB) string {
	if accountDB == nil {
		return ""
	}
	nonce := accountDB.GetNonce(address)
	return strconv.FormatUint(nonce, 10)
}

func GetReceipt(txHash common.Hash) string {
	transaction := GetTransactionPool().GetExecuted(txHash)
	if transaction == nil {
		return ""
	}
	result, _ := json.Marshal(transaction.Receipt)
	return string(result)
}

//获取合约的存储数据
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

// false 表示转账失败
// 这里的转账包括货币、FT、NFT
// 这里不处理事务。调用本方法之前自行处理事务
func ChangeAssets(source string, targets map[string]types.TransferData, accountdb *account.AccountDB) (string, bool) {
	sourceAddr := common.HexToAddress(source)

	responseBalance := ""
	responseCoin := types.NewJSONObject()
	responseFT := types.NewJSONObject()
	responseNFT := make([]types.NFTID, 0)

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
	if 0 != len(responseNFT) {
		response.Put("nft", responseNFT)
	}
	return response.TOJSONString(), true
}

func transferBalance(value string, source common.Address, target common.Address, accountDB *account.AccountDB) (bool, *big.Int) {
	balance, err := utility.StrToBigInt(value)
	if err != nil {
		return false, nil
	}
	// 不能扣钱
	if balance.Sign() == -1 {
		return false, nil
	}

	sourceBalance := accountDB.GetBalance(source)

	// 钱不够转账，再见
	if sourceBalance.Cmp(balance) == -1 {
		logger.Debugf("transfer bnt:bnt not enough!")
		return false, nil
	}

	// 目标加钱
	accountDB.AddBalance(target, balance)

	// 自己减钱
	left := accountDB.SubBalance(source, balance)

	return true, left
}
