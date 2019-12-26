package statemachine

import (
	"x/src/service"
	"x/src/common"
	"x/src/utility"
	"x/src/network"
	"strconv"
	"time"
)

func (self *wsServer) getBNTBalance(params map[string]string) (string, bool) {
	addr := params["addr"]
	coin := params["coin"]

	return service.GetCoinBalance(common.HexToAddress(addr), coin), true
}

func (self *wsServer) getAllCoinInfo(params map[string]string) (string, bool) {
	addr := params["addr"]
	return service.GetAllCoinInfo(common.HexToAddress(addr)), true
}

func (self *wsServer) getFTBalance(params map[string]string) (string, bool) {
	addr := params["addr"]
	id := params["id"]
	//mock异常情况 返回超时
	if addr == "0x0b7467fe7225e8adcb6b5779d68c20fceaa58d55" {
		time.Sleep(time.Second * 3)
	}
	return service.GetFTInfo(common.HexToAddress(addr), id), true
}

func (self *wsServer) getFTSet(params map[string]string) (string, bool) {
	id := params["id"]
	answer := service.GetFTSet(id)
	return answer, 0 != len(answer)
}

func (self *wsServer) getAllFT(params map[string]string) (string, bool) {
	addr := params["addr"]
	return service.GetAllFT(common.HexToAddress(addr)), true
}

func (self *wsServer) getNFTCount(params map[string]string) (string, bool) {
	addr := params["addr"]
	setId := params["setId"]
	appId := params["appId"]

	return strconv.Itoa(service.GetNFTCount(addr, setId, appId)), true
}
func (self *wsServer) getNFT(params map[string]string) (string, bool) {
	id := params["id"]
	setId := params["setId"]
	appId := params["appId"]

	return service.GetNFTInfo(setId, id, appId), true
}

func (self *wsServer) getAllNFT(params map[string]string) (string, bool) {
	addr := params["addr"]
	appId := params["appId"]
	return service.GetAllNFT(common.HexToAddress(addr), appId), true
}

func (self *wsServer) getNFTSet(params map[string]string) (string, bool) {
	setId := params["setId"]
	return service.GetNFTSet(setId), true
}

//todo: 暂时不需要
func (self *wsServer) GetBalance(params map[string]string) (string, bool) {
	addr := params["addr"]
	accountDB := service.AccountDBManagerInstance.GetAccountDB("", true)
	balance := accountDB.GetBalance(common.HexToAddress(addr))
	return utility.BigIntToStr(balance), true
}

func (self *wsServer) notify(params map[string]string) (string, bool) {
	gameId := params["gameId"]
	userId := params["userId"]
	message := params["message"]
	go network.GetNetInstance().Notify(true, gameId, userId, message)
	return "", true
}

func (self *wsServer) notifyGroup(params map[string]string) (string, bool) {
	gameId := params["gameId"]
	groupId := params["groupId"]
	message := params["message"]

	if 0 == len(groupId) {
		return "wrong groupId", false
	}
	go network.GetNetInstance().Notify(false, gameId, groupId, message)
	return "", true
}

func (self *wsServer) notifyBroadcast(params map[string]string) (string, bool) {
	gameId := params["gameId"]
	message := params["message"]

	go network.GetNetInstance().Notify(false, gameId, "", message)
	return "", true
}
