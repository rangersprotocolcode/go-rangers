package account

import (
	"testing"
	"math/big"
	"x/src/common"
	"x/src/middleware/types"
	"x/src/storage/rlp"
	"fmt"
)

func Test_RLP_account(t *testing.T) {
	account := Account{}
	if account.Balance == nil {
		account.Balance = new(big.Int)
	}
	if account.Ft == nil {
		account.Ft = make([]*types.FT, 0)
	}
	if account.CodeHash == nil {
		account.CodeHash = emptyCodeHash[:]
	}
	if account.GameData == nil {
		account.GameData = &types.GameData{}
		nftMap := &types.NFTMap{}
		nft := &types.NFT{ID: "sword1", Name: "yitai", Symbol: "yt", SetID: "game1"}
		nftMap.SetNFT(nft)
		account.GameData.SetNFTMaps("test1", nftMap)
	}

	data, err := rlp.EncodeToBytes(account)
	if err != nil {
		t.Fatalf("%s", err.Error())
	}

	fmt.Println(data)
	fmt.Println(common.ToHex(data))

	accountBeta := &Account{}
	err = rlp.DecodeBytes(data, accountBeta)

	fmt.Println(accountBeta.GameData.GetNFTMaps("test1").GetNFT("game1", "sword1").Name)
}
