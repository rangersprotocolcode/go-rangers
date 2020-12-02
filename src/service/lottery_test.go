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

package service

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/db"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCreateLottery(t *testing.T) {
	os.RemoveAll("storage0")
	os.RemoveAll("logs")
	os.Remove("1.ini")
	defer os.RemoveAll("logs")
	defer os.RemoveAll("storage0")
	defer os.Remove("1.ini")

	common.InitConf("1.ini")
	InitService()
	db, _ := db.NewLDBDatabase("test", 0, 0)
	defer db.Close()
	triedb := account.NewDatabase(db)

	accountdb, _ := account.NewAccountDB(common.Hash{}, triedb)
	NFTManagerInstance.PublishNFTSet(&types.NFTSet{
		SetID:     NFTSETID1,
		MaxSupply: 0,
		Owner:     sourceAddr.GetHexString(),
	}, accountdb)
	NFTManagerInstance.PublishNFTSet(&types.NFTSet{
		SetID:     NFTSETID2,
		MaxSupply: 0,
		Owner:     sourceAddr.GetHexString(),
	}, accountdb)
	FTManagerInstance.PublishFTSet(&types.FTSet{
		AppId:     "test",
		Symbol:    "f1",
		ID:        "ftSetId1",
		MaxSupply: big.NewInt(0),
		Owner:     sourceAddr.GetHexString(),
	}, accountdb)
	FTManagerInstance.PublishFTSet(&types.FTSet{
		AppId:     "test",
		Symbol:    "f2",
		ID:        "ftSetId2",
		MaxSupply: big.NewInt(0),
		Owner:     sourceAddr.GetHexString(),
	}, accountdb)

	conditions := `{"combo":{"p":"0.4","content":{"3":"0.2","2":"0.4","4":"0.4"}},"prizes":{"nft":{"p":"0.4","content":{"nftSetId1":"0.2","nftSetId2":"0.8"}},"ft":{"p":"0.3","content":{"ftSetId1":{"p":"0.2","range":"0-1"},"ftSetId2":{"p":"0.8","range":"1-100"}}}}}`
	id, reason := CreateLottery(sourceAddr.GetHexString(), conditions, accountdb)
	fmt.Println(id)
	fmt.Println(reason)
	root, err := accountdb.Commit(true)
	if nil != err {
		t.Fatalf("fail to commit accountdb after mint")
	}
	err = triedb.TrieDB().Commit(root, true)
	if nil != err {
		t.Fatalf("fail to commit TrieDB after mint")
	}
}

func TestJackpot(t *testing.T) {
	os.RemoveAll("storage0")
	os.RemoveAll("logs")
	os.Remove("1.ini")
	defer os.RemoveAll("logs")
	defer os.RemoveAll("storage0")
	defer os.Remove("1.ini")

	common.InitConf("1.ini")
	InitService()
	db, _ := db.NewLDBDatabase("test", 0, 0)
	defer db.Close()
	triedb := account.NewDatabase(db)

	accountdb, _ := account.NewAccountDB(common.Hash{}, triedb)
	NFTManagerInstance.PublishNFTSet(&types.NFTSet{
		SetID:     NFTSETID1,
		MaxSupply: 0,
		Owner:     sourceAddr.GetHexString(),
	}, accountdb)
	NFTManagerInstance.PublishNFTSet(&types.NFTSet{
		SetID:     NFTSETID2,
		MaxSupply: 0,
		Owner:     sourceAddr.GetHexString(),
	}, accountdb)
	FTManagerInstance.PublishFTSet(&types.FTSet{
		AppId:     "test",
		Symbol:    "f1",
		ID:        "ftSetId1",
		MaxSupply: big.NewInt(0),
		Owner:     sourceAddr.GetHexString(),
	}, accountdb)
	FTManagerInstance.PublishFTSet(&types.FTSet{
		AppId:     "test",
		Symbol:    "f2",
		ID:        "ftSetId2",
		MaxSupply: big.NewInt(0),
		Owner:     sourceAddr.GetHexString(),
	}, accountdb)

	conditions := `{"combo":{"p":"1","content":{"3":"0.4","2":"0.4"}},"prizes":{"nft":{"p":"0.4","content":{"nftSetId1":"0.2","nftSetId2":"0.8"}},"ft":{"p":"0.6","content":{"ftSetId1":{"p":"0.2","range":"0-1"},"ftSetId2":{"p":"0.8","range":"1-100"}}}}}`
	id, reason := CreateLottery(sourceAddr.GetHexString(), conditions, accountdb)
	if 0 != len(reason) {
		t.Fatalf(reason)
	}

	root, err := accountdb.Commit(true)
	if nil != err {
		t.Fatalf("fail to commit accountdb after mint")
	}
	err = triedb.TrieDB().Commit(root, true)
	if nil != err {
		t.Fatalf("fail to commit TrieDB after mint")
	}

	accountdb, _ = account.NewAccountDB(root, triedb)
	m := make(map[int]int, 0)
	for i := uint64(0); i < 100; i++ {
		answer, _ := Jackpot(id, sourceAddr2.GetHexString(), uint64(time.Now().UnixNano()), 1024+i*common.BlocksPerDay, accountdb)
		var it items
		json.Unmarshal(utility.StrToBytes(answer), &it)
		length := len(it.Nft) + len(it.Ft)
		m[length] = m[length] + 1
		for _, nft := range it.Nft {
			nft := accountdb.GetNFTById(nft.SetId, nft.Id)
			if nft == nil {
				t.Fatalf("fail to get nft %s %s", nft.SetID, nft.ID)
			}
			if 0 != strings.Compare(nft.Owner, sourceAddr2.GetHexString()) {
				t.Fatalf("fail to assign nft %s %s", nft.SetID, nft.ID)
			}
		}

	}
	fmt.Println(m)
}
