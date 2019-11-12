package account

import (
	"testing"
	"x/src/middleware/db"
	"x/src/common"
	"math/big"
	"fmt"
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
