package account

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/db"
	"com.tuntun.rocket/node/src/middleware/types"
	"fmt"
	"math/big"
	"os"
	"testing"
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
)

var (
	ftName1  = fmt.Sprintf("%s-%s", "0x10086", "abc")
	ftName2  = fmt.Sprintf("%s-%s", "0x10086", "123")
	ftName3  = fmt.Sprintf("%s-%s", "0x10086", "!@#")
	bntName1 = "ETH.ETH"
	bntName2 = "ONT"
)

var (
	sourceAddr  = common.HexStringToAddress("0xe7260a418579c2e6ca36db4fe0bf70f84d687bdf7ec6c0c181b43ee096a84aea")
	sourceAddr2 = common.HexStringToAddress("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443")
	targetAddr  = common.GenerateNFTSetAddress("0xa72263e15d3a48de8cbde1f75c889c85a15567b990aab402691938f4511581ab")
	targetAddr2 = common.GenerateNFTSetAddress("0x7dba6865f337148e5887d6bea97e6a98701a2fa774bd00474ea68bcc645142f2")
)

// source1 -> target1
func TestAccountDB_LockResource1(t *testing.T) {
	os.RemoveAll("storage0")
	defer os.RemoveAll("storage0")
	db, _ := db.NewLDBDatabase("test", 0, 0)
	defer db.Close()
	triedb := NewDatabase(db)
	state, _ := NewAccountDB(common.Hash{}, triedb)

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
	state.AddNFTByGameId(sourceAddr, APPID1, nft1)

	nft2 := &types.NFT{
		AppId: APPID2,
		SetID: NFTSETID2,
		ID:    NFTID2,
		Owner: sourceAddr.GetHexString(),
		Data:  make(map[string]string),
	}
	nft2.SetData(APPID2, "real2")
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

	state, err = NewAccountDB(root, triedb)
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
		Id:    NFTID1,
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

	left := state.GetBalance(sourceAddr)
	if nil != left && 0 != left.Sign() {
		t.Fatalf("left balance error: %s", left.String())
	}

	left = state.GetBNT(sourceAddr, bntName1)
	if left == nil || "9000000000" != left.String() {
		t.Fatalf("%s remains error: %s", bntName1, left)
	}
	left = state.GetBNT(sourceAddr, bntName2)
	if left == nil || "8000000000" != left.String() {
		t.Fatalf("%s remains error: %s", bntName2, left)
	}

	left = state.GetFT(sourceAddr, ftName1)
	if left == nil || "9000000000" != left.String() {
		t.Fatalf("%s remains error: %s", ftName1, left)
	}
	left = state.GetFT(sourceAddr, ftName2)
	if left == nil || "8000000000" != left.String() {
		t.Fatalf("%s remains error: %s", ftName2, left)
	}
	left = state.GetFT(sourceAddr, ftName3)
	if left == nil || "10000000000" != left.String() {
		t.Fatalf("%s remains error: %s", ftName3, left)
	}
	nftList := state.GetAllNFT(sourceAddr)
	if nil == nftList || 3 != len(nftList) {
		t.Fatalf("source: %s has wrong nftList", sourceAddr.String())
	} else {
		for _, nft := range nftList {
			if nft.ID == NFTID1 && nft.SetID == NFTSETID1 && nft.Status != 3 && targetAddr.String() != nft.Lock {
				t.Fatalf("nft status error")
			}
		}
	}

	if state.ChangeNFTOwner(sourceAddr, targetAddr, NFTSETID1, NFTID1) {
		t.Fatalf("nft ChangeNFTOwner error")
	}

	lockedResource := state.GetLockedResourceByAddress(targetAddr, sourceAddr)
	if "10.000000000" != lockedResource.Balance {
		t.Fatalf("locked balance error, expect: %s", lockedResource.Balance)
	}
	if nil == lockedResource.Coin || 2 != len(lockedResource.Coin) || "1.000000000" != lockedResource.Coin[bntName1] || "2.000000000" != lockedResource.Coin[bntName2] {
		t.Fatalf("locked bnt error: %s", lockedResource.Coin)
	}
	if nil == lockedResource.FT || 2 != len(lockedResource.FT) || "1.000000000" != lockedResource.FT[ftName1] || "2.000000000" != lockedResource.FT[ftName2] {
		t.Fatalf("locked bnt error: %s", lockedResource.FT)
	}
	if nil == lockedResource.NFT || 1 != len(lockedResource.NFT) || NFTSETID1 != lockedResource.NFT[0].SetId || NFTID1 != lockedResource.NFT[0].Id {
		t.Fatalf("locked bnt error: %s", lockedResource.NFT)
	}
}

// source1 -> target1
// source2 -> target1
func TestAccountDB_LockResource2(t *testing.T) {
	os.RemoveAll("storage0")
	defer os.RemoveAll("storage0")
	db, _ := db.NewLDBDatabase("test", 0, 0)
	defer db.Close()
	triedb := NewDatabase(db)
	state, _ := NewAccountDB(common.Hash{}, triedb)

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
	state.AddNFTByGameId(sourceAddr, APPID1, nft1)

	nft2 := &types.NFT{
		AppId: APPID2,
		SetID: NFTSETID2,
		ID:    NFTID2,
		Owner: sourceAddr.GetHexString(),
		Data:  make(map[string]string),
	}
	nft2.SetData(APPID2, "real2")
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

	state, err = NewAccountDB(root, triedb)
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
		Id:    NFTID1,
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

	left := state.GetBalance(sourceAddr)
	if nil != left && 0 != left.Sign() {
		t.Fatalf("left balance error: %s", left.String())
	}

	left = state.GetBNT(sourceAddr, bntName1)
	if left == nil || "9000000000" != left.String() {
		t.Fatalf("%s remains error: %s", bntName1, left)
	}
	left = state.GetBNT(sourceAddr, bntName2)
	if left == nil || "8000000000" != left.String() {
		t.Fatalf("%s remains error: %s", bntName2, left)
	}

	left = state.GetFT(sourceAddr, ftName1)
	if left == nil || "9000000000" != left.String() {
		t.Fatalf("%s remains error: %s", ftName1, left)
	}
	left = state.GetFT(sourceAddr, ftName2)
	if left == nil || "8000000000" != left.String() {
		t.Fatalf("%s remains error: %s", ftName2, left)
	}
	left = state.GetFT(sourceAddr, ftName3)
	if left == nil || "10000000000" != left.String() {
		t.Fatalf("%s remains error: %s", ftName3, left)
	}
	nftList := state.GetAllNFT(sourceAddr)
	if nil == nftList || 3 != len(nftList) {
		t.Fatalf("source: %s has wrong nftList", sourceAddr.String())
	} else {
		for _, nft := range nftList {
			if nft.ID == NFTID1 && nft.SetID == NFTSETID1 && nft.Status != 3 && targetAddr.String() != nft.Lock {
				t.Fatalf("nft status error")
			}
		}
	}

	if state.ChangeNFTOwner(sourceAddr, targetAddr, NFTSETID1, NFTID1) {
		t.Fatalf("nft ChangeNFTOwner error")
	}

	lockedResource := state.GetLockedResourceByAddress(targetAddr, sourceAddr)
	if "10.000000000" != lockedResource.Balance {
		t.Fatalf("locked balance error, expect: %s", lockedResource.Balance)
	}
	if nil == lockedResource.Coin || 2 != len(lockedResource.Coin) || "1.000000000" != lockedResource.Coin[bntName1] || "2.000000000" != lockedResource.Coin[bntName2] {
		t.Fatalf("locked bnt error: %s", lockedResource.Coin)
	}
	if nil == lockedResource.FT || 2 != len(lockedResource.FT) || "1.000000000" != lockedResource.FT[ftName1] || "2.000000000" != lockedResource.FT[ftName2] {
		t.Fatalf("locked bnt error: %s", lockedResource.FT)
	}
	if nil == lockedResource.NFT || 1 != len(lockedResource.NFT) || NFTSETID1 != lockedResource.NFT[0].SetId || NFTID1 != lockedResource.NFT[0].Id {
		t.Fatalf("locked bnt error: %s", lockedResource.NFT)
	}
}
