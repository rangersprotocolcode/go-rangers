package core

import (
	"x/src/middleware/types"
	"x/src/common"
	"x/src/storage/account"
	"math/big"
	"strconv"
	"x/src/statemachine"
	"strings"
	"fmt"
)

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
	if !transferFT(user.Coin, appAddr, userAddr, accountDB, true) {
		logger.Debugf("Change coin failed!")
		return false
	}

	// 转FT
	ftList := user.FT
	if 0 != len(ftList) {
		for ftName, valueString := range ftList {
			ftInfo := strings.Split(ftName, "-")
			if 2 != len(ftInfo) || ftInfo[0] != appId {
				return false
			}
			_, flag := TransferFT(ftInfo[0], ftInfo[1], user.Address, valueString, accountDB)
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
			if !nftManagerInstance.UpdateNFT(userAddr, appId, nft.SetId, nft.Id, nft.Data, accountDB) {
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
// 注意：如果source是游戏本身，那么FT会走其专属流程
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
		if !transferFT(transferData.Coin, sourceAddr, targetAddr, accountdb, true) {
			logger.Debugf("Change coin failed!")
			return false
		}

		// 转FT
		ftList := transferData.FT
		if 0 != len(ftList) {
			// 游戏给用户转FT的特殊流程
			// 根据source，来判断是否是游戏自己的地址
			if statemachine.Docker.IsGame(source) {
				for ftName, valueString := range ftList {
					ftInfo := strings.Split(ftName, "-")
					//
					if 2 != len(ftInfo) || ftInfo[0] != source {
						return false
					}
					_, flag := TransferFT(ftInfo[0], ftInfo[1], address, valueString, accountdb)
					if !flag {
						logger.Debugf("Game Change ft failed!")
						return false
					}
				}
			} else if !transferFT(ftList, sourceAddr, targetAddr, accountdb, false) {
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
		_, flag := nftManagerInstance.Transfer(appId, id.SetId, id.Id, source, target, db)
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

// 这里只处理玩家之间的ft转让，不涉及状态机给玩家转ft
func transferFT(ft map[string]string, source common.Address, target common.Address, accountDB *account.AccountDB, isCoin bool) bool {
	if 0 == len(ft) {
		return true
	}

	sourceFt := accountDB.GetAllFT(source)

	for ftName, valueString := range ft {
		var owner *big.Int
		if isCoin {
			owner = sourceFt[fmt.Sprintf("official-%s", ftName)]
		} else {
			owner = sourceFt[ftName]
		}

		if nil == owner {
			return false
		}

		value := convert(valueString)

		//logger.Debugf("ft name:%s,value:%s,value convert:%s,owner value:%s", ftName, valueString, value.String(), ownerValue.String())
		// ft 数量不够，再见
		if owner.Cmp(value) == -1 {
			return false
		}

		accountDB.AddFT(target, ftName, value)
		accountDB.SubFT(source, ftName, value)
	}

	return true
}
