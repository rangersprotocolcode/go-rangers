package core

import (
	"x/src/middleware/types"
	"x/src/common"
	"math/big"
	"strconv"
	"x/src/storage/account"
)

func GetSubAccount(address string, gameId string, account *account.AccountDB) *types.SubAccount {
	return account.GetSubAccount(common.HexToAddress(address), gameId)
}

func UpdateAsset(user types.UserData, gameId string, account *account.AccountDB) {
	if 0 != len(user.Balance) {
		changeBalance(user.Address, gameId, user.Balance, account)
	}

	if 0 != len(user.Assets) {
		setAsset(user.Address, gameId, user.Assets, account)
	}

}

func convert(s string) *big.Int {
	f, _ := strconv.ParseFloat(s, 64)
	return big.NewInt(int64(f * 1000000000))
}

func changeBalance(address string, gameId string, bstring string, accountdb *account.AccountDB) {
	balance := convert(bstring)
	sub := GetSubAccount(address, gameId, accountdb)
	if sub != nil {
		sub.Balance = sub.Balance.Add(balance, sub.Balance)
	} else {
		sub = &types.SubAccount{}
		sub.Balance = balance
	}

	accountdb.UpdateSubAccount(common.HexToAddress(address), gameId, *sub)
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
		sub.Assets = []*types.Asset{}
		for id, value := range assets {
			asset := &types.Asset{
				Id:    id,
				Value: []byte(value),
			}

			sub.Assets = append(sub.Assets, asset)
		}

		accountdb.UpdateSubAccount(common.HexToAddress(address), gameId, *sub)
		return
	}

	// update and append
	for assetId, assetValue := range assets {
		update := false
		for _, assetInner := range sub.Assets {
			// update
			if assetInner.Id == assetId {
				assetInner.Value = []byte(assetValue)
				update = true
				break
			}
		}

		//append if not exists
		if !update {
			asset := &types.Asset{
				Id:    assetId,
				Value: []byte(assetValue),
			}

			sub.Assets = append(sub.Assets, asset)
		}
	}

	accountdb.UpdateSubAccount(common.HexToAddress(address), gameId, *sub)
}
