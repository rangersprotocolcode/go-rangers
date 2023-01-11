package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/db"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"com.tuntun.rocket/node/src/vm"
	"fmt"
	"math/big"
	"os"
	"strconv"
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

	economyContract := createEconomyContract(block.Header, stateDB, conf)
	createSubCrossContract(block.Header, stateDB, conf.Name)
	money, _ := utility.StrToBigInt(strconv.FormatUint(conf.TotalSupply, 10))
	stateDB.SetBalance(economyContract, money)

	header := block.Header
	source := "0x1111111111111111111111111111111111111111"

	vmCtx := vm.Context{}
	vmCtx.CanTransfer = vm.CanTransfer
	vmCtx.Transfer = TransferWithLog
	vmCtx.GetHash = func(uint64) common.Hash { return emptyHash }

	vmCtx.Origin = common.HexToAddress(source)
	vmCtx.Coinbase = common.BytesToAddress(header.Castor)
	vmCtx.BlockNumber = new(big.Int).SetUint64(header.Height)
	vmCtx.Time = new(big.Int).SetUint64(uint64(header.CurTime.Unix()))

	vmCtx.GasPrice = big.NewInt(1)
	vmCtx.GasLimit = 30000000
	vmInstance := vm.NewEVMWithNFT(vmCtx, stateDB,stateDB)
	caller := vm.AccountRef(vmCtx.Origin)

	//code := "0x7822b9ac"
	//+ 出块人奖励地址 + "0000000000000000000000000000000000000000000000000000000000000060" + utility.GenerateCallDataUint((4+len(proposes))*32)
	//+utility.GenerateCallDataUint(len(proposes)) + 提案组成员地址 + utility.GenerateCallDataUint(len(验证组成员)) + 验证组成员地址

	code := "0x7822b9ac0000000000000000000000001111111111111111111111111111111111111111000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000001000000000000000000000000111111111111111111111111111111111111111100000000000000000000000000000000000000000000000000000000000000010000000000000000000000001111111111111111111111111111111111111111"
	codeBytes := common.FromHex(code)
	ret, _, logs, err := vmInstance.Call(caller, common.HexToAddress("0x71d9cfd1b7adb1e8eb4c193ce6ffbe19b4aee0db"), codeBytes, vmCtx.GasLimit, big.NewInt(0))
	fmt.Println(ret)
	fmt.Println(logs)
	if err != nil {
		t.Fatal(err)
	}
}


func TransferWithLog(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {
	fmt.Printf("sender: %s, recipient: %s, amount: %s\n",sender.String(),recipient.String(),amount.String())
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}
