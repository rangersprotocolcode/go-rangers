package statemachine

import (
	"fmt"
	"x/src/middleware/types"
	"x/src/common"
	"x/src/service"
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

	return self.transferFTOrCoin(authCode, appId, target, fmt.Sprintf("official-%s", chainType), balance)
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
	return self.transferFTOrCoin(authCode, appId, target, ftId, supply)
}

func (self *wsServer) transferFTOrCoin(authCode, appId, target, ftId, supply string) (string, bool) {
	if 0 == len(appId) || 0 == len(authCode) || !STMManger.ValidateAppId(appId, authCode) {
		return "wrong params", false
	}

	context, ok := self.checkTx(appId)
	if !ok {
		msg := fmt.Sprintf("wrong appId %s or not in transaction", appId)
		self.logger.Debugf(msg)
		return msg, false
	}

	result, _, flag := service.FTManagerInstance.TransferFT(appId, ftId, target, supply, context.AccountDB)
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
