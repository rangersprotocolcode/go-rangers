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
	"math/big"
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
	triedb.TrieDB().Commit(root,false)

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
	triedb.TrieDB().Commit(root,false)
	money = state2.GetFT(address, "official-eth")
	fmt.Printf("after commit %s\n", money)

}
