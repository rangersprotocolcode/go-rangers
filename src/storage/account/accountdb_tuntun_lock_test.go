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

// 2 steps:
// source1 -> target1
// then source1 -> target1
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
	resource.Coin[bntName2] = "1"
	resource.FT[ftName1] = "1"
	resource.FT[ftName2] = "1"
	resource.NFT = append(resource.NFT, types.NFTID{
		SetId: NFTSETID1,
		Id:    NFTID1,
	})

	ok := state.LockResource(sourceAddr, targetAddr, resource)
	if !ok {
		t.Fatalf("fail to lockResource")
	}

	resource = types.LockResource{
		Coin: make(map[string]string),
		FT:   make(map[string]string),
		NFT:  make([]types.NFTID, 0),
	}
	resource.Coin[bntName1] = "0"
	resource.Coin[bntName2] = "1"
	resource.FT[ftName1] = "0"
	resource.FT[ftName2] = "1"
	ok = state.LockResource(sourceAddr, targetAddr, resource)
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
func TestAccountDB_LockResource3(t *testing.T) {
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

	state.SetBalance(sourceAddr2, big.NewInt(10000000000))
	state.SetFT(sourceAddr2, ftName1, big.NewInt(10000000000))
	state.SetFT(sourceAddr2, ftName2, big.NewInt(10000000000))
	state.SetFT(sourceAddr2, ftName3, big.NewInt(10000000000))
	state.SetBNT(sourceAddr2, bntName1, big.NewInt(10000000000))
	state.SetBNT(sourceAddr2, bntName2, big.NewInt(10000000000))

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
	nft3.SetData(APPID3, "real3")
	state.AddNFTByGameId(sourceAddr, APPID3, nft3)

	nft4 := &types.NFT{
		AppId: APPID4,
		SetID: NFTSETID4,
		ID:    NFTID4,
		Owner: sourceAddr2.GetHexString(),
		Data:  make(map[string]string),
	}
	nft4.SetData(APPID4, "real4")
	state.AddNFTByGameId(sourceAddr2, APPID4, nft4)

	nft5 := &types.NFT{
		AppId: APPID5,
		SetID: NFTSETID5,
		ID:    NFTID5,
		Owner: sourceAddr2.GetHexString(),
		Data:  make(map[string]string),
	}
	nft5.SetData(APPID5, "real5")
	state.AddNFTByGameId(sourceAddr2, APPID5, nft5)

	nft6 := &types.NFT{
		AppId: APPID6,
		SetID: NFTSETID6,
		ID:    NFTID6,
		Owner: sourceAddr2.GetHexString(),
		Data:  make(map[string]string),
	}
	nft6.SetData(APPID6, "real6")
	state.AddNFTByGameId(sourceAddr2, APPID6, nft6)

	root, err := state.Commit(true)
	if nil != err {
		t.Fatalf("fail to commit")
	}
	triedb.TrieDB().Commit(root, false)

	// lock start
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

	resource2 := types.LockResource{
		Balance: "1",
		Coin:    make(map[string]string),
		FT:      make(map[string]string),
		NFT:     make([]types.NFTID, 0),
	}
	resource2.Coin[bntName1] = "1.5"
	resource2.Coin[bntName2] = "1"
	resource2.FT[ftName1] = "1"
	resource2.FT[ftName2] = "1.2"
	resource2.NFT = append(resource2.NFT, types.NFTID{
		SetId: NFTSETID5,
		Id:    NFTID5,
	})
	ok2 := state.LockResource(sourceAddr2, targetAddr, resource2)
	if !ok2 {
		t.Fatalf("fail to lockResource2")
	}

	root, err = state.Commit(true)
	if nil != err {
		t.Fatalf("fail to commit")
	}
	triedb.TrieDB().Commit(root, false)

	// check start
	state, err = NewAccountDB(root, triedb)
	left := state.GetBalance(sourceAddr)
	if nil != left && 0 != left.Sign() {
		t.Fatalf("left balance error: %s", left.String())
	}
	left = state.GetBalance(sourceAddr2)
	if nil != left && "9000000000" != left.String() {
		t.Fatalf("left2 balance error: %s", left.String())
	}

	left = state.GetBNT(sourceAddr, bntName1)
	if left == nil || "9000000000" != left.String() {
		t.Fatalf("%s remains error: %s", bntName1, left)
	}
	left = state.GetBNT(sourceAddr, bntName2)
	if left == nil || "8000000000" != left.String() {
		t.Fatalf("%s remains error: %s", bntName2, left)
	}

	left = state.GetBNT(sourceAddr2, bntName1)
	if left == nil || "8500000000" != left.String() {
		t.Fatalf("%s remains error: %s", bntName1, left)
	}
	left = state.GetBNT(sourceAddr2, bntName2)
	if left == nil || "9000000000" != left.String() {
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
	left = state.GetFT(sourceAddr2, ftName1)
	if left == nil || "9000000000" != left.String() {
		t.Fatalf("%s remains error: %s", ftName1, left)
	}
	left = state.GetFT(sourceAddr2, ftName2)
	if left == nil || "8800000000" != left.String() {
		t.Fatalf("%s remains error: %s", ftName2, left)
	}
	left = state.GetFT(sourceAddr2, ftName3)
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

	nftList = state.GetAllNFT(sourceAddr)
	if nil == nftList || 3 != len(nftList) {
		t.Fatalf("source: %s has wrong nftList", sourceAddr.String())
	} else {
		for _, nft := range nftList {
			if nft.ID == NFTID5 && nft.SetID == NFTSETID5 && nft.Status != 3 && targetAddr.String() != nft.Lock {
				t.Fatalf("nft status error")
			}
		}
	}

	if state.ChangeNFTOwner(sourceAddr, targetAddr, NFTSETID1, NFTID1) {
		t.Fatalf("nft ChangeNFTOwner error")
	}
	if state.ChangeNFTOwner(sourceAddr2, targetAddr, NFTSETID5, NFTID5) {
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

	lockedResource = state.GetLockedResourceByAddress(targetAddr, sourceAddr2)
	if "1.000000000" != lockedResource.Balance {
		t.Fatalf("locked balance error, expect: %s", lockedResource.Balance)
	}
	if nil == lockedResource.Coin || 2 != len(lockedResource.Coin) || "1.500000000" != lockedResource.Coin[bntName1] || "1.000000000" != lockedResource.Coin[bntName2] {
		t.Fatalf("locked bnt error: %s", lockedResource.Coin)
	}
	if nil == lockedResource.FT || 2 != len(lockedResource.FT) || "1.000000000" != lockedResource.FT[ftName1] || "1.200000000" != lockedResource.FT[ftName2] {
		t.Fatalf("locked bnt error: %s", lockedResource.FT)
	}
	if nil == lockedResource.NFT || 1 != len(lockedResource.NFT) || NFTSETID5 != lockedResource.NFT[0].SetId || NFTID5 != lockedResource.NFT[0].Id {
		t.Fatalf("locked bnt error: %s", lockedResource.NFT)
	}

	allLocked := state.GetLockedResource(targetAddr)
	lockedResource = allLocked[sourceAddr2.String()]

	fmt.Println(len(allLocked))
	if nil == allLocked {
		t.Fatalf("error to get lockedResource")
	}

}

// lock: source1 -> target1
// source2 -> target1
// unlock: target1->source1
// target1->source2 (all)
func TestAccountDB_LockResource4(t *testing.T) {
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

	state.SetBalance(sourceAddr2, big.NewInt(10000000000))
	state.SetFT(sourceAddr2, ftName1, big.NewInt(10000000000))
	state.SetFT(sourceAddr2, ftName2, big.NewInt(10000000000))
	state.SetFT(sourceAddr2, ftName3, big.NewInt(10000000000))
	state.SetBNT(sourceAddr2, bntName1, big.NewInt(10000000000))
	state.SetBNT(sourceAddr2, bntName2, big.NewInt(10000000000))

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
	nft3.SetData(APPID3, "real3")
	state.AddNFTByGameId(sourceAddr, APPID3, nft3)

	nft4 := &types.NFT{
		AppId: APPID4,
		SetID: NFTSETID4,
		ID:    NFTID4,
		Owner: sourceAddr2.GetHexString(),
		Data:  make(map[string]string),
	}
	nft4.SetData(APPID4, "real4")
	state.AddNFTByGameId(sourceAddr2, APPID4, nft4)

	nft5 := &types.NFT{
		AppId: APPID5,
		SetID: NFTSETID5,
		ID:    NFTID5,
		Owner: sourceAddr2.GetHexString(),
		Data:  make(map[string]string),
	}
	nft5.SetData(APPID5, "real5")
	state.AddNFTByGameId(sourceAddr2, APPID5, nft5)

	nft6 := &types.NFT{
		AppId: APPID6,
		SetID: NFTSETID6,
		ID:    NFTID6,
		Owner: sourceAddr2.GetHexString(),
		Data:  make(map[string]string),
	}
	nft6.SetData(APPID6, "real6")
	state.AddNFTByGameId(sourceAddr2, APPID6, nft6)

	root, err := state.Commit(true)
	if nil != err {
		t.Fatalf("fail to commit")
	}
	triedb.TrieDB().Commit(root, false)

	// lock start
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

	resource2 := types.LockResource{
		Balance: "1",
		Coin:    make(map[string]string),
		FT:      make(map[string]string),
		NFT:     make([]types.NFTID, 0),
	}
	resource2.Coin[bntName1] = "1.5"
	resource2.Coin[bntName2] = "1"
	resource2.FT[ftName1] = "1"
	resource2.FT[ftName2] = "1.2"
	resource2.NFT = append(resource2.NFT, types.NFTID{
		SetId: NFTSETID5,
		Id:    NFTID5,
	})
	ok2 := state.LockResource(sourceAddr2, targetAddr, resource2)
	if !ok2 {
		t.Fatalf("fail to lockResource2")
	}

	root, err = state.Commit(true)
	if nil != err {
		t.Fatalf("fail to commit")
	}
	triedb.TrieDB().Commit(root, false)

	// check start
	state, err = NewAccountDB(root, triedb)
	left := state.GetBalance(sourceAddr)
	if nil != left && 0 != left.Sign() {
		t.Fatalf("left balance error: %s", left.String())
	}
	left = state.GetBalance(sourceAddr2)
	if nil != left && "9000000000" != left.String() {
		t.Fatalf("left2 balance error: %s", left.String())
	}

	left = state.GetBNT(sourceAddr, bntName1)
	if left == nil || "9000000000" != left.String() {
		t.Fatalf("%s remains error: %s", bntName1, left)
	}
	left = state.GetBNT(sourceAddr, bntName2)
	if left == nil || "8000000000" != left.String() {
		t.Fatalf("%s remains error: %s", bntName2, left)
	}

	left = state.GetBNT(sourceAddr2, bntName1)
	if left == nil || "8500000000" != left.String() {
		t.Fatalf("%s remains error: %s", bntName1, left)
	}
	left = state.GetBNT(sourceAddr2, bntName2)
	if left == nil || "9000000000" != left.String() {
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
	left = state.GetFT(sourceAddr2, ftName1)
	if left == nil || "9000000000" != left.String() {
		t.Fatalf("%s remains error: %s", ftName1, left)
	}
	left = state.GetFT(sourceAddr2, ftName2)
	if left == nil || "8800000000" != left.String() {
		t.Fatalf("%s remains error: %s", ftName2, left)
	}
	left = state.GetFT(sourceAddr2, ftName3)
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

	nftList = state.GetAllNFT(sourceAddr)
	if nil == nftList || 3 != len(nftList) {
		t.Fatalf("source: %s has wrong nftList", sourceAddr.String())
	} else {
		for _, nft := range nftList {
			if nft.ID == NFTID5 && nft.SetID == NFTSETID5 && nft.Status != 3 && targetAddr.String() != nft.Lock {
				t.Fatalf("nft status error")
			}
		}
	}

	if state.ChangeNFTOwner(sourceAddr, targetAddr, NFTSETID1, NFTID1) {
		t.Fatalf("nft ChangeNFTOwner error")
	}
	if state.ChangeNFTOwner(sourceAddr2, targetAddr, NFTSETID5, NFTID5) {
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

	lockedResource = state.GetLockedResourceByAddress(targetAddr, sourceAddr2)
	if "1.000000000" != lockedResource.Balance {
		t.Fatalf("locked balance error, expect: %s", lockedResource.Balance)
	}
	if nil == lockedResource.Coin || 2 != len(lockedResource.Coin) || "1.500000000" != lockedResource.Coin[bntName1] || "1.000000000" != lockedResource.Coin[bntName2] {
		t.Fatalf("locked bnt error: %s", lockedResource.Coin)
	}
	if nil == lockedResource.FT || 2 != len(lockedResource.FT) || "1.000000000" != lockedResource.FT[ftName1] || "1.200000000" != lockedResource.FT[ftName2] {
		t.Fatalf("locked bnt error: %s", lockedResource.FT)
	}
	if nil == lockedResource.NFT || 1 != len(lockedResource.NFT) || NFTSETID5 != lockedResource.NFT[0].SetId || NFTID5 != lockedResource.NFT[0].Id {
		t.Fatalf("locked bnt error: %s", lockedResource.NFT)
	}

	allLocked := state.GetLockedResource(targetAddr)
	fmt.Println(len(allLocked))
	if nil == allLocked {
		t.Fatalf("error to get lockedResource")
	}

	lockedResource = allLocked[sourceAddr.String()]
	fmt.Println(lockedResource)

	//unlock
	state, err = NewAccountDB(root, triedb)
	lockedResource = &types.LockResource{
		Balance: "4",
		Coin:    make(map[string]string),
		FT:      make(map[string]string),
	}
	lockedResource.Coin[bntName1] = "1"
	lockedResource.Coin[bntName2] = "1.5"
	lockedResource.FT[ftName1] = "1"
	lockedResource.FT[ftName2] = "0.5"

	state.UnLockResource(sourceAddr, targetAddr, *lockedResource)

	root, err = state.Commit(true)
	if nil != err {
		t.Fatalf("fail to commit")
	}
	triedb.TrieDB().Commit(root, false)

	//check unlock
	state, err = NewAccountDB(root, triedb)
	left = state.GetBalance(sourceAddr)
	if left == nil || "4000000000" != left.String() {
		t.Fatalf("unlock failed: %s", state.GetBalance(sourceAddr).String())
	}
	left = state.GetBNT(sourceAddr, bntName1)
	if left == nil || "10000000000" != left.String() {
		t.Fatalf("%s remains error: %s", bntName1, left)
	}
	left = state.GetBNT(sourceAddr, bntName2)
	if left == nil || "9500000000" != left.String() {
		t.Fatalf("%s remains error: %s", bntName2, left)
	}
	left = state.GetFT(sourceAddr, ftName1)
	if left == nil || "10000000000" != left.String() {
		t.Fatalf("%s remains error: %s", ftName1, left)
	}
	left = state.GetFT(sourceAddr, ftName2)
	if left == nil || "8500000000" != left.String() {
		t.Fatalf("%s remains error: %s", ftName2, left)
	}
	left = state.GetFT(sourceAddr, ftName3)
	if left == nil || "10000000000" != left.String() {
		t.Fatalf("%s remains error: %s", ftName3, left)
	}

	allLocked = state.GetLockedResource(targetAddr)
	if nil == allLocked || 2 != len(allLocked) {
		t.Fatalf("error to get lockedResource")
	}
	fmt.Println(allLocked[sourceAddr.String()])
	fmt.Println(allLocked[sourceAddr2.String()])

	// unlock source2 all
	state, err = NewAccountDB(root, triedb)
	state.UnLockResource(sourceAddr2, targetAddr, *allLocked[sourceAddr2.String()])
	allLocked = state.GetLockedResource(targetAddr)
	if nil == allLocked || 1 != len(allLocked) {
		t.Fatalf("error to get lockedResource")
	}
	fmt.Println(allLocked[sourceAddr.String()])
	fmt.Println(allLocked[sourceAddr2.String()])

	root, err = state.Commit(true)
	if nil != err {
		t.Fatalf("fail to commit")
	}
	triedb.TrieDB().Commit(root, false)
}
