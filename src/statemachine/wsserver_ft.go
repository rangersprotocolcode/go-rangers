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
	"fmt"
)

func (self *wsServer) checkTx(appId string) (*service.TxContext, bool) {
	if 0 == len(appId) {
		return nil, false
	}

	context := service.TxManagerInstance.GetContext(appId)
	// 不在事务里，不应该啊
	if nil == context {
		common.DefaultLogger.Debugf("transferFT is nil!")
		return nil, false
	}

	tx := context.Tx
	if nil == tx || tx.Target != appId {
		msg := fmt.Sprintf("wrong appId %s", appId)
		common.DefaultLogger.Debugf(msg)
		return context, false
	}

	return context, true
}

// 状态机转主链币给玩家
func (self *wsServer) transferBNT(data map[string]string) (string, bool) {
	authCode := data["authCode"]
	appId := data["appId"]
	target := data["target"]
	chainType := data["chainType"]
	balance := data["balance"]

	return self.transferFTOrCoin(authCode, appId, target, chainType, balance, true)
}

// todo: 经济模型，转币的费用问题
// 状态机转币给玩家
func (self *wsServer) transferFT(data map[string]string) (string, bool) {
	authCode := data["authCode"]
	appId := data["appId"]
	ftId := data["ftId"]
	target := data["target"]
	supply := data["supply"]

	self.logger.Debugf("Transfer FT appId:%s,target:%s,ftId:%s,supply:%s", appId, target, ftId, supply)
	return self.transferFTOrCoin(authCode, appId, target, ftId, supply, false)
}

func (self *wsServer) transferFTOrCoin(authCode, appId, target, ftId, supply string, isBNT bool) (string, bool) {
	if 0 == len(appId) || 0 == len(authCode) || !STMManger.ValidateAppId(appId, authCode) {
		return "wrong params", false
	}

	context, ok := self.checkTx(appId)
	if !ok {
		msg := fmt.Sprintf("wrong appId %s or not in transaction", appId)
		self.logger.Debugf(msg)
		return msg, false
	}

	var (
		result string
		flag   bool
	)
	if isBNT {
		result, _, flag = service.FTManagerInstance.TransferBNT(appId, ftId, target, supply, context.AccountDB)
	} else {
		result, _, flag = service.FTManagerInstance.TransferFT(appId, ftId, target, supply, context.AccountDB)
	}

	self.logger.Debugf("Transfer FTOrCoin result:%t,message:%s", flag, result)
	if flag {
		// 生成交易，上链 context.Tx.SubTransactions
		data := types.UserData{}
		data.Address = "TransferFT"
		data.Assets = make(map[string]string, 0)
		data.Assets["gameId"] = appId
		data.Assets["target"] = target
		data.Assets["symbol"] = ftId
		data.Assets["supply"] = supply

		// 生成交易，上链
		context.Tx.AppendSubTransaction(data)

		return result, true
	}
	return result, false
}

// 发行ft
func (self *wsServer) mintFT(data map[string]string) (string, bool) {
	authCode := data["authCode"]
	appId := data["appId"]
	ftId := data["ftId"]
	target := data["target"]
	balance := data["balance"]

	self.logger.Debugf("mintFT start. authCode: %s, appId: %s, ftId: %s, target: %s, balance: %s", authCode, appId, ftId, target, balance)

	if 0 == len(appId) || 0 == len(authCode) || !STMManger.ValidateAppId(appId, authCode) {
		self.logger.Errorf("appId/authCode wrong")
		return "wrong params", false
	}

	context, ok := self.checkTx(appId)
	if !ok {
		msg := fmt.Sprintf("wrong appId %s or not in transaction", appId)
		self.logger.Errorf(msg)
		return msg, false
	}

	result, flag := service.FTManagerInstance.MintFT(appId, ftId, target, balance, context.AccountDB)
	self.logger.Debugf("mintFT end. result: %s, flag: %t", result, flag)

	if flag {
		// 生成交易，上链 context.Tx.SubTransactions
		data := types.UserData{}
		data.Address = "MintFT"
		data.Assets = make(map[string]string, 0)
		data.Assets["appId"] = appId
		data.Assets["target"] = target
		data.Assets["ftId"] = ftId
		data.Assets["balance"] = balance

		// 生成交易，上链
		context.Tx.AppendSubTransaction(data)

		return result, true
	}

	return result, false
}

// todo: 经济模型，发币的费用问题
// 状态机发币
func (self *wsServer) publishFTSet(data map[string]string) (string, bool) {
	authCode := data["authCode"]
	appId := data["appId"]
	owner := data["owner"]
	name := data["name"]
	symbol := data["symbol"]
	totalSupply := data["totalSupply"]
	createTime := data["createTime"]

	if 0 == len(appId) || 0 == len(authCode) || !STMManger.ValidateAppId(appId, authCode) {
		return "wrong params", false
	}

	context := service.TxManagerInstance.GetContext(appId)

	// 不在事务里，不应该啊
	if nil == context {
		self.logger.Debugf("not in transaction")
		return "not in transaction", false
	}

	result, flag := service.FTManagerInstance.PublishFTSet(service.FTManagerInstance.GenerateFTSet(name, symbol, appId, totalSupply, owner, createTime, 1), context.AccountDB)
	if flag {
		data := types.UserData{}
		data.Address = "StartFT"
		data.Assets = make(map[string]string, 0)
		data.Assets["gameId"] = appId
		data.Assets["name"] = name
		data.Assets["symbol"] = symbol
		data.Assets["totalSupply"] = totalSupply
		data.Assets["owner"] = owner
		data.Assets["createTime"] = createTime
		// 生成交易，上链
		context.Tx.AppendSubTransaction(data)

		return result, true
	}

	return result, false
}
