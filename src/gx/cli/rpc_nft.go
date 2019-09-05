package cli

import (
	"x/src/core"
	"fmt"
	"x/src/common"
	"x/src/middleware/types"
	"encoding/json"
)

func (api *GtasAPI) UpdateNFT(appId, setId, id, data string) (*Result, error) {
	context, tx, ok := api.checkTx(appId)
	if !ok {
		msg := fmt.Sprintf("wrong appId %s or not in transaction", appId)
		common.DefaultLogger.Debugf(msg)
		return failResult(msg)
	}

	accountDB := context.AccountDB
	addr := core.NFTManagerInstance.GetNFTOwner(setId, id, accountDB)
	if nil == addr {
		msg := fmt.Sprintf("wrong setId %s or id %s", setId, id)
		common.DefaultLogger.Debugf(msg)
		return failResult(msg)
	}

	if core.NFTManagerInstance.UpdateNFT(*addr, appId, setId, id, data, accountDB) {
		// 生成交易，上链 context.Tx.SubTransactions
		dataList := make([]types.UserData, 0)
		userData := types.UserData{}
		userData.Address = "UpdateNFT"
		userData.Assets = make(map[string]string, 0)
		userData.Assets["appId"] = appId
		userData.Assets["setId"] = setId
		userData.Assets["id"] = id
		userData.Assets["data"] = data
		userData.Assets["addr"] = addr.String()

		dataList = append(dataList, userData)
		rawJson, _ := json.Marshal(dataList)

		// 生成交易，上链
		context.Tx.SubTransactions = append(tx.SubTransactions, string(rawJson))
		return successResult("success update nft")
	} else {
		msg := fmt.Sprintf("fail to update setId %s or id %s", setId, id)
		common.DefaultLogger.Debugf(msg)
		return failResult(msg)
	}
}

func (api *GtasAPI) BatchUpdateNFT(appId, setId string, id, data []string) (*Result, error) {
	if len(id) != len(data) {
		msg := fmt.Sprintf("fail to BatchUpdateNFT setId %s", setId)
		common.DefaultLogger.Debugf(msg)
		return failResult(msg)
	}

	for i := range id {
		_, err := api.UpdateNFT(appId, setId, id[i], data[i])
		if nil!=err{
			msg := fmt.Sprintf("fail to BatchUpdateNFT setId %s", setId)
			common.DefaultLogger.Debugf(msg)
			return failResult(msg)
		}
	}

	return successResult("success BatchUpdateNFT nft")
}

// 将状态机持有的NFT转给指定地址
func (api *GtasAPI) TransferNFT(appId, setId, id, target string) (*Result, error) {
	context, tx, ok := api.checkTx(appId)
	if !ok {
		msg := fmt.Sprintf("wrong appId %s or not in transaction", appId)
		common.DefaultLogger.Debugf(msg)
		return failResult(msg)
	}

	accountDB := context.AccountDB
	_, ok = core.NFTManagerInstance.Transfer(appId, setId, id, common.HexToAddress(appId), common.HexToAddress(target), accountDB)
	if ok {
		// 生成交易，上链 context.Tx.SubTransactions
		dataList := make([]types.UserData, 0)
		userData := types.UserData{}
		userData.Address = "TransferNFT"
		userData.Assets = make(map[string]string, 0)
		userData.Assets["appId"] = appId
		userData.Assets["setId"] = setId
		userData.Assets["id"] = id
		userData.Assets["target"] = target

		dataList = append(dataList, userData)
		rawJson, _ := json.Marshal(dataList)

		// 生成交易，上链
		context.Tx.SubTransactions = append(tx.SubTransactions, string(rawJson))
		return successResult("success update nft")
	} else {
		msg := fmt.Sprintf("fail to TransferNFT setId %s or id %s from %s to %s", setId, id, appId, target)
		common.DefaultLogger.Debugf(msg)
		return failResult(msg)
	}
}

// 将状态机持有的NFT的使用权授予某地址
func (api *GtasAPI) ApproveNFT(appId, setId, id, target string) (*Result, error) {
	context, tx, ok := api.checkTx(appId)
	if !ok {
		msg := fmt.Sprintf("wrong appId %s or not in transaction", appId)
		common.DefaultLogger.Debugf(msg)
		return failResult(msg)
	}

	accountDB := context.AccountDB
	if accountDB.ApproveNFT(common.HexToAddress(appId), appId, setId, id, target) {
		// 生成交易，上链 context.Tx.SubTransactions
		dataList := make([]types.UserData, 0)
		userData := types.UserData{}
		userData.Address = "ApproveNFT"
		userData.Assets = make(map[string]string, 0)
		userData.Assets["appId"] = appId
		userData.Assets["setId"] = setId
		userData.Assets["id"] = id
		userData.Assets["target"] = target

		dataList = append(dataList, userData)
		rawJson, _ := json.Marshal(dataList)

		// 生成交易，上链
		context.Tx.SubTransactions = append(tx.SubTransactions, string(rawJson))
		return successResult("success approve nft")
	} else {
		msg := fmt.Sprintf("fail to ApproveNFT setId %s or id %s from %s to %s", setId, id, appId, target)
		common.DefaultLogger.Debugf(msg)
		return failResult(msg)
	}
}

// 将状态机持有的NFT的使用权回收
func (api *GtasAPI) RevokeNFT(appId, setId, id string) (*Result, error) {
	return api.ApproveNFT(appId, setId, id, appId)
}
