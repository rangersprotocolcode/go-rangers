package cli

import (
	"x/src/core"
	"fmt"
	"x/src/common"
	"x/src/middleware/types"
	"encoding/json"
	"strconv"
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
		if nil != err {
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

// 锁定游戏持有的nft
func (api *GtasAPI) LockNFT(appId, setId, id string) (*Result, error) {
	return api.changeNFTStatus(appId, setId, id, 1)
}

// 解锁游戏持有的nft
func (api *GtasAPI) UnLockNFT(appId, setId, id string) (*Result, error) {
	return api.changeNFTStatus(appId, setId, id, 0)
}

func (api *GtasAPI) changeNFTStatus(appId, setId, id string, status int) (*Result, error) {
	context, tx, ok := api.checkTx(appId)
	if !ok {
		msg := fmt.Sprintf("wrong appId %s or not in transaction", appId)
		common.DefaultLogger.Debugf(msg)
		return failResult(msg)
	}

	accountDB := context.AccountDB
	if accountDB.ChangeNFTStatus(common.HexToAddress(appId), appId, setId, id, 1) {
		// 生成交易，上链 context.Tx.SubTransactions
		dataList := make([]types.UserData, 0)
		userData := types.UserData{}
		userData.Address = "changeNFTStatus"
		userData.Assets = make(map[string]string, 0)
		userData.Assets["appId"] = appId
		userData.Assets["setId"] = setId
		userData.Assets["id"] = id
		userData.Assets["status"] = strconv.Itoa(status)

		dataList = append(dataList, userData)
		rawJson, _ := json.Marshal(dataList)

		// 生成交易，上链
		context.Tx.SubTransactions = append(tx.SubTransactions, string(rawJson))
		return successResult("success LockNFT nft")
	} else {
		msg := fmt.Sprintf("fail to LockNFT setId %s or id %s appId %s", setId, id, appId)
		common.DefaultLogger.Debugf(msg)
		return failResult(msg)
	}
}

// 发行NFTSet
func (api *GtasAPI) PublishNFTSet(appId, setId, name, symbol, createTime string, maxSupply uint) (*Result, error) {
	context, tx, ok := api.checkTx(appId)
	if !ok {
		msg := fmt.Sprintf("wrong appId %s or not in transaction", appId)
		common.DefaultLogger.Debugf(msg)
		return failResult(msg)
	}

	accountDB := context.AccountDB
	if _, ok, _ := core.NFTManagerInstance.PublishNFTSet(setId, name, symbol, appId, appId, maxSupply, createTime, accountDB); ok {
		// 生成交易，上链 context.Tx.SubTransactions
		dataList := make([]types.UserData, 0)
		userData := types.UserData{}
		userData.Address = "PublishNFTSet"
		userData.Assets = make(map[string]string, 0)
		userData.Assets["setId"] = setId
		userData.Assets["name"] = name
		userData.Assets["symbol"] = symbol
		userData.Assets["maxSupply"] = strconv.FormatInt(int64(maxSupply), 10)
		userData.Assets["appId"] = appId
		userData.Assets["createTime"] = createTime

		dataList = append(dataList, userData)
		rawJson, _ := json.Marshal(dataList)

		// 生成交易，上链
		context.Tx.SubTransactions = append(tx.SubTransactions, string(rawJson))
		return successResult("success PublishNFTSet")
	} else {
		msg := fmt.Sprintf("fail to PublishNFTSet setId %s  appId %s", setId, appId)
		common.DefaultLogger.Debugf(msg)
		return failResult(msg)
	}
}

// NFT铸币
func (api *GtasAPI) MintNFT(appId, setId, id, target, data, createTime string) (*Result, error) {
	context, tx, ok := api.checkTx(appId)
	if !ok {
		msg := fmt.Sprintf("wrong appId %s or not in transaction", appId)
		common.DefaultLogger.Debugf(msg)
		return failResult(msg)
	}

	accountDB := context.AccountDB
	if _, ok := core.NFTManagerInstance.MintNFT(appId, setId, id, data, createTime, common.HexToAddress(target), accountDB); ok {
		// 生成交易，上链 context.Tx.SubTransactions
		dataList := make([]types.UserData, 0)
		userData := types.UserData{}
		userData.Address = "MintNFT"
		userData.Assets = make(map[string]string, 0)
		userData.Assets["appId"] = appId
		userData.Assets["setId"] = setId
		userData.Assets["id"] = id
		userData.Assets["target"] = target
		userData.Assets["data"] = data

		dataList = append(dataList, userData)
		rawJson, _ := json.Marshal(dataList)

		// 生成交易，上链
		context.Tx.SubTransactions = append(tx.SubTransactions, string(rawJson))
		return successResult("success MintNFT")
	} else {
		msg := fmt.Sprintf("fail to MintNFT setId %s id %s appId %s", setId, id, appId)
		common.DefaultLogger.Debugf(msg)
		return failResult(msg)
	}
}
