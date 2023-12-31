// Copyright 2020 The RangersProtocol Authors
// This file is part of the RangersProtocol library.
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
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	"com.tuntun.rangers/node/src/common"
	crypto "com.tuntun.rangers/node/src/eth_crypto"
	"com.tuntun.rangers/node/src/middleware/db"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/storage/account"
	"math"
	"math/big"
	"time"
)

//vm test functions

// Config is a basic type specifying certain configuration flags for running the VM
type testConfig struct {
	Difficulty  *big.Int
	Origin      common.Address
	Coinbase    common.Address
	BlockNumber *big.Int
	Time        *big.Int
	GasLimit    uint64
	GasPrice    *big.Int
	Value       *big.Int

	State           *account.AccountDB
	AccountDatabase account.AccountDatabase

	GetHashFn   func(n uint64) common.Hash
	CanTransfer CanTransferFunc
	Transfer    TransferFunc
}

// sets defaults on the config
func setDefaults(cfg *testConfig) {
	if cfg.Difficulty == nil {
		cfg.Difficulty = new(big.Int)
	}
	if cfg.Time == nil {
		cfg.Time = new(big.Int).SetUint64(uint64(time.Now().Unix()))
	}
	if cfg.GasLimit == 0 {
		cfg.GasLimit = math.MaxUint64
	}
	if cfg.GasPrice == nil {
		cfg.GasPrice = new(big.Int)
	}
	if cfg.Value == nil {
		cfg.Value = new(big.Int)
	}
	if cfg.BlockNumber == nil {
		cfg.BlockNumber = new(big.Int)
	}
	if cfg.State == nil {
		database, _ := db.NewLDBDatabase("test", 0, 0)
		cfg.AccountDatabase = account.NewDatabase(database)
		cfg.State, _ = account.NewAccountDB(common.Hash{}, cfg.AccountDatabase)
	}

	if cfg.GetHashFn == nil {
		cfg.GetHashFn = func(n uint64) common.Hash {
			return common.BytesToHash(crypto.Keccak256([]byte(new(big.Int).SetUint64(n).String())))
		}
	}

	if cfg.CanTransfer == nil {
		cfg.CanTransfer = CanTransfer
	}

	if cfg.Transfer == nil {
		cfg.Transfer = Transfer
	}
}

func mockInit() {
	common.Init(0, "1.ini", "mainnet")
	InitVM()
}

// Execute executes the code using the input as call data during the execution.
// It returns the EVM's return value, the new state and an error if it failed.
//
// Execute sets up an in-memory, temporary, environment for the execution of
// the given code. It makes sure that it's restored to its original state afterwards.
func mockExecute(code, input []byte, cfg *testConfig) ([]byte, *account.AccountDB, error) {
	if cfg == nil {
		cfg = new(testConfig)
	}
	setDefaults(cfg)

	if cfg.State == nil {
		database, _ := db.NewLDBDatabase("test", 0, 0)
		cfg.State, _ = account.NewAccountDB(common.Hash{}, account.NewDatabase(database))
	}
	var (
		address = common.BytesToAddress([]byte("contract"))
		vmenv   = mockEVM(cfg)
		sender  = AccountRef(cfg.Origin)
	)
	cfg.State.AddAddressToAccessList(cfg.Origin)
	cfg.State.AddAddressToAccessList(address)
	for _, addr := range vmenv.ActivePrecompiles() {
		cfg.State.AddAddressToAccessList(addr)
		cfg.State.AddAddressToAccessList(addr)
	}

	cfg.State.CreateAccount(address)
	// set the receiver's (the executing contract) code for execution.
	cfg.State.SetCode(address, code)
	// Call the code with the given configuration.
	ret, _, _, err := vmenv.Call(
		sender,
		common.BytesToAddress([]byte("contract")),
		input,
		cfg.GasLimit,
		cfg.Value,
	)

	return ret, cfg.State, err
}

// Create executes the code using the EVM create method
func mockCreate(input []byte, cfg *testConfig) ([]byte, common.Address, uint64, error) {
	if cfg == nil {
		cfg = new(testConfig)
	}
	setDefaults(cfg)

	if cfg.State == nil {
		database, _ := db.NewLDBDatabase("test", 0, 0)
		cfg.State, _ = account.NewAccountDB(common.Hash{}, account.NewDatabase(database))
	}
	var (
		vmenv  = mockEVM(cfg)
		sender = AccountRef(cfg.Origin)
	)
	//cfg.State.AddAddressToAccessList(cfg.Origin)
	//for _, addr := range vmenv.ActivePrecompiles() {
	//	cfg.State.AddAddressToAccessList(addr)
	//}

	// Call the code with the given configuration.
	code, address, leftOverGas, _, err := vmenv.Create(
		sender,
		input,
		cfg.GasLimit,
		cfg.Value,
	)
	return code, address, leftOverGas, err
}

// Call executes the code given by the contract's address. It will return the
// EVM's return value or an error if it failed.
//
// Call, unlike Execute, requires a config and also requires the State field to
// be set.
func mockCall(address common.Address, input []byte, cfg *testConfig) ([]byte, uint64, error) {
	setDefaults(cfg)
	vmenv := mockEVM(cfg)

	sender := mockContractRef{cfg.Origin}
	//cfg.State.AddAddressToAccessList(cfg.Origin)
	//cfg.State.AddAddressToAccessList(address)
	//for _, addr := range vmenv.ActivePrecompiles() {
	//	cfg.State.AddAddressToAccessList(addr)
	//}

	// Call the code with the given configuration.
	ret, leftOverGas, _, err := vmenv.Call(
		sender,
		address,
		input,
		cfg.GasLimit,
		cfg.Value,
	)
	return ret, leftOverGas, err
}

// Call executes the code given by the contract's address. It will return the
// EVM's return value or an error if it failed.
//
// Call, unlike Execute, requires a config and also requires the State field to
// be set.
func mockCallWithLogs(address common.Address, input []byte, cfg *testConfig) ([]byte, uint64, []*types.Log, error) {
	setDefaults(cfg)
	vmenv := mockEVM(cfg)

	sender := mockContractRef{cfg.Origin}
	//cfg.State.AddAddressToAccessList(cfg.Origin)
	//cfg.State.AddAddressToAccessList(address)
	//for _, addr := range vmenv.ActivePrecompiles() {
	//	cfg.State.AddAddressToAccessList(addr)
	//}

	// Call the code with the given configuration.
	return vmenv.Call(
		sender,
		address,
		input,
		cfg.GasLimit,
		cfg.Value,
	)
}

func mockEVM(cfg *testConfig) *EVM {
	context := Context{
		CanTransfer: cfg.CanTransfer,
		Transfer:    cfg.Transfer,
		GetHash:     cfg.GetHashFn,
		Origin:      cfg.Origin,
		Coinbase:    cfg.Coinbase,
		BlockNumber: cfg.BlockNumber,
		Time:        cfg.Time,
		Difficulty:  cfg.Difficulty,
		GasLimit:    cfg.GasLimit,
		GasPrice:    cfg.GasPrice,
	}
	return NewEVMWithNFT(context, cfg.State, cfg.State)
}

type mockContractRef struct {
	address common.Address
}

func (m mockContractRef) Address() common.Address {
	return m.address
}
