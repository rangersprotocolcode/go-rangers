package core

import (
	"x/src/middleware/types"
	"x/src/common"
	"x/src/storage/account"
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
			if 0 == len(value) {
				continue
			}

			asset := &types.Asset{
				Id:    id,
				Value: value,
			}

			sub.Assets = append(sub.Assets, asset)
		}

		accountdb.UpdateSubAccount(common.HexToAddress(address), gameId, *sub)
		return
	}

	// update and append
	for assetId, assetValue := range assets {
		update := false
		for i, assetInner := range sub.Assets {
			// update
			if assetInner.Id == assetId {
				update = true

				//assetValue空字符串，则是移除
				if 0 != len(assetValue) {
					assetInner.Value = assetValue
				} else {
					sub.Assets = append(sub.Assets[:i], sub.Assets[i+1:]...)
				}
				break
			}
		}

		//append if not exists
		if !update {
			asset := &types.Asset{
				Id:    assetId,
				Value: assetValue,
			}

			sub.Assets = append(sub.Assets, asset)
		}
	}

	accountdb.UpdateSubAccount(common.HexToAddress(address), gameId, *sub)
}
