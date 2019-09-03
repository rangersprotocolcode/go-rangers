package cli

import (
	"x/src/core"
	"x/src/common"
	"x/src/middleware/types"
	"encoding/json"
)

// todo: 经济模型，发币的费用问题
// 状态机发币
func (api *GtasAPI) PublishFT(gameId string, name string, symbol string, totalSupply string) (*Result, error) {
	if 0 == len(gameId) {
		return failResult("wrong params")
	}

	context := core.TxManagerInstance.GetContext(gameId)

	// 不在事务里，不应该啊
	if nil == context {
		common.DefaultLogger.Debugf("startFT is nil!")
		return failResult("not in transaction")
	}

	result, flag := core.FTManagerInstance.PublishFTSet(name, symbol, gameId, totalSupply, 1, context.AccountDB)
	if flag {
		dataList := make([]types.UserData, 0)
		data := types.UserData{}
		data.Address = "StartFT"
		data.Assets = make(map[string]string, 0)
		data.Assets["gameId"] = gameId
		data.Assets["name"] = name
		data.Assets["symbol"] = symbol
		data.Assets["totalSupply"] = totalSupply

		dataList = append(dataList, data)
		rawJson, _ := json.Marshal(dataList)

		// 生成交易，上链
		context.Tx.SubTransactions = append(context.Tx.SubTransactions, string(rawJson))
		return successResult(result)
	} else {
		return failResult(result)
	}

}

// todo: 经济模型，转币的费用问题
// 状态机转币给玩家
func (api *GtasAPI) TransferFT(gameId string, symbol string, target string, supply string) (*Result, error) {
	if 0 == len(gameId) {
		return failResult("wrong params")
	}

	context := core.TxManagerInstance.GetContext(gameId)
	// 不在事务里，不应该啊
	if nil == context {
		common.DefaultLogger.Debugf("startFT is nil!")
		return failResult("not in transaction")
	}

	result, flag := core.TransferFT(gameId, symbol, target, supply, context.AccountDB)
	if flag {
		// 生成交易，上链 context.Tx.SubTransactions

		dataList := make([]types.UserData, 0)
		data := types.UserData{}
		data.Address = "TransferFT"
		data.Assets = make(map[string]string, 0)
		data.Assets["gameId"] = gameId
		data.Assets["target"] = target
		data.Assets["symbol"] = symbol
		data.Assets["supply"] = supply

		dataList = append(dataList, data)
		rawJson, _ := json.Marshal(dataList)

		// 生成交易，上链
		context.Tx.SubTransactions = append(context.Tx.SubTransactions, string(rawJson))

		return successResult(result)
	} else {
		return failResult(result)
	}
}
