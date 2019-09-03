package cli

import (
	"x/src/statemachine"
	"x/src/core"
	"x/src/common"
	"strconv"
	"x/src/middleware/types"
	"encoding/json"
	"x/src/network"
)

func (api *GtasAPI) GetGameType(gameId string) (*Result, error) {
	gameType := statemachine.Docker.GetType(gameId)
	return successResult(gameType)
}

func (api *GtasAPI) GetBalance(address string, gameId string) (*Result, error) {
	gxLock.RLock()
	defer gxLock.RUnlock()

	accountDB := core.AccountDBManagerInstance.GetAccountDB(gameId)
	balance := accountDB.GetBalance(common.HexToAddress(address))
	floatdata := float64(balance.Int64()) / 1000000000
	return successResult(strconv.FormatFloat(floatdata, 'f', -1, 64))
}

func (api *GtasAPI) GetAsset(address string, gameId string, assetId string) (*Result, error) {
	gxLock.RLock()
	defer gxLock.RUnlock()

	accountDB := core.AccountDBManagerInstance.GetAccountDB(gameId)
	nft := accountDB.GetNFTByGameId(common.HexToAddress(address), gameId, assetId)

	return successResult(nft)
}

func (api *GtasAPI) GetAllAssets(address string, gameId string) (*Result, error) {
	gxLock.RLock()
	defer gxLock.RUnlock()

	return getAssets(address, gameId)
}

func (api *GtasAPI) GetAccount(address string, gameId string) (*Result, error) {
	gxLock.RLock()
	defer gxLock.RUnlock()

	accountDB := core.AccountDBManagerInstance.GetAccountDB(gameId)
	source := common.HexToAddress(address)

	subAccountData := make(map[string]interface{})

	ftList := accountDB.GetAllFT(source)
	ftMap := make(map[string]string)
	if 0 != len(ftList) {
		for id, value := range ftList {
			ftMap[id] = strconv.FormatFloat(float64(value.Int64())/1000000000, 'f', -1, 64)
		}
	}
	subAccountData["ft"] = ftMap
	subAccountData["nft"] = accountDB.GetAllNFTByGameId(source, gameId)

	balance := accountDB.GetBalance(source)
	floatdata := float64(balance.Int64()) / 1000000000
	subAccountData["balance"] = strconv.FormatFloat(floatdata, 'f', -1, 64)

	return successResult(subAccountData)
}

func getAssets(address string, gameId string) (*Result, error) {
	accountDB := core.AccountDBManagerInstance.GetAccountDB(gameId)
	sub := accountDB.GetAllNFTByGameId(common.HexToAddress(address), gameId)
	return successResult(sub)
}

// 通过rpc的方式，让本地的docker镜像调用
func (api *GtasAPI) UpdateAssets(appId, rawjson string, nonce uint64) (*Result, error) {
	common.DefaultLogger.Debugf("UpdateAssets Rcv gameId:%s,rawJson:%s,nonce:%d\n", appId, rawjson, nonce)
	//todo 并发问题 临时加锁控制
	gxLock.Lock()
	defer gxLock.Unlock()

	data := make([]types.UserData, 0)
	if err := json.Unmarshal([]byte(rawjson), &data); err != nil {
		common.DefaultLogger.Debugf("Json unmarshal error:%s,raw:%s\n", err.Error(), rawjson)
		return failResult(err.Error())
	}

	if nil == data || 0 == len(data) {
		common.DefaultLogger.Debugf("update asset data is nil!")
		return failResult("nil data")
	}

	// 立即执行
	context := core.TxManagerInstance.GetContext(appId)

	// 不在事务里，不应该啊
	if nil == context {
		common.DefaultLogger.Debugf("update asset game context is nil!")
		return failResult("not in transaction")
	}

	// 校验appId
	tx := context.Tx
	if nil == tx || tx.Target != appId {
		common.DefaultLogger.Debugf("wrong appId: %s!", appId)
		return failResult("appId wrong")
	}

	accountDb := context.AccountDB
	for _, user := range data {
		flag := core.UpdateAsset(user, appId, accountDb)
		if !flag {
			// 这里不应该回滚了
			//accountDb.RevertToSnapshot(snapshot)
			return failResult("not enough balance")
		}
	}

	// 记录下，供上链用
	context.Tx.SubTransactions = append(tx.SubTransactions, rawjson)

	return successResult(data)
}

func (api *GtasAPI) Notify(gameId string, userid string, message string) {
	go network.GetNetInstance().Notify(true, gameId, userid, message)
}

func (api *GtasAPI) NotifyGroup(gameId string, groupId string, message string) {
	go network.GetNetInstance().Notify(false, gameId, groupId, message)
}

func (api *GtasAPI) NotifyBroadcast(gameId string, message string) {
	api.NotifyGroup(gameId, "", message)
}
