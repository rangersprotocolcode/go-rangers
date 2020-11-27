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

package statemachine

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"encoding/json"
	"fmt"
	"strconv"
)

// 更新NFT
func (self *wsServer) updateNFT(params map[string]string) (string, bool) {
	authCode := params["authCode"]
	appId := params["appId"]
	setId := params["setId"]
	id := params["id"]
	data := params["data"]
	property := params["property"]

	self.logger.Debugf("Update NFT! appId:%s,setId:%s,id:%s,data:%s", appId, setId, id, data)

	if 0 == len(appId) || 0 == len(authCode) || !STMManger.ValidateAppId(appId, authCode) {
		return "param error", false
	}

	context, ok := self.checkTx(appId)
	if !ok {
		msg := fmt.Sprintf("wrong appId %s or not in transaction", appId)
		self.logger.Debugf(msg)
		return msg, false
	}

	accountDB := context.AccountDB
	addr := service.NFTManagerInstance.GetNFTOwner(setId, id, accountDB)
	if nil == addr {
		msg := fmt.Sprintf("wrong setId %s or id %s", setId, id)
		self.logger.Debugf(msg)
		return msg, false
	}

	if service.NFTManagerInstance.UpdateNFT(appId, setId, id, data, property, accountDB) {
		// 生成交易，上链 context.Tx.SubTransactions
		userData := types.UserData{}
		userData.Address = "UpdateNFT"
		userData.Assets = make(map[string]string, 0)
		userData.Assets["appId"] = appId
		userData.Assets["setId"] = setId
		userData.Assets["id"] = id
		userData.Assets["data"] = data
		userData.Assets["addr"] = addr.String()

		// 生成交易，上链
		context.Tx.AppendSubTransaction(userData)

		return "success update nft", true
	} else {
		msg := fmt.Sprintf("fail to update setId %s or id %s", setId, id)
		self.logger.Debugf(msg)
		return msg, false
	}
}

// 批量更新NFT
func (self *wsServer) batchUpdateNFT(params map[string]string) (string, bool) {
	authCode := params["authCode"]
	appId := params["appId"]
	setId := params["setId"]
	idString := params["id"]
	dataString := params["data"]

	var id, data []string
	json.Unmarshal([]byte(idString), &id)
	json.Unmarshal([]byte(dataString), &data)

	if len(id) != len(data) {
		msg := fmt.Sprintf("fail to BatchUpdateNFT setId %s", setId)
		self.logger.Debugf(msg)
		return msg, false
	}

	for i := range id {
		nftData := make(map[string]string)
		nftData["authCode"] = authCode
		nftData["appId"] = appId
		nftData["setId"] = setId
		nftData["id"] = id[i]
		nftData["data"] = data[i]
		_, ok := self.updateNFT(nftData)
		if !ok {
			msg := fmt.Sprintf("fail to BatchUpdateNFT setId %s", setId)
			self.logger.Debugf(msg)
			return msg, false
		}
	}

	return "success BatchUpdateNFT nft", true
}

// 将状态机持有的NFT转给指定地址
func (self *wsServer) transferNFT(params map[string]string) (string, bool) {
	authCode := params["authCode"]
	appId := params["appId"]
	setId := params["setId"]
	id := params["id"]
	target := params["target"]
	if 0 == len(appId) || 0 == len(authCode) || !STMManger.ValidateAppId(appId, authCode) {
		return "param error", false
	}

	context, ok := self.checkTx(appId)
	if !ok {
		msg := fmt.Sprintf("wrong appId %s or not in transaction", appId)
		self.logger.Debugf(msg)
		return msg, false
	}

	accountDB := context.AccountDB
	var reason string
	reason, ok = service.NFTManagerInstance.Transfer(setId, id, common.HexToAddress(appId), common.HexToAddress(target), accountDB)
	if ok {
		// 生成交易，上链 context.Tx.SubTransactions
		userData := types.UserData{}
		userData.Address = "TransferNFT"
		userData.Assets = make(map[string]string, 0)
		userData.Assets["appId"] = appId
		userData.Assets["setId"] = setId
		userData.Assets["id"] = id
		userData.Assets["target"] = target

		// 生成交易，上链
		context.Tx.AppendSubTransaction(userData)

		return "success update nft", true
	} else {
		msg := fmt.Sprintf("fail to TransferNFT setId %s or id %s from %s to %s:%s", setId, id, appId, target, reason)
		self.logger.Debugf(msg)
		return reason, false
	}
}

// 将状态机持有的NFT的使用权授予某地址
func (self *wsServer) approveNFT(params map[string]string) (string, bool) {
	authCode := params["authCode"]
	appId := params["appId"]
	setId := params["setId"]
	id := params["id"]
	target := params["target"]
	if 0 == len(appId) || 0 == len(authCode) || !STMManger.ValidateAppId(appId, authCode) {
		return "param error", false
	}

	context, ok := self.checkTx(appId)
	if !ok {
		msg := fmt.Sprintf("wrong appId %s or not in transaction", appId)
		self.logger.Debugf(msg)
		return msg, false
	}

	accountDB := context.AccountDB
	if accountDB.ApproveNFT(common.HexToAddress(appId), appId, setId, id, target) {
		// 生成交易，上链 context.Tx.SubTransactions
		userData := types.UserData{}
		userData.Address = "ApproveNFT"
		userData.Assets = make(map[string]string, 0)
		userData.Assets["appId"] = appId
		userData.Assets["setId"] = setId
		userData.Assets["id"] = id
		userData.Assets["target"] = target

		// 生成交易，上链
		context.Tx.AppendSubTransaction(userData)

		return "success approve nft", true
	} else {
		msg := fmt.Sprintf("fail to ApproveNFT setId %s or id %s from %s to %s", setId, id, appId, target)
		self.logger.Debugf(msg)
		return msg, false
	}
}

// 将状态机持有的NFT的使用权回收
func (self *wsServer) revokeNFT(params map[string]string) (string, bool) {
	authCode := params["authCode"]
	appId := params["appId"]
	if 0 == len(appId) || 0 == len(authCode) || !STMManger.ValidateAppId(appId, authCode) {
		return "param error", false
	}

	params["target"] = appId
	return self.approveNFT(params)
}

// 锁定游戏持有的nft
func (self *wsServer) lockNFT(params map[string]string) (string, bool) {
	authCode := params["authCode"]
	appId := params["appId"]
	setId := params["setId"]
	id := params["id"]
	if 0 == len(appId) || 0 == len(authCode) || !STMManger.ValidateAppId(appId, authCode) {
		return "param error", false
	}

	return self.changeNFTStatus(appId, setId, id, 1)
}

// 解锁游戏持有的nft
func (self *wsServer) unLockNFT(params map[string]string) (string, bool) {
	authCode := params["authCode"]
	appId := params["appId"]
	setId := params["setId"]
	id := params["id"]
	if 0 == len(appId) || 0 == len(authCode) || !STMManger.ValidateAppId(appId, authCode) {
		return "param error", false
	}

	return self.changeNFTStatus(appId, setId, id, 0)
}

func (self *wsServer) changeNFTStatus(appId, setId, id string, status int) (string, bool) {
	context, ok := self.checkTx(appId)
	if !ok {
		msg := fmt.Sprintf("wrong appId %s or not in transaction", appId)
		self.logger.Debugf(msg)
		return msg, false
	}

	accountDB := context.AccountDB
	if accountDB.ChangeNFTStatus(common.HexToAddress(appId), appId, setId, id, 1) {
		// 生成交易，上链 context.Tx.SubTransactions
		userData := types.UserData{}
		userData.Address = "changeNFTStatus"
		userData.Assets = make(map[string]string, 0)
		userData.Assets["appId"] = appId
		userData.Assets["setId"] = setId
		userData.Assets["id"] = id
		userData.Assets["status"] = strconv.Itoa(status)

		// 生成交易，上链
		context.Tx.AppendSubTransaction(userData)
		if status == 0 {
			return "success UnLockNFT nft", true
		}
		return "success LockNFT nft", true
	} else {
		msg := ""
		if status == 0 {
			msg = fmt.Sprintf("fail to UnLockNFT setId %s or id %s appId %s", setId, id, appId)
		} else {
			msg = fmt.Sprintf("fail to LockNFT setId %s or id %s appId %s", setId, id, appId)
		}
		self.logger.Debugf(msg)
		return msg, false
	}
}

// 发行NFTSet
func (self *wsServer) publishNFTSet(params map[string]string) (string, bool) {
	authCode := params["authCode"]
	appId := params["appId"]
	setId := params["setId"]
	name := params["name"]
	symbol := params["symbol"]
	maxSupply := params["maxSupply"]
	createTime := params["createTime"]

	if 0 == len(appId) || 0 == len(authCode) || !STMManger.ValidateAppId(appId, authCode) {
		return "param error", false
	}

	context, ok := self.checkTx(appId)
	if !ok {
		msg := fmt.Sprintf("wrong appId %s or not in transaction", appId)
		self.logger.Debugf(msg)
		return msg, false
	}

	accountDB := context.AccountDB
	value, _ := strconv.ParseUint(maxSupply, 10, 64)
	nftSet := service.NFTManagerInstance.GenerateNFTSet(setId, name, symbol, appId, appId, types.NFTConditions{}, value, createTime)
	if reason, ok := service.NFTManagerInstance.PublishNFTSet(nftSet, accountDB); ok {
		// 生成交易，上链 context.Tx.SubTransactions
		userData := types.UserData{}
		userData.Address = "PublishNFTSet"
		userData.Assets = make(map[string]string, 0)
		userData.Assets["setId"] = setId
		userData.Assets["name"] = name
		userData.Assets["symbol"] = symbol
		userData.Assets["maxSupply"] = maxSupply
		userData.Assets["appId"] = appId
		userData.Assets["createTime"] = createTime

		// 生成交易，上链
		context.Tx.AppendSubTransaction(userData)
		return "success PublishNFTSet", true
	} else {
		msg := fmt.Sprintf("fail to PublishNFTSet setId %s  appId %s :%s", setId, appId, reason)
		self.logger.Debugf(msg)
		return msg, false
	}
}

// NFT铸币
func (self *wsServer) mintNFT(params map[string]string) (string, bool) {
	authCode := params["authCode"]
	source := params["source"]
	appId := params["appId"]
	setId := params["setId"]
	target := params["target"]
	id := params["id"]
	data := params["data"]
	createTime := params["createTime"]

	self.logger.Debugf("Mint nft!appId:%s,setId:%s,id:%s,target:%s,data:%s,createTime:%s", appId, setId, id, target, data, createTime)
	if 0 == len(appId) || 0 == len(authCode) || !STMManger.ValidateAppId(appId, authCode) {
		return "param error", false
	}

	context, ok := self.checkTx(appId)
	if !ok {
		msg := fmt.Sprintf("wrong appId %s or not in transaction", appId)
		self.logger.Debugf(msg)
		return msg, false
	}

	accountDB := context.AccountDB
	if reason, ok := service.NFTManagerInstance.MintNFT(source, appId, setId, id, data, createTime, common.HexToAddress(target), accountDB); ok {
		// 生成交易，上链 context.Tx.SubTransactions
		userData := types.UserData{}
		userData.Address = "MintNFT"
		userData.Assets = make(map[string]string, 0)
		userData.Assets["appId"] = appId
		userData.Assets["setId"] = setId
		userData.Assets["id"] = id
		userData.Assets["target"] = target
		userData.Assets["data"] = data
		userData.Assets["createTime"] = createTime

		// 生成交易，上链
		context.Tx.AppendSubTransaction(userData)

		return "success MintNFT", true
	} else {
		msg := fmt.Sprintf("fail to MintNFT setId %s id %s appId %s:%s", setId, id, appId, reason)
		self.logger.Debugf(msg)
		return msg, false
	}
}
