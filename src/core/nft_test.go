package core

import (
	"testing"
	"x/src/common"
	"x/src/storage/account"
	"x/src/middleware/db"
	"fmt"
)

func TestNFTManager_MintNFT(t *testing.T) {
	initNFTManager()
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
		nftSet = NFTManagerInstance.GenerateNFTSet(setId, name, symbol, creator, creator, 0, "0")
		NFTManagerInstance.PublishNFTSet(nftSet, accountdb)
	}

	appId := "0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54"

	// 发行
	owner := common.HexToAddress("0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54")
	_, ok := NFTManagerInstance.GenerateNFT(nftSet, appId, setId, id, "pppp", creator, "0", owner, nil, accountdb)
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

	fmt.Println(accountdb.RemoveNFTByGameId(owner, appId, setId, id))

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
