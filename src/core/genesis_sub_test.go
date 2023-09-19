// Copyright 2020 The RangersProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RangersProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RangersProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/db"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"com.tuntun.rocket/node/src/vm"
	"fmt"
	"golang.org/x/crypto/sha3"
	"math/big"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestGnosisContract(t *testing.T) {
	defer func() {
		os.RemoveAll("storage0")
		os.RemoveAll("logs")
		os.RemoveAll("1.ini")
	}()

	common.Init(0, "1.ini", "dev")
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

	createGnosisContract(block.Header, stateDB)
}

func TestCreateSubCrossContract(t *testing.T) {
	defer func() {
		os.RemoveAll("storage0")
		os.RemoveAll("logs")
		os.RemoveAll("1.ini")
	}()

	common.Init(0, "1.ini", "dev")
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
		Creator:        "0xAb8483F64d9C6d1EcF9b849Ae677dD3315835cb2",
		Name:           "testChain001",
		TokenName:      "myCoin",
		Symbol:         "mc",
		TotalSupply:    100000,
		Cast:           2000,
		TimeCycle:      10,
		ProposalToken:  30,
		ValidatorToken: 40,
		Stake:          "1000",
		TotalReward:    "10000",
		TargetHeight:   100000,
	}
	createEconomyContract(block.Header, stateDB, conf)

	proxy, rpg := createSubCrossContract(block.Header, stateDB, conf)

	amount, err := utility.StrToBigInt(conf.Stake)
	if err != nil {
		panic(err)
	}
	createSubGovernance(block.Header, stateDB, rpg, proxy, common.HexToAddress(conf.Creator), amount, conf.TotalReward, conf.TargetHeight)

	fmt.Println(service.GetSubChainStatus(stateDB))

	source := conf.Creator
	vmCtx := vm.Context{}
	vmCtx.CanTransfer = vm.CanTransfer
	vmCtx.Transfer = vm.Transfer
	vmCtx.GetHash = func(uint64) common.Hash { return emptyHash }
	vmCtx.Origin = common.HexToAddress(source)
	vmCtx.Coinbase = vmCtx.Origin
	vmCtx.BlockNumber = new(big.Int).SetUint64(1)
	vmCtx.Time = new(big.Int).SetInt64(time.Now().UnixMilli())
	vmCtx.GasPrice = big.NewInt(1)
	vmCtx.GasLimit = 30000000
	vmInstance := vm.NewEVMWithNFT(vmCtx, stateDB, stateDB)
	caller := vm.AccountRef(vmCtx.Origin)
	whitelist := common.HexToAddress("0x826f575031a074fd914a869b5dc1c4eae620fef5")
	contractCodeBytes := common.FromHex("0a3b0a4f" + common.GenerateCallDataAddress(whitelist))
	_, _, _, err = vmInstance.Call(caller, common.CreateWhiteListAddr, contractCodeBytes, vmCtx.GasLimit, big.NewInt(0))
	if err != nil {
		panic("Genesis cross contract create error:" + err.Error())
	}

	data := [64]byte{}
	copy(data[12:], common.FromHex("0x826f575031a074fd914a869b5dc1c4eae620fef5"))
	copy(data[64-len(common.CreateWhiteListPostion):], common.CreateWhiteListPostion)
	hasher := sha3.NewLegacyKeccak256().(common.KeccakState)
	hasher.Write(data[:])
	key := [32]byte{}
	hasher.Read(key[:])

	value := stateDB.GetData(common.CreateWhiteListAddr, key[:])
	fmt.Println(value)
	if 0 == len(value) || 1 != value[len(value)-1] {
		panic("whitelist error")
	}
	value = stateDB.GetData(common.CreateWhiteListAddr, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	fmt.Println(common.ToHex(value))
}

func TestEconomyContract(t *testing.T) {
	defer func() {
		os.RemoveAll("storage0")
		os.RemoveAll("logs")
		os.RemoveAll("1.ini")
	}()

	common.Init(0, "1.ini", "dev")
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
	createSubCrossContract(block.Header, stateDB, conf)
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
	vmInstance := vm.NewEVMWithNFT(vmCtx, stateDB, stateDB)
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
	fmt.Printf("sender: %s, recipient: %s, amount: %s\n", sender.String(), recipient.String(), amount.String())
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}

func TestSubWhiteList(t *testing.T) {
	defer func() {
		os.RemoveAll("storage0")
		os.RemoveAll("logs")
		os.RemoveAll("1.ini")
	}()

	common.Init(0, "1.ini", "dev")
	vm.InitVM()

	db, err := db.NewLDBDatabase("state", 128, 2048)
	if err != nil {
		t.Fatal(err)
	}
	stateDB, err := account.NewAccountDB(common.Hash{}, account.NewDatabase(db))
	if err != nil {
		t.Fatal(err)
	}

	source := "0x5B38Da6a701c568545dCfcB03FcB875f56beddC4"
	vmCtx := vm.Context{}
	vmCtx.CanTransfer = vm.CanTransfer
	vmCtx.Transfer = vm.Transfer
	vmCtx.GetHash = func(uint64) common.Hash { return emptyHash }

	vmCtx.Origin = common.HexToAddress(source)
	vmCtx.Coinbase = common.HexToAddress("0x5B38Da6a701c568545dCfcB03FcB875f56beddC4")
	vmCtx.BlockNumber = new(big.Int).SetUint64(1)
	vmCtx.Time = new(big.Int).SetUint64(uint64(time.Now().UnixMilli()))

	vmCtx.GasPrice = big.NewInt(1)
	vmCtx.GasLimit = 30000000
	vmInstance := vm.NewEVMWithNFT(vmCtx, stateDB, stateDB)
	caller := vm.AccountRef(vmCtx.Origin)

	addr := common.HexToAddress("0x5B38Da6a701c568545dCfcB03FcB875f56beddC4")
	code := createWhiteList + common.GenerateCallDataAddress(addr)
	_, whitelist, _, _, err := vmInstance.Create(caller, common.FromHex(code), vmCtx.GasLimit, big.NewInt(0))
	if err != nil {
		panic("Genesis createWhiteList create error:" + err.Error())
	}
	fmt.Println("After execute whitelist contract create! Contract address: " + whitelist.GetHexString())

	data := stateDB.GetData(whitelist, getKey(addr))
	fmt.Println(data[len(data)-1])

	addr1 := common.HexToAddress("0x5B38Da6a701c568545dCfcB03FcB875f56beddC1")
	fmt.Println(stateDB.GetData(whitelist, getKey(addr1)))
}

func getKey(address common.Address) []byte {
	data := [64]byte{}
	copy(data[12:], address.Bytes())
	copy(data[64-len(common.CreateWhiteListPostion):], common.CreateWhiteListPostion)
	hasher := sha3.NewLegacyKeccak256().(common.KeccakState)
	hasher.Write(data[:])
	key := [32]byte{}
	hasher.Read(key[:])

	return key[:]
}

func TestWorkContract(t *testing.T) {
	defer func() {
		os.RemoveAll("storage0")
		os.RemoveAll("logs")
		os.RemoveAll("1.ini")
	}()

	common.Init(0, "1.ini", "dev")
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
	header := block.Header

	db, err := db.NewLDBDatabase("state", 128, 2048)
	if err != nil {
		t.Fatal(err)
	}
	stateDB, err := account.NewAccountDB(common.Hash{}, account.NewDatabase(db))
	if err != nil {
		t.Fatal(err)
	}

	//source := "0x0"
	vmCtx := vm.Context{}
	vmCtx.CanTransfer = vm.CanTransfer
	vmCtx.Transfer = vm.Transfer
	vmCtx.GetHash = func(uint64) common.Hash { return emptyHash }

	//vmCtx.Origin = common.HexToAddress(source)
	vmCtx.Coinbase = common.BytesToAddress(header.Castor)
	vmCtx.BlockNumber = new(big.Int).SetUint64(header.Height)
	vmCtx.Time = new(big.Int).SetUint64(uint64(header.CurTime.Unix()))

	vmCtx.GasPrice = big.NewInt(1)
	vmCtx.GasLimit = 30000000
	vmInstance := vm.NewEVMWithNFT(vmCtx, stateDB, stateDB)
	caller := vm.AccountRef(vmCtx.Origin)

	contractCodeBytes := common.FromHex("608060405234801561001057600080fd5b506101fd806100206000396000f3fe608060405234801561001057600080fd5b506004361061002b5760003560e01c806387f614e114610030575b600080fd5b6100a76004803603602081101561004657600080fd5b810190808035906020019064010000000081111561006357600080fd5b82018360208201111561007557600080fd5b8035906020019184602083028401116401000000008311171561009757600080fd5b90919293919293905050506100a9565b005b60008282905011610122576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260138152602001807f5365744d696e6572732070617261206e756c6c0000000000000000000000000081525060200191505060405180910390fd5b600073ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16146101c4576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260188152602001807f5365744d696e65727320696e76616c69642063616c6c6572000000000000000081525060200191505060405180910390fd5b505056fea265627a7a72315820aa0ff6663c54f3dd299a2eafecc4d6f33ec6aa4516c5835320537c25dab9c56564736f6c63430005110032")
	_, subGovernance, _, _, err := vmInstance.Create(caller, contractCodeBytes, vmCtx.GasLimit, big.NewInt(0))
	if err != nil {
		panic("Genesis subGovernance contract create error:" + err.Error())
	}
	fmt.Println("After execute subGovernance contract create! Contract address: " + subGovernance.GetHexString())

	contractCodeBytes = common.FromHex("0x87f614e100000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000004000000000000000000000000c6ea7fe7060d6251f0be2e6e2aba9b071c4943fc0000000000000000000000000e05d86e7943d7f041fabde02f25d53a2aa4cc29000000000000000000000000e9b59d7af13bf6d3f838da7f73c2e369802ea2110000000000000000000000001aab2207e31dff81240fc4976c301ab0a0e0da26")
	_, _, _, err = vmInstance.Call(caller, subGovernance, contractCodeBytes, vmCtx.GasLimit, big.NewInt(0))
	if err != nil {
		panic("Genesis cross contract create error:" + err.Error())
	}

}
