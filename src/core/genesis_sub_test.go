package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/db"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/vm"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"
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

	conf := &common.GenesisConf{
		Name:           "testChain001",
		TokenName:      "myCoin",
		Symbol:         "mc",
		TotalSupply:    100000,
		Cast:           2000,
		TimeCycle:      10,
		ProposalToken:  30,
		ValidatorToken: 40,
	}
	createEconomyContract(block.Header, stateDB, conf)
	createSubCrossContract(block.Header, stateDB, conf.Name)
}

func TestEconomyContract(t *testing.T) {
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
		Height:       1,
		ProveValue:   pv,
		TotalQN:      0,
		Transactions: make([]common.Hashes, 0), //important!!
		EvictedTxs:   make([]common.Hash, 0),   //important!!
		Nonce:        ChainDataVersion,
		CurTime:      time.Now(),
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

	conf := &common.GenesisConf{
		Name:           "testChain001",
		TokenName:      "myCoin",
		Symbol:         "mc",
		TotalSupply:    100000,
		Cast:           2000,
		TimeCycle:      10,
		ProposalToken:  30,
		ValidatorToken: 40,
	}
	createEconomyContract(block.Header, stateDB, conf)
	createSubCrossContract(block.Header, stateDB, conf.Name)

	header := block.Header
	source := "0x0000000000000000000000000000000000000001"
	//         0x826f575031a074fd914a869b5dc1c4eae620fef5
	vmCtx := vm.Context{}
	vmCtx.CanTransfer = vm.CanTransfer
	vmCtx.Transfer = vm.Transfer
	vmCtx.GetHash = func(uint64) common.Hash { return emptyHash }

	vmCtx.Origin = common.HexToAddress(source)
	vmCtx.Coinbase = common.BytesToAddress(header.Castor)
	vmCtx.BlockNumber = new(big.Int).SetUint64(header.Height)
	vmCtx.Time = new(big.Int).SetUint64(uint64(header.CurTime.Unix()))

	vmCtx.GasPrice = big.NewInt(1)
	vmCtx.GasLimit = 30000000
	vmInstance := vm.NewEVM(vmCtx, stateDB)
	caller := vm.AccountRef(vmCtx.Origin)

	//code := "0x7822b9ac"
	//+ 出块人奖励地址 + "0000000000000000000000000000000000000000000000000000000000000060" + utility.GenerateCallDataUint((4+len(proposes))*32)
	//+utility.GenerateCallDataUint(len(proposes)) + 提案组成员地址 + utility.GenerateCallDataUint(len(验证组成员)) + 验证组成员地址

	code:="0x7822b9ac000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000002e00000000000000000000000000000000000000000000000000000000000000013000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef50000000000000000000000000000000000000000000000000000000000000014000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5000000000000000000000000826f575031a074fd914a869b5dc1c4eae620fef5"
	codeBytes := common.FromHex(code)
	ret, _, logs, err := vmInstance.Call(caller, common.HexToAddress("0x71d9cfd1b7adb1e8eb4c193ce6ffbe19b4aee0db"), codeBytes, vmCtx.GasLimit, big.NewInt(0))
	fmt.Println(ret)
	fmt.Println(logs)
	if err != nil {
		t.Fatal(err)
	}
}
