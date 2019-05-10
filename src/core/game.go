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
func changeBalances(gameId string, source string, targets map[string]string, accountdb *account.AccountDB) bool {
	snapshot := accountdb.Snapshot()
	overall := big.NewInt(0)

	for address, valueString := range targets {
		value := convert(valueString)

		// 不能扣钱
		if value.Sign() == -1 {
			accountdb.RevertToSnapshot(snapshot)
			return false
		}

		if !changeBalance(address, gameId, value, accountdb) {
			accountdb.RevertToSnapshot(snapshot)
			return false
		}
		overall = overall.Add(overall, value)
	}

	// source 账户中扣钱
	overall = overall.Mul(overall, big.NewInt(-1))
	if !changeBalance(source, gameId, overall, accountdb) {
		accountdb.RevertToSnapshot(snapshot)
		return false
	}

	return true
}

// false 表示转账失败
func changeBalance(address string, gameId string, balance *big.Int, accountdb *account.AccountDB) bool {
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
		sub = &types.SubAccount{}
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
