package account

import (
	"testing"
	"x/src/middleware/db"
	"x/src/common"
	"math/big"
)

func TestAccountDB_AddFT(t *testing.T) {
	db, _ := db.NewLDBDatabase("test", 0, 0)
	defer db.Close()
	triedb := NewDatabase(db)
	state, _ := NewAccountDB(common.Hash{}, triedb)
	value := big.NewInt(60)
	state.AddFT(common.HexToAddress("0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54"), "official-eth", value)

	money := state.GetFT(common.HexToAddress("0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54"), "official-eth")
	if value.Cmp(money) != 0 {
		t.Fatalf("123-%s", money)
	}

	root, _ := state.Commit(true)
	money = state.GetFT(common.HexToAddress("0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54"), "official-eth")
	if value.Cmp(money) != 0 {
		t.Fatalf("123-%s", money)
	}

	state2, _ := NewAccountDB(root, triedb)
	money = state2.GetFT(common.HexToAddress("0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54"), "official-eth")
	if value.Cmp(money) != 0 {
		t.Fatalf("123-%s", money)
	}
}
