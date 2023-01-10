package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/db"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/vm"
	"math/big"
	"os"
	"testing"
)

func TestCreateSubCrossContract(t *testing.T) {
	defer func() {
		os.RemoveAll("storage0")
		os.RemoveAll("logs")
		os.RemoveAll("1.ini")
	}()

	common.InitChainConfig("dev")
	common.InitConf("1.ini")
	vm.InitVM()

	block := new(types.Block)
	pv := big.NewInt(0)
	block.Header = &types.BlockHeader{
		Height:       0,
		ProveValue:   pv,
		TotalQN:      0,
		Transactions: make([]common.Hashes, 0), //important!!
		EvictedTxs:   make([]common.Hash, 0),   //important!!
		Nonce:        ChainDataVersion,
	}

	block.Header.RequestIds = make(map[string]uint64)

	db, err := db.NewLDBDatabase("state", 128, 2048)
	if err != nil {
		t.Fatal(err)
	}
	stateDB, err := account.NewAccountDB(common.Hash{}, account.NewDatabase(db))
	if err != nil {
		t.Fatal(err)
	}

	createEconomyContract(block.Header, stateDB, "mycoin", "mc", 100000000)
	createSubCrossContract(block.Header, stateDB, "testChain001")
}
