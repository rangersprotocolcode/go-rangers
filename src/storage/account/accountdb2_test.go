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
	"fmt"
	"testing"
)

func TestAccountDB_AddBalance(t *testing.T) {
	// Create an empty state database
	//db, _ := db.NewMemDatabase()
	db, _ := db.NewLDBDatabase("account/test", 0, 0)
	//db, _ := db.NewLDBDatabase("/Volumes/Work/work/test", 0, 0)
	defer db.Close()
	triedb := NewDatabase(db)
	//state, _ := NewAccountDB(common.Hash{}, triedb)
	state, _ := NewAccountDB(common.HexToHash("0xb9642a136059a6723b481d43969bd3b2c749d440af17d1a49b056728f60e9033"), triedb)

	//state.SetBalance(common.BytesToAddress([]byte("1")), big.NewInt(1000000))
	//state.AddBalance(common.BytesToAddress([]byte("3")), big.NewInt(1))
	//state.SubBalance(common.BytesToAddress([]byte("2")), big.NewInt(2))
	//fmt.Println(state.GetData(common.BytesToAddress([]byte("1")),"ab"))
	//state.SetData(common.BytesToAddress([]byte("1")), "ab", []byte{1,2,3})
	balance := state.GetBalance(common.BytesToAddress([]byte("1")))
	//
	//balance = state.GetBalance(common.BytesToAddress([]byte("3")))
	fmt.Println(balance)
	//state.Fstring()
	//fmt.Println(state.IntermediateRoot(true).Hex())
	//state.Fstring()
	root, _ := state.Commit(true)
	fmt.Println(root.Hex())
	triedb.TrieDB().Commit(root, true)
}

func TestAccountDB_GetBalance(t *testing.T) {
	db, _ := db.NewLDBDatabase("/Volumes/Work/work/test", 0, 0)
	defer db.Close()
	triedb := NewDatabase(db)
	state, _ := NewAccountDB(common.HexToHash("0x5d283f9d1a0bbafa7d0187ce616f1e8067d59828e1bae79e15a6a4ca06389e60"), triedb)
	balance := state.GetBalance(common.BytesToAddress([]byte("1")))
	fmt.Println(balance)
}

func TestAccountDB_SetData(t *testing.T) {
	// Create an empty state database
	//db, _ := db.NewMemDatabase()
	//db, _ := db.NewLDBDatabase("/Users/Kaede/TasProject/work/test", 0, 0)
	db, _ := db.NewLDBDatabase("/Volumes/Work/work/test", 0, 0)
	defer db.Close()
	triedb := NewDatabase(db)
	state, _ := NewAccountDB(common.Hash{}, triedb)
	state.SetData(common.BytesToAddress([]byte("1")), []byte("aa"), []byte{1,2,3})

	state.SetData(common.BytesToAddress([]byte("1")), []byte("bb"), []byte{1})
	snapshot := state.Snapshot()
	state.SetData(common.BytesToAddress([]byte("1")), []byte("bb"), []byte{2})
	state.RevertToSnapshot(snapshot)
	state.SetData(common.BytesToAddress([]byte("2")), []byte("cc"), []byte{1,2})
	fmt.Println(state.IntermediateRoot(false).Hex())
	root, _ := state.Commit(false)
	fmt.Println(root.Hex())
	triedb.TrieDB().Commit(root, false)
}

func TestAccountDB_GetData(t *testing.T) {
	//db, _ := db.NewLDBDatabase("/Users/Kaede/TasProject/work/test", 0, 0)
	db, _ := db.NewLDBDatabase("/Volumes/Work/work/test", 0, 0)
	defer db.Close()
	triedb := NewDatabase(db)
	state, _ := NewAccountDB(common.HexToHash("0x8df8e749765d8bca4db3e957c66369bb058e64108a25c69f3513430ceba79eff"), triedb)
	//fmt.Println(string(state.Dump()))
	sta := state.GetData(common.BytesToAddress([]byte("1")), []byte("aa"))
	fmt.Println(sta)
	sta = state.GetData(common.BytesToAddress([]byte("1")), []byte("bb"))
	fmt.Println(sta)
	sta = state.GetData(common.BytesToAddress([]byte("2")), []byte("cc"))
	fmt.Println(sta)
	hash := state.IntermediateRoot(true)
	fmt.Println(hash.Hex())
}