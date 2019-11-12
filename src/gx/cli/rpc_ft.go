package cli

import (
	"x/src/core"
	"x/src/common"
	"x/src/middleware/types"
	"fmt"
	"x/src/statemachine"
)

// 状态机转主链币给玩家
func (api *GtasAPI) TransferBNT(authCode, appId, target, chainType, balance string) (*Result, error) {
	return api.transferFTOrCoin(authCode, appId, target, fmt.Sprintf("official-%s", chainType), balance)
}

// todo: 经济模型，发币的费用问题
// 状态机发币
func (api *GtasAPI) PublishFT(authCode, appId, owner, name, symbol, totalSupply, createTime string) (*Result, error) {
	if 0 == len(appId) || 0 == len(authCode) || !statemachine.Docker.ValidateAppId(appId, authCode) {
		return failResult("wrong params")
	}

	context := core.TxManagerInstance.GetContext(appId)

	// 不在事务里，不应该啊
	if nil == context {
		common.DefaultLogger.Debugf("startFT is nil!")
		return failResult("not in transaction")
	}

	result, flag := core.FTManagerInstance.PublishFTSet(name, symbol, appId, totalSupply, owner, createTime, 1, context.AccountDB, true)
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

		return successResult(result)
	} else {
		return failResult(result)
	}

}

func (api *GtasAPI) MintFT(authCode, appId, ftId, target, balance string) (*Result, error) {
	api.logger.Debugf("mintFT start. authCode: %s, appId: %s, ftId: %s, target: %s, balance: %s", authCode, appId, ftId, target, balance)

	if 0 == len(appId) || 0 == len(authCode) || !statemachine.Docker.ValidateAppId(appId, authCode) {
		api.logger.Errorf("appId/authCode wrong")
		return failResult("wrong params")
	}

	context, ok := api.checkTx(appId)
	if !ok {
		msg := fmt.Sprintf("wrong appId %s or not in transaction", appId)
		api.logger.Errorf(msg)
		return failResult(msg)
	}

	result, flag := core.FTManagerInstance.MintFT(appId, ftId, target, balance, context.AccountDB)
	api.logger.Debugf("mintFT end. result: %s, flag: %t", result, flag)

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

		return successResult(result)
	} else {
		return failResult(result)
	}
}

// todo: 经济模型，转币的费用问题
// 状态机转币给玩家
func (api *GtasAPI) TransferFT(authCode, appId, target, ftId, supply string) (*Result, error) {
	common.DefaultLogger.Debugf("Transfer FT appId:%s,target:%s,ftId:%s,supply:%s", appId, target, ftId, supply)
	return api.transferFTOrCoin(authCode, appId, target, ftId, supply)
}

func (api *GtasAPI) transferFTOrCoin(authCode, appId, target, ftId, supply string) (*Result, error) {
	if 0 == len(appId) || 0 == len(authCode) || !statemachine.Docker.ValidateAppId(appId, authCode) {
		return failResult("wrong params")
	}

	context, ok := api.checkTx(appId)
	if !ok {
		msg := fmt.Sprintf("wrong appId %s or not in transaction", appId)
		common.DefaultLogger.Debugf(msg)
		return failResult(msg)
	}

	result, _, flag := core.FTManagerInstance.TransferFT(appId, ftId, target, supply, context.AccountDB)
	common.DefaultLogger.Debugf("Transfer FTOrCoin result:%t,message:%s", flag, result)
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

		return successResult(result)
	} else {
		return failResult(result)
	}
}

func (api *GtasAPI) checkTx(appId string) (*core.TxContext, bool) {
	if 0 == len(appId) {
		return nil, false
	}

	context := core.TxManagerInstance.GetContext(appId)
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
