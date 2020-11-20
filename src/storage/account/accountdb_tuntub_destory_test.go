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
	"math/big"
	"os"
	"testing"
)

// source1 -> target1
// source2 -> target1
// destroy: source2 all
func TestAccountDB_DestroyResource(t *testing.T) {
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
	if nil == allLocked || 2 != len(allLocked) {
		t.Fatalf("error to get lockedResource")
	}

	// destroy start
	// destroy sourceAddr2 all
	state, err = NewAccountDB(root, triedb)
	if !state.DestroyResource(sourceAddr2, targetAddr, *allLocked[sourceAddr2.String()]) {
		t.Fatalf("DestroyResource sourceAddr2 failed")
	}
	// destroy sourceAddr something
	resource = types.LockResource{
		Balance: "4",
		Coin:    make(map[string]string),
		FT:      make(map[string]string),
		NFT:     make([]types.NFTID, 0),
	}
	resource.Coin[bntName1] = "1"
	resource.Coin[bntName2] = "1.6"
	resource.FT[ftName1] = "0.9"
	resource.FT[ftName2] = "1.2"
	resource.NFT = append(resource.NFT, types.NFTID{
		SetId: NFTSETID1,
		Id:    NFTID1,
	})
	if !state.DestroyResource(sourceAddr, targetAddr, resource) {
		t.Fatalf("DestroyResource sourceAddr failed")
	}
	root, err = state.Commit(true)
	if nil != err {
		t.Fatalf("fail to commit")
	}
	triedb.TrieDB().Commit(root, false)

	// check destroy
	state, err = NewAccountDB(root, triedb)
	allLocked = state.GetLockedResource(targetAddr)
	if nil == allLocked || 1 != len(allLocked) || nil != allLocked[sourceAddr2.String()] {
		t.Fatalf("error to get lockedResource")
	}
	lockedResource = allLocked[sourceAddr.String()]
	if "6.000000000" != lockedResource.Balance {
		t.Fatalf("lockedResource.Balance error: %v", lockedResource.Balance)
	}
	if nil == lockedResource.Coin || 1 != len(lockedResource.Coin) || "0.400000000" != lockedResource.Coin[bntName2] {
		t.Fatalf("lockedResource.Coin error: %v", lockedResource.Coin)
	}
	if nil == lockedResource.FT || 2 != len(lockedResource.FT) || "0.800000000" != lockedResource.FT[ftName2] || "0.100000000" != lockedResource.FT[ftName1] {
		t.Fatalf("lockedResource.FT error: %v", lockedResource.FT)
	}
	if 0 != len(lockedResource.NFT) {
		t.Fatalf("lockedResource.NFT error: %v", lockedResource.NFT)
	}
	// check balance
	left = state.GetBalance(sourceAddr)
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

	nftList = state.GetAllNFT(sourceAddr)
	if nil == nftList || 2 != len(nftList) {
		t.Fatalf("source: %s has wrong nftList", sourceAddr.String())
	}

	nftList = state.GetAllNFT(sourceAddr2)
	if nil == nftList || 2 != len(nftList) {
		t.Fatalf("source: %s has wrong nftList", sourceAddr2.String())
	}

	if state.ChangeNFTOwner(sourceAddr, targetAddr, NFTSETID1, NFTID1) {
		t.Fatalf("nft ChangeNFTOwner error")
	}
	if state.ChangeNFTOwner(sourceAddr2, targetAddr, NFTSETID5, NFTID5) {
		t.Fatalf("nft ChangeNFTOwner error")
	}
}