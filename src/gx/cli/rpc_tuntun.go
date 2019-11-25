package cli

import (
	"x/src/common"
	"x/src/middleware/types"
	"encoding/json"
	"x/src/network"
	"x/src/utility"
	"x/src/service"
)

func (api *GtasAPI) GetBNTBalance(addr, coin string) (*Result, error) {
	return successResult(service.GetCoinBalance(common.HexToAddress(addr), coin))
}

func (api *GtasAPI) GetAllCoinInfo(addr string) (*Result, error) {
	return successResult(service.GetAllCoinInfo(common.HexToAddress(addr)))
}

func (api *GtasAPI) GetFTBalance(addr, ft string) (*Result, error) {
	return successResult(service.GetFTInfo(common.HexToAddress(addr), ft))
}

func (api *GtasAPI) GetFTSet(id string) (*Result, error) {
	return successResult(service.GetFTSet(id))
}

func (api *GtasAPI) GetAllFT(addr string) (*Result, error) {
	return successResult(service.GetAllFT(common.HexToAddress(addr)))
}

func (api *GtasAPI) GetNFTCount(addr, setId, appId string) (*Result, error) {
	return successResult(service.GetNFTCount(addr, setId, appId))
}
func (api *GtasAPI) GetNFT(setId, id, appId string) (*Result, error) {
	return successResult(service.GetNFTInfo(setId, id, appId))
}

func (api *GtasAPI) GetAllNFT(addr, appId string) (*Result, error) {
	return successResult(service.GetAllNFT(common.HexToAddress(addr), appId))
}

func (api *GtasAPI) GetNFTSet(setId string) (*Result, error) {
	return successResult(service.GetNFTSet(setId))
}

func (api *GtasAPI) GetBalance(address string) (*Result, error) {
	gxLock.RLock()
	defer gxLock.RUnlock()

	accountDB := service.AccountDBManagerInstance.GetAccountDB("", true)
	balance := accountDB.GetBalance(common.HexToAddress(address))
	return successResult(utility.BigIntToStr(balance))
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
	context := service.TxManagerInstance.GetContext(appId)

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
		flag := service.UpdateAsset(user, appId, accountDb)
		if !flag {
			// 这里不应该回滚了
			return failResult("not enough balance")
		}

		// 记录下，供上链用
		context.Tx.AppendSubTransaction(user)
	}

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
