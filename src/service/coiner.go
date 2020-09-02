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
	"encoding/json"
	"fmt"
	"strings"
)

//主链币充值确认
func CoinDeposit(accountdb *account.AccountDB, transaction *types.Transaction) (bool, string) {
	txLogger.Tracef("Execute coin deposit ack tx:%s", transaction.ToTxJson().ToString())
	if transaction.Data == "" {
		return false, fmt.Sprintf("data error, data: %s", transaction.Data)
	}
	var depositCoinData types.DepositCoinData
	err := json.Unmarshal([]byte(transaction.Data), &depositCoinData)
	if err != nil {
		txLogger.Errorf("Deposit coin data unmarshal error:%s", err.Error())
		return false, fmt.Sprintf("data error, data: %s", transaction.Data)
	}
	txLogger.Tracef("deposit coin data: %v,target address:%s", depositCoinData, transaction.Source)
	if depositCoinData.Amount == "" || depositCoinData.ChainType == "" {
		return false, fmt.Sprintf("data error, data: %s", transaction.Data)
	}

	if !IsRateExisted(depositCoinData.ChainType, accountdb) {
		msg := fmt.Sprintf("chainType data error, data: %s", depositCoinData.ChainType)
		txLogger.Errorf(msg)
		return false, msg
	}

	value, _ := utility.StrToBigInt(depositCoinData.Amount)
	result := accountdb.AddFT(common.HexToAddress(transaction.Source), fmt.Sprintf("official-%s", depositCoinData.ChainType), value)
	if result {
		return result, fmt.Sprintf("coin: %s, deposit %s", fmt.Sprintf("official-%s", depositCoinData.ChainType), value)
	}
	return result, fmt.Sprintf("too much value %s", value)

}

//FT充值确认
func FTDeposit(accountdb *account.AccountDB, transaction *types.Transaction) (bool, string) {
	txLogger.Tracef("Execute ft deposit ack tx:%s", transaction.ToTxJson().ToString())
	if transaction.Data == "" {
		return false, fmt.Sprintf("data error, data: %s", transaction.Data)
	}
	var depositFTData types.DepositFTData
	err := json.Unmarshal([]byte(transaction.Data), &depositFTData)
	if err != nil {
		txLogger.Errorf("Deposit ft data unmarshal error:%s", err.Error())
		return false, fmt.Sprintf("data error, data: %s", transaction.Data)
	}
	txLogger.Tracef("deposit ft data:%v, address:%s", depositFTData, transaction.Source)
	if depositFTData.Amount == "" || depositFTData.FTId == "" {
		return false, fmt.Sprintf("data error, data: %s", transaction.Data)
	}

	if !IsRateExisted(depositFTData.FTId, accountdb) {
		msg := fmt.Sprintf("depositFTData data error, data: %s", depositFTData.FTId)
		txLogger.Errorf(msg)
		return false, msg
	}

	//todo 先不检查此ft是否存在
	value, _ := utility.StrToBigInt(depositFTData.Amount)
	result := false

	// ERC20的max，特殊处理
	if 0 == strings.Compare(strings.ToLower(depositFTData.FTId), "max") {
		accountdb.AddBalance(common.HexToAddress(transaction.Source), value)
		result = true
	} else {
		result = accountdb.AddFT(common.HexToAddress(transaction.Source), depositFTData.FTId, value)
	}

	if result {
		return result, fmt.Sprintf("coin: %s, deposit %s", depositFTData.FTId, value)
	}
	return result, fmt.Sprintf("too much value %s", value)
}

//NFT充值确认
func NFTDeposit(accountdb *account.AccountDB, transaction *types.Transaction) (bool, string) {
	txLogger.Tracef("Execute nft deposit ack tx:%s", transaction.ToTxJson().ToString())
	if transaction.Data == "" {
		return false, fmt.Sprintf("data error, data: %s", transaction.Data)
	}
	var depositNFTData types.DepositNFTData
	err := json.Unmarshal([]byte(transaction.Data), &depositNFTData)
	if err != nil {
		txLogger.Errorf("Deposit nft data unmarshal error:%s", err.Error())
		return false, fmt.Sprintf("data error, data: %s", transaction.Data)
	}
	//todo 这里需要重写
	txLogger.Tracef("deposit nft data:%v,target address:%s", depositNFTData, transaction.Source)
	if depositNFTData.SetId == "" || depositNFTData.ID == "" {
		return false, fmt.Sprintf("data error, data: %s", transaction.Data)
	}

	// 检查setId
	nftSet := NFTManagerInstance.GetNFTSet(depositNFTData.SetId, accountdb)
	if nil == nftSet {
		nftSet = NFTManagerInstance.GenerateNFTSet(depositNFTData.SetId, depositNFTData.Name, depositNFTData.Symbol, depositNFTData.Creator, depositNFTData.Owner, 0, depositNFTData.CreateTime)
		NFTManagerInstance.PublishNFTSet(nftSet, accountdb)
	}

	appId := transaction.Target
	str, ok := NFTManagerInstance.GenerateNFT(nftSet, appId, depositNFTData.SetId, depositNFTData.ID, "", depositNFTData.Creator, depositNFTData.CreateTime, "official", common.HexToAddress(transaction.Source), depositNFTData.Data, accountdb)
	msg := fmt.Sprintf("depositNFT result: %s, %t", str, ok)
	txLogger.Debugf(msg)
	return ok, msg
}
