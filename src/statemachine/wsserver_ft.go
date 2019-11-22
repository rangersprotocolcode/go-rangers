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
