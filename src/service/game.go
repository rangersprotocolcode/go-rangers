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
	"math/big"
	"strings"
)

func GetBalance(source common.Address) string {
	logger.Debugf("Get balance before get balance.source:%s", source)
	accountDB := AccountDBManagerInstance.GetAccountDB("", true)
	logger.Debugf("Get balance after get balance.")
	balance := accountDB.GetBalance(source)

	return utility.BigIntToStr(balance)
}

func GetCoinBalance(source common.Address, ft string) string {
	ftName := fmt.Sprintf("official-%s", ft)
	logger.Debugf("Get coin balance before get balance.source:%s,ft:%s", source, ft)
	accountDB := AccountDBManagerInstance.GetAccountDB("", true)
	logger.Debugf("Get coin balance after get balance.")
	balance := accountDB.GetFT(source, ftName)

	return utility.BigIntToStr(balance)
}

func GetAllCoinInfo(source common.Address) string {
	accountDB := AccountDBManagerInstance.GetAccountDB("", true)
	ftMap := accountDB.GetAllFT(source)
	data := make(map[string]string, 0)
	for key, value := range ftMap {
		keyItems := strings.Split(key, "-")
		if "official" == keyItems[0] {
			data[keyItems[1]] = utility.BigIntToStr(value)
		}
	}
	bytes, _ := json.Marshal(data)
	return string(bytes)
}

func GetFTInfo(source common.Address, ft string) string {
	accountDB := AccountDBManagerInstance.GetAccountDB("", true)
	balance := accountDB.GetFT(source, ft)

	return utility.BigIntToStr(balance)
}

func GetAllFT(source common.Address) string {
	accountDB := AccountDBManagerInstance.GetAccountDB("", true)
	ftMap := accountDB.GetAllFT(source)
	data := make(map[string]string, 0)
	for key, value := range ftMap {
		keyItems := strings.Split(key, "-")
		if "official" != keyItems[0] {
			data[key] = utility.BigIntToStr(value)
		}
	}
	bytes, _ := json.Marshal(data)
	return string(bytes)
}

func GetNFTCount(addr, setId, appId string) int {
	accountDB := AccountDBManagerInstance.GetAccountDB(appId, true)
	nftSet := NFTManagerInstance.GetNFTSet(setId, accountDB)
	if nil == nftSet {
		return 0
	}

	count := 0
	for _, owner := range nftSet.OccupiedID {
		if owner.String() == addr {
			count++
		}
	}

	return count
}
func GetNFTInfo(setId, id, appId string) string {
	txLogger.Tracef("Get nft nfo.setId:%s,id:%s,appid:%s,", setId, id, appId)
	accountDB := AccountDBManagerInstance.GetAccountDB(appId, true)
	nft := NFTManagerInstance.GetNFT(setId, id, accountDB)
	if nil != nft {
		txLogger.Tracef("Got nft info:%s,", nft.ToJSONString())
		return nft.ToJSONString()
	}
	txLogger.Tracef("Got nil nft ")
	return ""
}

func GetAllNFT(source common.Address, appId string) string {
	accountDB := AccountDBManagerInstance.GetAccountDB(appId, true)
	nftList := NFTManagerInstance.GetNFTListByAddress(source, appId, accountDB)
	bytes, _ := json.Marshal(nftList)
	return string(bytes)
}

func GetAllNFTBySetId(source string, setId string) string {
	accountDB := AccountDBManagerInstance.GetAccountDB("", true)
	nftList := NFTManagerInstance.GetNFTListByAddress(common.HexToAddress(source), "", accountDB)

	result := make([]string, 0)

	if 0 != len(nftList) {
		for _, nft := range nftList {
			if nft.SetID == setId {
				result = append(result, nft.ToJSONString())
			}
		}
	}

	bytes, _ := json.Marshal(result)
	return string(bytes)
}

func GetNFTSet(setId string) string {
	txLogger.Tracef("Get nft set id:%s,", setId)
	accountDB := AccountDBManagerInstance.GetAccountDB("", true)
	nftSet := NFTManagerInstance.GetNFTSet(setId, accountDB)
	if nil != nftSet {
		txLogger.Tracef("Got nft set info:%s,", nftSet.ToJSONString())
		return nftSet.ToJSONString()
	}
	txLogger.Tracef("Got nil nft set:%v", nftSet)
	return ""
}

func GetFTSet(id string) string {
	accountDB := AccountDBManagerInstance.GetAccountDB("", true)
	ftSet := FTManagerInstance.GetFTSet(id, accountDB)

	response := make(map[string]string)
	if nil != ftSet {
		response["createTime"] = ftSet.CreateTime
		response["owner"] = ftSet.Owner
		response["maxSupply"] = utility.BigIntToStr(ftSet.MaxSupply)
		response["symbol"] = ftSet.Symbol
		response["name"] = ftSet.Name
		response["setId"] = ftSet.ID
		response["creator"] = ftSet.AppId
		if ftSet.TotalSupply != nil {
			response["totalSupply"] = utility.BigIntToStr(ftSet.TotalSupply)
		} else {
			response["totalSupply"] = "0"
		}

		bytes, _ := json.Marshal(response)
		return string(bytes)
	}

	return ""
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
			if !NFTManagerInstance.UpdateNFT(userAddr, appId, nft.SetId, nft.Id, nft.Data, accountDB) {
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
	responseNFT := make([]string, 0)

	for address, transferData := range targets {
		targetAddr := common.HexToAddress(address)

		// 转钱
		ok, leftBalance := transferBalance(transferData.Balance, sourceAddr, targetAddr, accountdb)
		if !ok {
			logger.Debugf("Transfer balance failed!")
			return "Transfer Balance Failed", false
		} else {
			logger.Debugf("%s to %s, target: %s", source, address, utility.BigIntToStr(accountdb.GetBalance(targetAddr)))
			responseBalance = leftBalance.String()
		}

		// 转coin
		left, ok := transferCoin(transferData.Coin, source, address, accountdb)
		if !ok {
			logger.Debugf("Transfer coin failed!")
			return "Transfer BNT Failed", false
		} else {
			responseCoin.Merge(left, types.ReplaceBigInt)
		}

		// 转FT
		ftList := transferData.FT
		if 0 != len(ftList) {
			left, ok := transferFT(ftList, source, address, accountdb)
			if !ok {
				logger.Debugf("Transfer ft failed!")
				return "Transfer FT Filed", false
			} else {
				responseFT.Merge(left, types.ReplaceBigInt)
			}
		}

		// 转NFT
		nftList, ok := transferNFT(transferData.NFT, sourceAddr, targetAddr, accountdb)
		if !ok {
			logger.Debugf("Transfer nft failed!")
			return "Transfer NFT Failed", false
		} else if 0 != len(nftList) {
			responseNFT = append(responseNFT, nftList...)
		}

	}

	response := types.NewJSONObject()

	if responseBalance != "" {
		response.Put("balance", responseBalance)
	}
	if !responseCoin.IsEmpty() {
		response.Put("coin", responseCoin.TOJSONString())
	}
	if !responseFT.IsEmpty() {
		response.Put("ft", responseFT.TOJSONString())
	}
	if 0 != len(responseNFT) {
		data, _ := json.Marshal(responseNFT)
		response.Put("nft", string(data))
	}

	return response.TOJSONString(), true
}

func transferNFT(nftIDList []types.NFTID, source common.Address, target common.Address, db *account.AccountDB) ([]string, bool) {
	length := len(nftIDList)
	if 0 == length {
		return nil, true
	}

	response := make([]string, length)
	for _, id := range nftIDList {
		_, flag := NFTManagerInstance.Transfer(id.SetId, id.Id, source, target, db)
		if !flag {
			return nil, false
		}

		idBytes, _ := json.Marshal(id)
		response = append(response, string(idBytes))
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

		response.Put(strings.TrimPrefix(ftName, "official-"), left)
	}

	return &response, true
}

func transferCoin(coin map[string]string, source string, target string, accountDB *account.AccountDB) (*types.JSONObject, bool) {
	if 0 == len(coin) {
		return nil, true
	}

	ft := make(map[string]string, len(coin))
	for key, value := range coin {
		ft[fmt.Sprintf("official-%s", key)] = value
	}

	return transferFT(ft, source, target, accountDB)
}

// tx.source : 发币方
// tx.type = 110
// tx.data 发行参数，map jsonString
// {"symbol":"","name":"","totalSupply":"12345678"}
func PublishFT(accountdb *account.AccountDB, tx *types.Transaction) (string, bool) {
	txLogger.Debugf("Execute publish ft tx:%s", tx.ToTxJson().ToString())
	var ftSet map[string]string
	if err := json.Unmarshal([]byte(tx.Data), &ftSet); nil != err {
		txLogger.Errorf("Unmarshal data error:%s", err.Error())
		return "Publish FT Bad Format", false
	}

	appId := tx.Source
	createTime := ftSet["createTime"]
	id, ok := FTManagerInstance.PublishFTSet(FTManagerInstance.GenerateFTSet(ftSet["name"], ftSet["symbol"], appId, ftSet["maxSupply"], appId, createTime, 1), accountdb)
	txLogger.Debugf("Publish ft name:%s,symbol:%s,totalSupply:%s,appId:%s,id:%s,publish result:%t", ftSet["name"], ftSet["symbol"], ftSet["totalSupply"], appId, id, ok)

	return id, ok
}

func PublishNFTSet(accountdb *account.AccountDB, tx *types.Transaction) (bool, string) {
	txLogger.Tracef("Execute publish nft tx:%s", tx.ToTxJson().ToString())

	var nftSet types.NFTSet
	if err := json.Unmarshal([]byte(tx.Data), &nftSet); nil != err {
		txLogger.Errorf("Unmarshal data error:%s", err.Error())
		return false, "Publish NFT Set Bad Format"
	}

	appId := tx.Source
	message, flag := NFTManagerInstance.PublishNFTSet(NFTManagerInstance.GenerateNFTSet(nftSet.SetID, nftSet.Name, nftSet.Symbol, appId, appId, nftSet.MaxSupply, nftSet.CreateTime), accountdb)
	return flag, message
}

func MintFT(accountdb *account.AccountDB, tx *types.Transaction) (bool, string) {
	data := make(map[string]string)
	json.Unmarshal([]byte(tx.Data), &data)

	message, result := FTManagerInstance.MintFT(tx.Source, data["ftId"], tx.Target, data["supply"], accountdb)
	return result, message
}

func ShuttleNFT(db *account.AccountDB, tx *types.Transaction) (bool, string) {
	data := make(map[string]string)
	json.Unmarshal([]byte(tx.Data), &data)

	message, ok := NFTManagerInstance.Shuttle(tx.Source, data["setId"], data["id"], data["newAppId"], db)

	return ok, message
}

func MintNFT(accountdb *account.AccountDB, tx *types.Transaction) (bool, string) {
	data := make(map[string]string)
	json.Unmarshal([]byte(tx.Data), &data)

	message, ok := NFTManagerInstance.MintNFT(tx.Source, data["setId"], data["id"], data["data"], data["createTime"], common.HexToAddress(data["target"]), accountdb)
	return ok, message
}

func UpdateNFT(accountDB *account.AccountDB, tx *types.Transaction) (bool, string) {
	params := make(map[string]string, 0)
	json.Unmarshal([]byte(tx.Data), &params)

	appId := tx.Source
	setId := params["setId"]
	id := params["id"]
	data := params["data"]

	if 0 == len(appId) || 0 == len(setId) || 0 == len(id) {
		return false, "param error"
	}

	addr := NFTManagerInstance.GetNFTOwner(setId, id, accountDB)
	if nil == addr {
		msg := fmt.Sprintf("wrong setId %s or id %s", setId, id)
		txLogger.Debugf(msg)
		return false, msg
	}

	if NFTManagerInstance.UpdateNFT(*addr, appId, setId, id, data, accountDB) {
		return true, "success update nft"
	} else {
		msg := fmt.Sprintf("fail to update setId %s or id %s", setId, id)
		txLogger.Debugf(msg)
		return false, msg
	}
}

func approveNFT(accountDB *account.AccountDB, params map[string]string, owner string) (bool, string) {
	setId := params["setId"]
	id := params["id"]
	target := params["target"]

	if 0 == len(owner) || 0 == len(target) || 0 == len(setId) || 0 == len(id) {
		return false, fmt.Sprintf("param error. owner: %s, setId: %s, id: %s, target: %s", owner, setId, id, target)
	}

	if accountDB.ApproveNFT(common.HexToAddress(owner), owner, setId, id, target) {
		return true, "success"
	} else {
		msg := fmt.Sprintf("fail to approve/revoke NFT, setId %s or id %s from %s to %s", setId, id, owner, target)
		txLogger.Debugf(msg)
		return false, msg
	}
}

func ApproveNFT(accountDB *account.AccountDB, tx *types.Transaction) (bool, string) {
	params := make(map[string]string, 0)
	json.Unmarshal([]byte(tx.Data), &params)

	return approveNFT(accountDB, params, tx.Source)
}

func RevokeNFT(accountDB *account.AccountDB, tx *types.Transaction) (bool, string) {
	params := make(map[string]string, 0)
	json.Unmarshal([]byte(tx.Data), &params)

	params["target"] = tx.Source
	return approveNFT(accountDB, params, tx.Source)
}
