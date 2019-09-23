package core

import (
	"x/src/middleware/types"
	"x/src/common"
	"x/src/storage/account"
	"math/big"
	"strconv"
	"strings"
	"fmt"
	"encoding/json"
)

func GetCoinBalance(source common.Address, ft string) string {
	ftName := fmt.Sprintf("official-%s", ft)
	accountDB := AccountDBManagerInstance.GetAccountDB("", true)
	balance := accountDB.GetFT(source, ftName)
	floatdata := float64(balance.Int64()) / 1000000000
	return strconv.FormatFloat(floatdata, 'f', -1, 64)
}

func GetAllCoinInfo(source common.Address) string {
	accountDB := AccountDBManagerInstance.GetAccountDB("", true)
	ftMap := accountDB.GetAllFT(source)
	data := make(map[string]string, 0)
	for key, value := range ftMap {
		keyItems := strings.Split(key, "-")
		if "official" == keyItems[0] {
			data[keyItems[1]] = strconv.FormatFloat(float64(value.Int64())/1000000000, 'f', -1, 64)
		}
	}
	bytes, _ := json.Marshal(data)
	return string(bytes)
}

func GetFTInfo(source common.Address, ft string) string {
	accountDB := AccountDBManagerInstance.GetAccountDB("", true)
	balance := accountDB.GetFT(source, ft)
	floatData := float64(balance.Int64()) / 1000000000
	return strconv.FormatFloat(floatData, 'f', -1, 64)
}

func GetAllFT(source common.Address) string {
	accountDB := AccountDBManagerInstance.GetAccountDB("", true)
	ftMap := accountDB.GetAllFT(source)
	data := make(map[string]string, 0)
	for key, value := range ftMap {
		keyItems := strings.Split(key, "-")
		if "official" != keyItems[0] {
			data[key] = strconv.FormatFloat(float64(value.Int64())/1000000000, 'f', -1, 64)
		}
	}
	bytes, _ := json.Marshal(data)
	return string(bytes)
}

func GetNFTCount(addr, setId, appId string) int {
	accountDB := AccountDBManagerInstance.GetAccountDB(appId, true)
	nftSet := NFTManagerInstance.GetNFTSet(setId, accountDB)
	if nil == nftSet {
		return 0
	}

	count := 0
	for _, owner := range nftSet.OccupiedID {
		if owner.String() == addr {
			count++
		}
	}

	return count
}
func GetNFTInfo(setId, id, appId string) string {
	common.DefaultLogger.Debugf("Get nft nfo.setId:%s,id:%s,appid:%s,", setId, id, appId)
	accountDB := AccountDBManagerInstance.GetAccountDB(appId, true)
	nft := NFTManagerInstance.GetNFT(setId, id, accountDB)
	if nil != nft {
		common.DefaultLogger.Debugf("Got nft info:%s,", nft.ToJSONString())
		return nft.ToJSONString()
	}
	common.DefaultLogger.Debugf("Got nil nft ")
	return ""
}

func GetAllNFT(source common.Address, appId string) string {
	accountDB := AccountDBManagerInstance.GetAccountDB(appId, true)
	nftList := NFTManagerInstance.GetNFTListByAddress(source, appId, accountDB)
	bytes, _ := json.Marshal(nftList)
	return string(bytes)
}

func GetNFTSet(setId string) string {
	accountDB := AccountDBManagerInstance.GetAccountDB("", true)
	nftSet := NFTManagerInstance.GetNFTSet(setId, accountDB)
	bytes, _ := json.Marshal(nftSet)
	return string(bytes)
}

func GetFTSet(id string) string {
	accountDB := AccountDBManagerInstance.GetAccountDB("", true)
	ftSet := FTManagerInstance.GetFTSet(id, accountDB)

	bytes, _ := json.Marshal(ftSet)
	return string(bytes)
}

// 状态机更新资产
// 包括货币转账、NFT资产修改
func UpdateAsset(user types.UserData, appId string, accountDB *account.AccountDB) bool {
	userAddr := common.HexToAddress(user.Address)
	appAddr := common.HexToAddress(appId)

	// 转balance
	if !transferBalance(user.Balance, appAddr, userAddr, accountDB) {
		logger.Debugf("Change balance failed!")
		return false
	}

	// 转coin
	if !transferCoin(user.Coin, appId, user.Address, accountDB) {
		logger.Debugf("Change coin failed!")
		return false
	}

	// 转FT
	ftList := user.FT
	if 0 != len(ftList) {
		for ftName, valueString := range ftList {
			_, flag := TransferFT(appId, ftName, user.Address, valueString, accountDB)
			if !flag {
				logger.Debugf("Game Change ft failed!")
				return false
			}
		}
	}

	// 修改NFT属性
	// 若修改不存在的NFT，则会失败
	nftList := user.NFT
	if 0 != len(nftList) {
		for _, nft := range nftList {
			if !NFTManagerInstance.UpdateNFT(userAddr, appId, nft.SetId, nft.Id, nft.Data, accountDB) {
				return false
			}
		}
	}

	return true
}

func convert(s string) *big.Int {
	f, _ := strconv.ParseFloat(s, 64)
	return big.NewInt(int64(f * 1000000000))
}

func convertWithoutBase(s string) *big.Int {
	f, _ := strconv.ParseFloat(s, 64)
	return big.NewInt(int64(f))
}

// false 表示转账失败
// 这里的转账包括货币、FT、NFT
// 这里不处理事务。调用本方法之前自行处理事务
func changeAssets(gameId string, source string, targets map[string]types.TransferData, accountdb *account.AccountDB) bool {
	//sub := GetSubAccount(source, gameId, accountdb)
	sourceAddr := common.HexToAddress(source)

	for address, transferData := range targets {
		targetAddr := common.HexToAddress(address)

		// 转钱
		if !transferBalance(transferData.Balance, sourceAddr, targetAddr, accountdb) {
			logger.Debugf("Change balance failed!")
			return false
		}

		// 转coin
		if !transferCoin(transferData.Coin, source, address, accountdb) {
			logger.Debugf("Change coin failed!")
			return false
		}

		// 转FT
		ftList := transferData.FT
		if 0 != len(ftList) {
			if !transferFT(ftList, source, address, accountdb) {
				logger.Debugf("Change ft failed!")
				return false
			}
		}

		// 转NFT
		if !transferNFT(transferData.NFT, sourceAddr, targetAddr, gameId, accountdb) {
			logger.Debugf("Change nft failed!")
			return false
		}

	}

	return true
}

func transferNFT(nftIDList []types.NFTID, source common.Address, target common.Address, appId string, db *account.AccountDB) bool {
	if 0 == len(nftIDList) {
		return true
	}

	for _, id := range nftIDList {
		_, flag := NFTManagerInstance.Transfer(appId, id.SetId, id.Id, source, target, db)
		if !flag {
			return false
		}
	}

	return true
}

func transferBalance(value string, source common.Address, target common.Address, accountDB *account.AccountDB) bool {
	balance := convert(value)
	// 不能扣钱
	if balance.Sign() == -1 {
		return false
	}

	sourceBalance := accountDB.GetBalance(source)

	// 钱不够转账，再见
	if sourceBalance.Cmp(balance) == -1 {
		return false
	}

	// 目标加钱
	accountDB.AddBalance(target, balance)

	// 自己减钱
	accountDB.SubBalance(source, balance)

	return true
}

func transferFT(ft map[string]string, source string, target string, accountDB *account.AccountDB) bool {
	if 0 == len(ft) {
		return true
	}

	for ftName, valueString := range ft {
		if _, ok := TransferFT(source, ftName, target, valueString, accountDB); !ok {
			return false
		}
	}

	return true
}

func transferCoin(coin map[string]string, source string, target string, accountDB *account.AccountDB) bool {
	if 0 == len(coin) {
		return true
	}

	ft := make(map[string]string, len(coin))
	for key, value := range coin {
		ft[fmt.Sprintf("official-%s", key)] = value
	}

	return transferFT(ft, source, target, accountDB)
}
