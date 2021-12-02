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

// 状态机更新资产
// 包括货币转账、NFT资产修改
func UpdateAsset(user types.UserData, appId string, accountDB *account.AccountDB) bool {
	userAddr := common.HexToAddress(user.Address)
	appAddr := common.HexToAddress(appId)

	// 转balance
	transferBalanceOk, _ := transferBalance(user.Balance, appAddr, userAddr, accountDB)
	if !transferBalanceOk {
		logger.Debugf("Change balance failed!")
		return false
	}

	// 转coin
	_, ok := transferCoin(user.Coin, appId, user.Address, accountDB)
	if !ok {
		logger.Debugf("Change coin failed!")
		return false
	}

	// 转FT
	ftList := user.FT
	if 0 != len(ftList) {
		for ftName, valueString := range ftList {
			_, _, flag := FTManagerInstance.TransferFT(appId, ftName, user.Address, valueString, accountDB)
			if !flag {
				logger.Debugf("Game Change ft failed!")
				return false
			}
		}
	}

	// 修改NFT属性
	// 若修改不存在的NFT，则会失败
	nftList := user.NFT
	if 0 != len(nftList) {
		for _, nft := range nftList {
			if !NFTManagerInstance.UpdateNFT(appId, nft.SetId, nft.Id, nft.Data, nft.Property, accountDB) {
				return false
			}
		}
	}

	return true
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

func transferNFT(nftIDList []types.NFTID, source common.Address, target common.Address, db *account.AccountDB) ([]types.NFTID, bool) {
	length := len(nftIDList)
	if 0 == length {
		return nil, true
	}

	response := make([]types.NFTID, 0)
	for _, id := range nftIDList {
		_, flag := NFTManagerInstance.Transfer(id.SetId, id.Id, source, target, db)
		if !flag {
			return nil, false
		}

		response = append(response, id)
	}

	return response, true
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

func transferFT(ft map[string]string, source string, target string, accountDB *account.AccountDB) (*types.JSONObject, bool) {
	if 0 == len(ft) {
		return nil, true
	}
	response := types.NewJSONObject()

	for ftName, valueString := range ft {
		message, left, ok := FTManagerInstance.TransferFT(source, ftName, target, valueString, accountDB)
		if !ok {
			logger.Debugf("Transfer FT Failed:%s", message)
			return nil, false
		}

		response.Put(ftName, left)
	}

	return &response, true
}

func transferCoin(coin map[string]string, source string, target string, accountDB *account.AccountDB) (*types.JSONObject, bool) {
	if 0 == len(coin) {
		return nil, true
	}
	response := types.NewJSONObject()

	for ftName, valueString := range coin {
		message, left, ok := FTManagerInstance.TransferBNT(source, ftName, target, valueString, accountDB)
		if !ok {
			logger.Debugf("Transfer FT Failed:%s", message)
			return nil, false
		}

		response.Put(ftName, left)
	}

	return &response, true
}
