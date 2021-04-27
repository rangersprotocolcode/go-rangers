package service

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/db"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/vm"
	"com.tuntun.rocket/node/src/vm/crypto"
	"fmt"
	"math"
	"math/big"
	"os"
	"testing"
	"time"
)

func TestRocketProtocol(t *testing.T) {
	os.RemoveAll("storage0")
	os.RemoveAll("logs")
	os.Remove("1.ini")
	defer os.RemoveAll("logs")
	defer os.RemoveAll("storage0")
	defer os.Remove("1.ini")
	common.InitConf("1.ini")
	vm.InitVM()
	InitService()

	config := new(testConfig)
	setDefaults(config)
	defer log.Close()

	config.GasLimit = 3000000
	config.GasPrice = big.NewInt(1)

	contractCodeBytes := common.Hex2Bytes("608060405234801561001057600080fd5b506040518060400160405280600581526020017f73657449640000000000000000000000000000000000000000000000000000008152506040518060400160405280600781526020017f6e66744e616d65000000000000000000000000000000000000000000000000008152506040518060400160405280600981526020017f6e667453796d626f6c0000000000000000000000000000000000000000000000815250600ae0506040518060400160405280600581526020017f6f776e65720000000000000000000000000000000000000000000000000000008152506040518060400160405280600581526020017f61707049640000000000000000000000000000000000000000000000000000008152506040518060400160405280600581526020017f73657449640000000000000000000000000000000000000000000000000000008152506040518060400160405280600581526020017f6e6674496400000000000000000000000000000000000000000000000000000081525073dcad3a6d3569df655070ded06cb7a1b2ccd1d3af6040518060400160405280600481526020017f6461746100000000000000000000000000000000000000000000000000000000815250e1506040518060400160405280600581526020017f73657449640000000000000000000000000000000000000000000000000000008152506040518060400160405280600581526020017f6e6674496400000000000000000000000000000000000000000000000000000081525073dcad3a6d3569df655070ded06cb7a1b2ccd1d3afe2506040518060400160405280600581526020017f73657449640000000000000000000000000000000000000000000000000000008152506040518060400160405280600581526020017f6e667449640000000000000000000000000000000000000000000000000000008152506040518060400160405280600581526020017f61707049640000000000000000000000000000000000000000000000000000008152506040518060400160405280600b81526020017f7461726765744170704964000000000000000000000000000000000000000000815250e3506040518060400160405280600581526020017f73657449640000000000000000000000000000000000000000000000000000008152506040518060400160405280600581526020017f6e6674496400000000000000000000000000000000000000000000000000000081525073dcad3a6d3569df655070ded06cb7a1b2ccd1d3afe4506040518060400160405280600581526020017f736574496400000000000000000000000000000000000000000000000000000081525060405180604001604052806005")
	createResult, contractAddress, createLeftGas, createErr := mockCreate(contractCodeBytes, config)
	fmt.Printf("New create contract address:%s\n", contractAddress.GetHexString())
	fmt.Printf("New create contract createResult:%v,%d\n", createResult, len(createResult))
	fmt.Printf("New create contract costGas:%v,createErr:%v\n", config.GasLimit-createLeftGas, createErr)
}

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

	State       *account.AccountDB
	GetHashFn   func(n uint64) common.Hash
	CanTransfer vm.CanTransferFunc
	Transfer    vm.TransferFunc
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
		cfg.State, _ = account.NewAccountDB(common.Hash{}, account.NewDatabase(database))
	}

	if cfg.GetHashFn == nil {
		cfg.GetHashFn = func(n uint64) common.Hash {
			return common.BytesToHash(crypto.Keccak256([]byte(new(big.Int).SetUint64(n).String())))
		}
	}

	if cfg.CanTransfer == nil {
		cfg.CanTransfer = vm.CanTransfer
	}

	if cfg.Transfer == nil {
		cfg.Transfer = vm.Transfer
	}
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
		sender = vm.AccountRef(cfg.Origin)
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

func mockEVM(cfg *testConfig) *vm.EVM {
	context := vm.Context{
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
	return vm.NewEVMWithNFT(context, cfg.State, NFTManagerInstance, cfg.State)
}

