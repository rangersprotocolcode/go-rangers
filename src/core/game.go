package core

import (
	"x/src/middleware/types"
	"x/src/common"
	"math/big"
	"strconv"
)

func GetSubAccount(address string, gameId string) *types.SubAccount {
	accountdb := GetBlockChain().GetAccountDB()
	return accountdb.GetSubAccount(common.HexToAddress(address), gameId)
}

func UpdateAsset(user types.UserData, gameId string) {
	if 0 != len(user.Balance) {
		changeBalance(user.Address, gameId, user.Balance)
	}

	if 0 != len(user.AssetId) && len(user.AssetId) == len(user.AssetValue) {
		setAsset(user.Address, gameId, user.AssetId, user.AssetValue)
	}

}

func convert(s string) *big.Int {
	f, _ := strconv.ParseFloat(s, 64)
	return big.NewInt(int64(f * 1000000000))
}

func changeBalance(address string, gameId string, bstring string) {
	balance := convert(bstring)
	sub := GetSubAccount(address, gameId)
	if sub != nil {
		sub.Balance = sub.Balance.Add(balance, sub.Balance)
	} else {
		sub = &types.SubAccount{}
		sub.Balance = balance
	}

	GetBlockChain().GetAccountDB().UpdateSubAccount(common.HexToAddress(address), gameId, *sub)
}

func setAsset(address string, gameId string, assetIds, assetValues []string) {
	if nil == assetIds || 0 == len(assetIds) {
		return
	}

	sub := GetSubAccount(address, gameId)
	if sub == nil {
		sub = &types.SubAccount{}
	}

	// append everything if there is no asset right now
	if nil == sub.Assets || 0 == len(sub.Assets) {
		sub.Assets = []types.Asset{}
		for i, _ := range assetIds {
			asset := &types.Asset{
				Id:    assetIds[i],
				Value: []byte(assetValues[i]),
			}

			sub.Assets = append(sub.Assets, *asset)
		}

		GetBlockChain().GetAccountDB().UpdateSubAccount(common.HexToAddress(address), gameId, *sub)
		return
	}

	// update and append
	for i, assetId := range assetIds {
		update := false
		for _, assetInner := range sub.Assets {
			// update
			if assetInner.Id == assetId {
				assetInner.Value = []byte(assetValues[i])
				update = true
				break
			}
		}

		//append if not exists
		if !update {
			asset := &types.Asset{
				Id:    assetId,
				Value: []byte(assetValues[i]),
			}

			sub.Assets = append(sub.Assets, *asset)
		}
	}

	GetBlockChain().GetAccountDB().UpdateSubAccount(common.HexToAddress(address), gameId, *sub)
}
