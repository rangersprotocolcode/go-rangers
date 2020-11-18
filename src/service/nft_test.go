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
	"fmt"
	"math/big"
	"os"
	"testing"
)

var (
	sourceAddr  = common.HexStringToAddress("0xe7260a418579c2e6ca36db4fe0bf70f84d687bdf7ec6c0c181b43ee096a84aea")
	sourceAddr2 = common.HexStringToAddress("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443")
	targetAddr  = common.GenerateNFTSetAddress("test1")
	targetAddr2 = common.GenerateNFTSetAddress("0x7dba6865f337148e5887d6bea97e6a98701a2fa774bd00474ea68bcc645142f2")
)

const (
	NFTSETID1 = "nftSetId1"
	NFTID1    = "id1"
	APPID1    = "testApp1"

	NFTSETID2 = "nftSetId2"
	NFTID2    = "id2"
	APPID2    = "testApp2"

	NFTSETID3 = "nftSetId3"
	NFTID3    = "id3"
	APPID3    = "testApp3"

	NFTSETID4 = "nftSetId4"
	NFTID4    = "id4"
	APPID4    = "testApp4"

	NFTSETID5 = "nftSetId5"
	NFTID5    = "id5"
	APPID5    = "testApp5"

	NFTSETID6 = "nftSetId6"
	NFTID6    = "id6"
	APPID6    = "testApp6"
)

var (
	ftName1  = fmt.Sprintf("%s-%s", "0x10086", "abc")
	ftName2  = fmt.Sprintf("%s-%s", "0x10086", "123")
	ftName3  = fmt.Sprintf("%s-%s", "0x10086", "!@#")
	bntName1 = "ETH.ETH"
	bntName2 = "ONT"
)

func TestNFTManager_MintNFT(t *testing.T) {
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

	setId := "test1"
	name := "aaaaahh"
	symbol := "eth"

	id := "007"
	creator := "jdai"

	// 检查setId
	nftSet := NFTManagerInstance.GetNFTSet(setId, accountdb)
	if nil == nftSet {
		nftSet = NFTManagerInstance.GenerateNFTSet(setId, name, symbol, creator, creator, "", 0, "0")
		NFTManagerInstance.PublishNFTSet(nftSet, accountdb)
	}

	appId := "0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54"

	// 发行
	owner := common.HexToAddress("0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54")
	_, ok := NFTManagerInstance.GenerateNFT(nftSet, appId, setId, id, "pppp", creator, "", "0", owner, nil, accountdb)
	if !ok {
		t.Fatalf("fail to mint")
	}

	root, err := accountdb.Commit(true)
	if nil != err {
		t.Fatalf("fail to commit accountdb after mint")
	}
	err = triedb.TrieDB().Commit(root, true)
	if nil != err {
		t.Fatalf("fail to commit TrieDB after mint")
	}

	accountdb, err = account.NewAccountDB(root, triedb)
	if nil != err {
		t.Fatalf("fail to find accountdb after mint")
	}

	nftSet = NFTManagerInstance.GetNFTSet(setId, accountdb)
	if nil == nftSet || nftSet.SetID != setId {
		t.Fatalf("fail to get nftSet after mint")
	}

	nft := NFTManagerInstance.GetNFT(setId, id, accountdb)
	if nil == nft || nft.SetID != setId {
		t.Fatalf("fail to get nft after mint")
	}

	fmt.Println(accountdb.RemoveNFTByGameId(owner, setId, id))

	root, err = accountdb.Commit(true)
	if nil != err {
		t.Fatalf("fail to commit accountdb after mint")
	}
	err = triedb.TrieDB().Commit(root, true)
	if nil != err {
		t.Fatalf("fail to commit TrieDB after mint")
	}

	nft = NFTManagerInstance.GetNFT(setId, id, accountdb)
	if nil != nft {
		t.Fatalf("fail to RemoveNFTByGameId commit")
	}

	accountdb, err = account.NewAccountDB(root, triedb)
	if nil != err {
		t.Fatalf("fail to find accountdb after RemoveNFTByGameId")
	}

	nft = NFTManagerInstance.GetNFT(setId, id, accountdb)
	if nil != nft {
		t.Fatalf("fail to RemoveNFTByGameId")
	}
}

func TestNFTManager_MintNFTWithCondition(t *testing.T) {
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

	setId := "test1"
	name := "aaaaahh"
	symbol := "eth"
	id := "007"
	creator := "0xe7260a418579c2e6ca36db4fe0bf70f84d687bdf7ec6c0c181b43ee096a84aea"

	// 检查setId
	nftSet := NFTManagerInstance.GetNFTSet(setId, accountdb)
	if nil == nftSet {
		nftSet = NFTManagerInstance.GenerateNFTSet(setId, name, symbol, creator, creator, `{"nft":[{"setId":"nftSetId1","attribute":{"a":{"operate":"eq","value":"1"},"b":{"operate":"between","value":"[1, 20]"}}},{"setId":"nftSetId2","attribute":{"a":{"operate":"ne","value":"1"},"b":{"operate":"ge","value":"3"}}}],"ft":{"0x10086-abc":"0.1","0x10086-123":"0.4"},"coin":{"ETH.ETH":"1","ONT":"0.5"},"balance":"1"}`, 0, "0")
		NFTManagerInstance.PublishNFTSet(nftSet, accountdb)
	}

	// lock resource
	lockedRoot := lockResource(accountdb, triedb, t)
	accountdb, _ = account.NewAccountDB(lockedRoot, triedb)

	appId := "0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54"

	// 发行
	msg, ok := NFTManagerInstance.MintNFT(creator, appId, setId, id, "pppp", creator, sourceAddr, accountdb)
	if !ok {
		t.Fatalf(msg)
	}

	root, err := accountdb.Commit(true)
	if nil != err {
		t.Fatalf("fail to commit accountdb after mint")
	}
	err = triedb.TrieDB().Commit(root, true)
	if nil != err {
		t.Fatalf("fail to commit TrieDB after mint")
	}

	accountdb, err = account.NewAccountDB(root, triedb)
	if nil != err {
		t.Fatalf("fail to find accountdb after mint")
	}

	nftSet = NFTManagerInstance.GetNFTSet(setId, accountdb)
	if nil == nftSet || nftSet.SetID != setId {
		t.Fatalf("fail to get nftSet after mint")
	}

	nft := NFTManagerInstance.GetNFT(setId, id, accountdb)
	if nil == nft || nft.SetID != setId {
		t.Fatalf("fail to get nft after mint")
	}

	if !accountdb.RemoveNFTByGameId(sourceAddr, setId, id) {
		t.Fatalf("remove failed")
	}

	root, err = accountdb.Commit(true)
	if nil != err {
		t.Fatalf("fail to commit accountdb after mint")
	}
	err = triedb.TrieDB().Commit(root, true)
	if nil != err {
		t.Fatalf("fail to commit TrieDB after mint")
	}

	nft = NFTManagerInstance.GetNFT(setId, id, accountdb)
	if nil != nft {
		t.Fatalf("fail to RemoveNFTByGameId commit")
	}

	accountdb, err = account.NewAccountDB(root, triedb)
	if nil != err {
		t.Fatalf("fail to find accountdb after RemoveNFTByGameId")
	}

	nft = NFTManagerInstance.GetNFT(setId, id, accountdb)
	if nil != nft {
		t.Fatalf("fail to RemoveNFTByGameId")
	}

	resource := accountdb.GetLockedResourceByAddress(targetAddr, sourceAddr)
	if nil == resource {
		t.Fatalf("fail to get LockResource")
	}

	if "9.000000000" != resource.Balance {
		t.Fatalf("fail to remain balance")
	}
	if "1.500000000" != resource.Coin[bntName2] || 1 != len(resource.Coin) {
		t.Fatalf("fail to remain Coin")
	}
	if "1.600000000" != resource.FT[ftName2] || 2 != len(resource.FT) || "0.900000000" != resource.FT[ftName1] {
		t.Fatalf("fail to remain FT")
	}
	if 1 != len(resource.NFT) || resource.NFT[0].SetId != NFTSETID1 || resource.NFT[0].Id != NFTID1 {
		t.Fatalf("fail to remain FT")
	}
}

func lockResource(state *account.AccountDB, triedb account.AccountDatabase, t *testing.T) common.Hash {
	state.SetBalance(sourceAddr, big.NewInt(10000000000))
	state.SetFT(sourceAddr, ftName1, big.NewInt(10000000000))
	state.SetFT(sourceAddr, ftName2, big.NewInt(10000000000))
	state.SetFT(sourceAddr, ftName3, big.NewInt(10000000000))
	state.SetBNT(sourceAddr, bntName1, big.NewInt(10000000000))
	state.SetBNT(sourceAddr, bntName2, big.NewInt(10000000000))

	nft1 := &types.NFT{
		AppId: APPID1,
		SetID: NFTSETID1,
		ID:    NFTID1,
		Owner: sourceAddr.GetHexString(),
		Data:  make(map[string]string),
	}
	nft1.SetData(APPID1, "real")
	nft1.SetProperty(APPID1, "a", "1")
	nft1.SetProperty(APPID1, "b", "109")
	state.AddNFTByGameId(sourceAddr, APPID1, nft1)

	nft1 = &types.NFT{
		AppId: APPID1,
		SetID: NFTSETID1,
		ID:    NFTID2,
		Owner: sourceAddr.GetHexString(),
		Data:  make(map[string]string),
	}
	nft1.SetData(APPID1, "real")
	nft1.SetProperty(APPID1, "a", "1")
	nft1.SetProperty(APPID1, "b", "10")
	state.AddNFTByGameId(sourceAddr, APPID1, nft1)

	nft2 := &types.NFT{
		AppId: APPID2,
		SetID: NFTSETID2,
		ID:    NFTID2,
		Owner: sourceAddr.GetHexString(),
		Data:  make(map[string]string),
	}
	nft2.SetData(APPID2, "real2")
	nft2.SetProperty(APPID1, "a", "2")
	nft2.SetProperty(APPID1, "b", "5")
	state.AddNFTByGameId(sourceAddr, APPID2, nft2)

	nft3 := &types.NFT{
		AppId: APPID3,
		SetID: NFTSETID3,
		ID:    NFTID3,
		Owner: sourceAddr.GetHexString(),
		Data:  make(map[string]string),
	}
	nft2.SetData(APPID3, "real3")
	state.AddNFTByGameId(sourceAddr, APPID2, nft3)

	root, err := state.Commit(true)
	if nil != err {
		t.Fatalf("fail to commit")
	}
	triedb.TrieDB().Commit(root, false)

	state, err = account.NewAccountDB(root, triedb)
	resource := types.LockResource{
		Balance: "10",
		Coin:    make(map[string]string),
		FT:      make(map[string]string),
		NFT:     make([]types.NFTID, 0),
	}
	resource.Coin[bntName1] = "1"
	resource.Coin[bntName2] = "2"
	resource.FT[ftName1] = "1"
	resource.FT[ftName2] = "2"
	resource.NFT = append(resource.NFT, types.NFTID{
		SetId: NFTSETID1,
		Id:    NFTID2,
	})
	resource.NFT = append(resource.NFT, types.NFTID{
		SetId: NFTSETID1,
		Id:    NFTID1,
	})
	resource.NFT = append(resource.NFT, types.NFTID{
		SetId: NFTSETID2,
		Id:    NFTID2,
	})

	ok := state.LockResource(sourceAddr, targetAddr, resource)
	if !ok {
		t.Fatalf("fail to lockResource")
	}

	root, err = state.Commit(true)
	if nil != err {
		t.Fatalf("fail to commit")
	}
	triedb.TrieDB().Commit(root, false)

	return root
}
