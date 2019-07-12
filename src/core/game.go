package core

import (
	"x/src/middleware/types"
	"x/src/common"
	"x/src/storage/account"
	"math/big"
	"strconv"
)

func GetSubAccount(address string, gameId string, account *account.AccountDB) *types.SubAccount {
	return account.GetSubAccount(common.HexToAddress(address), gameId)
}

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

// false 表示转账失败
// 这里是玩家与玩家（游戏）之间的转账，不操作游戏对玩家对转账
// 这里不处理事务。调用本方法之前处理
func changeBalances(gameId string, source string, targets map[string]string, accountdb *account.AccountDB) bool {
	overall := big.NewInt(0)

	for address, valueString := range targets {
		value := convert(valueString)

		// 不能扣钱
		if value.Sign() == -1 {
			return false
		}

		if !changeBalance(address, gameId, value, accountdb) {
			return false
		}
		overall = overall.Add(overall, value)
	}

	// source 账户中扣钱
	overall = overall.Mul(overall, big.NewInt(-1))
	if !changeBalance(source, gameId, overall, accountdb) {
		return false
	}

	return true
}

// false 表示转账失败
// 给address账户下的gameId子账户转账
func changeBalance(address string, gameId string, balance *big.Int, accountdb *account.AccountDB) bool {
	common.DefaultLogger.Debugf("change balance: addr:%s,balance:%v,gameId:%s", address, balance, gameId)
	sub := GetSubAccount(address, gameId, accountdb)

	if sub != nil {
		sub.Balance = sub.Balance.Add(balance, sub.Balance)
	} else {
		sub = &types.SubAccount{}
		sub.Balance = balance
	}

	if sub.Balance.Sign() == -1 {
		return false
	}

	accountdb.UpdateSubAccount(common.HexToAddress(address), gameId, *sub)
	return true
}

func setAsset(address string, gameId string, assets map[string]string, accountdb *account.AccountDB) {
	if nil == assets || 0 == len(assets) {
		return
	}

	sub := GetSubAccount(address, gameId, accountdb)
	if sub == nil {
		sub = &types.SubAccount{Balance: new(big.Int).SetUint64(0)}
	}

	// append everything if there is no asset right now
	if nil == sub.Assets || 0 == len(sub.Assets) {
		sub.Assets = make(map[string]string)
		for id, value := range assets {
			if 0 == len(value) {
				continue
			}

			sub.Assets[id] = value
		}

		accountdb.UpdateSubAccount(common.HexToAddress(address), gameId, *sub)
		return
	}

	// update/add and delete
	for assetId, assetValue := range assets {
		// 已有，assetValue空字符串，则是移除
		if 0 != len(sub.Assets[assetId]) && 0 == len(assetValue) {
			delete(sub.Assets, assetId)
			continue
		}

		//update/add
		sub.Assets[assetId] = assetValue
	}

	accountdb.UpdateSubAccount(common.HexToAddress(address), gameId, *sub)
}
