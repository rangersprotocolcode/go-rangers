package account

import (
	"testing"
	"math/big"
	"x/src/middleware/types"
	"x/src/storage/rlp"
	"fmt"
	"x/src/common"
)

func Test_RLP_account(t *testing.T) {
	account := Account{}
	if account.Balance == nil {
		account.Balance = new(big.Int)
	}
	if account.Ft == nil {
		account.Ft = make([]*types.FT, 0)
	}
	if account.CodeHash == nil {
		account.CodeHash = emptyCodeHash[:]
	}
	if account.GameData == nil {
		account.GameData = &types.GameData{}
	}

	data, err := rlp.EncodeToBytes(account)
	if err != nil {
		t.Fatalf("%s", err.Error())
	}

	fmt.Println(data)
}

type AccountTest struct {
	Nonce    uint64
	Root     common.Hash
	CodeHash []byte

	Balance *big.Int
	Ft      []*types.FT
}
