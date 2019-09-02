package core

import (
	"x/src/middleware/types"
	"x/src/common"
	"x/src/storage/account"
	"math/big"
	"strconv"
	"x/src/statemachine"
)

func UpdateAsset(user types.UserData, gameId string, account *account.AccountDB) bool {
	balanceDelta := convert(user.Balance)
	if balanceDelta.Sign() == -1 {
		// 扣玩家钱。这里允许扣钱，为了状态机操作方便（理论上是需要用户签名的）

		// 1. 先从玩家账户里扣
		if !changeBalance(user.Address, gameId, balanceDelta, account) {
			return false
		}

		// 2. 给游戏钱，游戏账户也即gameId
		var b = big.NewInt(0)
		b.Mul(balanceDelta, big.NewInt(-1))
		changeBalance(gameId, gameId, b, account)
	} else if balanceDelta.Sign() == 1 {
		// 1. 先从游戏账户里扣，游戏账户也即gameId
		var b = big.NewInt(0)
		b.Mul(balanceDelta, big.NewInt(-1))
		if !changeBalance(gameId, gameId, b, account) {
			return false
		}

		// 2. 给玩家钱
		changeBalance(user.Address, gameId, balanceDelta, account)
	}

	if 0 != len(user.Assets) {
		setAsset(user.Address, gameId, user.Assets, account)
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
func changeBalances(gameId string, source string, targets map[string]types.TransferData, accountdb *account.AccountDB) bool {
	//sub := GetSubAccount(source, gameId, accountdb)
	sourceAddr := common.HexToAddress(source)

	for address, transferData := range targets {
		targetAddr := common.HexToAddress(address)

		// 转钱
		if !transferBalance(transferData.Balance, sourceAddr, targetAddr, gameId, accountdb) {
			logger.Debugf("Change balance failed!")
			return false
		}

		// 转FT
		ftList := transferData.FT
		if 0 != len(ftList) {
			// 游戏给用户转FT的特殊流程
			// 根据source，来判断是否是游戏自己的地址
			if statemachine.Docker.IsGame(source) {
				for ftName, valueString := range ftList {
					_, flag := TransferFT(gameId, ftName, address, valueString, accountdb)
					if !flag {
						logger.Debugf("Game Change ft failed!")
						return false
					}
				}
			} else if !transferFT(ftList, sourceAddr, targetAddr, gameId, accountdb) {
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

func transferNFT(nft []string, source common.Address, target common.Address, gameId string, db *account.AccountDB) bool {
	if 0 == len(nft) {
		return true
	}

	sourceNFT := db.GetAllNFTByGameId(source, gameId)
	for _, id := range nft {
		value := sourceNFT[id]
		if 0 == len(value) {
			return false
		}

		//todo: target
		db.SetNFTByGameId(target, gameId, id, value)
		db.RemoveNFTByGameId(source, gameId, id)
	}

	return true
}

func transferBalance(value string, source common.Address, target common.Address, gameId string, accountDB *account.AccountDB) bool {
	balance := convert(value)
	// 不能扣钱
	if balance.Sign() == -1 {
		return false
	}

	sourceBalance := accountDB.GetBalanceByGameId(source, gameId)

	// 钱不够转账，再见
	if sourceBalance.Cmp(balance) == -1 {
		return false
	}

	// 目标加钱
	targetBalance := accountDB.GetBalanceByGameId(target, gameId)
	targetBalance = targetBalance.Add(balance, targetBalance)
	accountDB.SetBalanceByGameId(target, gameId, targetBalance)

	// 自己减钱
	sourceBalance = sourceBalance.Sub(sourceBalance, balance)
	accountDB.SetBalanceByGameId(source, gameId, sourceBalance)
	return true
}

func transferFT(ft map[string]string, source common.Address, target common.Address, gameId string, accountDB *account.AccountDB) bool {
	if 0 == len(ft) {
		return true
	}

	sourceFt := accountDB.GetAllFT(source)

	for ftName, valueString := range ft {
		owner := sourceFt[ftName]
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

// false 表示转账失败
// 给address账户下的gameId子账户转账
// 允许扣钱
func changeBalance(addressString string, gameId string, balance *big.Int, accountdb *account.AccountDB) bool {
	common.DefaultLogger.Debugf("change balance: addr:%s,balance:%v,gameId:%s", addressString, balance, gameId)
	address := common.HexToAddress(addressString)
	subBalance := accountdb.GetBalanceByGameId(address, gameId)

	if subBalance != nil {
		subBalance = subBalance.Add(balance, subBalance)
	} else {
		subBalance = balance
	}

	if subBalance.Sign() == -1 {
		return false
	}

	accountdb.SetBalanceByGameId(address, gameId, subBalance)
	return true
}

func setAsset(addressString string, gameId string, assets map[string]string, accountdb *account.AccountDB) {
	if nil == assets || 0 == len(assets) {
		return
	}

	address := common.HexToAddress(addressString)
	sub := accountdb.GetAllNFTByGameId(address, gameId)

	// append everything if there is no asset right now
	if nil == sub || 0 == len(sub) {
		for id, value := range assets {
			if 0 == len(value) {
				continue
			}

			accountdb.SetNFTByGameId(address, gameId, id, value)
		}

		return
	}

	// update/add and delete
	for assetId, assetValue := range assets {
		// 已有，assetValue空字符串，则是移除
		if 0 != len(sub[assetId]) && 0 == len(assetValue) {
			accountdb.RemoveNFTByGameId(address, gameId, assetId)
			continue
		}

		//update/add
		accountdb.SetNFTByGameId(address, gameId, assetId, assetValue)
	}

}
