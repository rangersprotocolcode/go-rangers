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
	"com.tuntun.rocket/node/src/utility"
	"fmt"
	"math/big"
	"os"
	"testing"
)

func TestAccountDB_AddFT(t *testing.T) {
	db, _ := db.NewLDBDatabase("test", 0, 0)
	defer db.Close()
	triedb := NewDatabase(db)
	state, _ := NewAccountDB(common.Hash{}, triedb)
	value := big.NewInt(60)
	address := common.HexToAddress("0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54")
	state.AddFT(address, "official-eth", value)

	money := state.GetFT(address, "official-eth")
	if value.Cmp(money) != 0 {
		t.Fatalf("123-%s", money)
	}
	fmt.Printf("before commit %s\n", money)
	root, _ := state.Commit(true)
	triedb.TrieDB().Commit(root, false)

	money = state.GetFT(address, "official-eth")
	fmt.Printf("after commit %s\n", money)
	if value.Cmp(money) != 0 {
		t.Fatalf("123-%s", money)
	}

	state2, _ := NewAccountDB(root, triedb)
	money = state2.GetFT(address, "official-eth")
	fmt.Printf("new accountdb %s\n", money)
	if value.Cmp(money) != 0 {
		t.Fatalf("123-%s", money)
	}

	state2.AddFT(address, "official-eth", value)
	money = state2.GetFT(address, "official-eth")
	fmt.Printf("add again %s\n", money)

	root, _ = state2.Commit(true)
	triedb.TrieDB().Commit(root, false)
	money = state2.GetFT(address, "official-eth")
	fmt.Printf("after commit %s\n", money)

}

func TestAccountDB_AddNFT(t *testing.T) {
	os.RemoveAll("storage0")
	defer os.RemoveAll("storage0")
	db, _ := db.NewLDBDatabase("test", 0, 0)
	defer db.Close()
	triedb := NewDatabase(db)
	state, _ := NewAccountDB(common.Hash{}, triedb)
	address := common.HexToAddress("0x443")

	nft := &types.NFT{}
	nft.SetID = "1"
	nft.ID = "a"
	nft.SetData("sword", "test")
	nft.AppId = "test"
	state.AddNFTByGameId(address, "test", nft)

	nft1 := &types.NFT{}
	nft1.SetID = "11"
	nft1.ID = "ab"
	nft1.SetData("bow", "test")
	nft1.AppId = "test"
	state.AddNFTByGameId(address, "test", nft1)

	nft2 := &types.NFT{}
	nft2.SetID = "11"
	nft2.ID = "abc"
	nft2.SetData("bow", "test2")
	nft2.AppId = "test2"
	state.AddNFTByGameId(address, "test", nft2)

	state.SetData(address, utility.StrToBytes("dj"), utility.StrToBytes("rp"))
	nftList := state.GetAllNFT(address)
	fmt.Println(len(nftList))

	root, _ := state.Commit(true)
	triedb.TrieDB().Commit(root, false)

	accountDB, _ := NewAccountDB(root, triedb)

	nftRead := accountDB.GetNFTById(address, "11", "abc")
	if nil == nftRead {
		t.Fatalf("no nft for 11&abd")
	}

	nftList1 := accountDB.GetAllNFT(address)
	fmt.Println(len(nftList1))

	nftList2 := accountDB.GetAllNFTByGameId(address, "test")
	fmt.Println(len(nftList2))

}


func TestAccountDB_SetCode(t *testing.T) {
	os.RemoveAll("storage0")
	defer os.RemoveAll("storage0")
	db, _ := db.NewLDBDatabase("test", 0, 0)
	defer db.Close()
	triedb := NewDatabase(db)
	state, _ := NewAccountDB(common.Hash{}, triedb)
	address := common.HexToAddress("0x443")

	state.SetNFTSetDefinition(address, utility.StrToBytes("12345"))
	fmt.Println(utility.BytesToStr(state.getNFTSetDefinition(address)))

	root, err := state.Commit(true)
	if err != nil {
		t.Fatalf(err.Error())
	}

	state, _ = NewAccountDB(root, triedb)
	fmt.Println(utility.BytesToStr(state.getNFTSetDefinition(address)))
}
