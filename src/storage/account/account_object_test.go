// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package account

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/rlp"
	"fmt"
	"math/big"
	"testing"
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
