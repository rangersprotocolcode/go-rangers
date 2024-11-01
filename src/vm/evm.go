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
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/storage/account"
	"errors"
	"github.com/holiman/uint256"
	"golang.org/x/crypto/sha3"
	"math/big"
	"sync/atomic"
)

// emptyCodeHash is used by create to ensure deployment is disallowed to already
// deployed contract addresses (relevant after the account abstraction).
var (
	emptyCodeHash       = crypto.Keccak256Hash(nil)
	errSubChainNoCreate = errors.New("not allowed to create contract")
)

type (
	// CanTransferFunc is the signature of a transfer guard function
	CanTransferFunc func(StateDB, common.Address, *big.Int) bool
	// TransferFunc is the signature of a transfer function
	TransferFunc func(StateDB, common.Address, common.Address, *big.Int)
	// GetHashFunc returns the n'th block hash in the blockchain
	// and is used by the BLOCKHASH EVM op code.
	GetHashFunc func(uint64) common.Hash
)

// ActivePrecompiles returns the addresses of the precompiles enabled with the current
// configuration
func (evm *EVM) ActivePrecompiles() []common.Address {
	return PrecompiledAddresses
}

func (evm *EVM) precompile(addr common.Address) (PrecompiledContract, bool) {
	p, ok := PrecompiledContracts[addr]
	return p, ok
}

// run runs the given contract and takes care of running precompiles with a fallback to the byte code interpreter.
func run(evm *EVM, contract *Contract, input []byte, readOnly bool) ([]byte, []*types.Log, error) {
	for _, interpreter := range evm.interpreters {
		if interpreter.CanRun(contract.Code) {
			if evm.interpreter != interpreter {
				// Ensure that the interpreter pointer is set back
				// to its current value upon return.
				defer func(i Interpreter) {
					evm.interpreter = i
				}(evm.interpreter)
				evm.interpreter = interpreter
			}
			return interpreter.Run(contract, input, readOnly)
		}
	}
	return nil, nil, errors.New("no compatible interpreter")
}

// Context provides the EVM with auxiliary information. Once provided
// it shouldn't be modified.
type Context struct {
	// CanTransfer returns whether the account contains
	// sufficient ether to transfer the value
	CanTransfer CanTransferFunc
	// Transfer transfers ether from one account to the other
	Transfer TransferFunc
	// GetHash returns the hash corresponding to n
	GetHash GetHashFunc

	// Message information
	Origin   common.Address // Provides information for ORIGIN
	GasPrice *big.Int       // Provides information for GASPRICE

	// Block information
	Coinbase    common.Address // Provides information for COINBASE
	GasLimit    uint64         // Provides information for GASLIMIT
	BlockNumber *big.Int       // Provides information for NUMBER
	Time        *big.Int       // Provides information for TIME
	Difficulty  *big.Int       // Provides information for DIFFICULTY
}

// EVM is the Ethereum Virtual Machine base object and provides
// the necessary tools to run a contract on the given state with
// the provided context. It should be noted that any error
// generated through any of the calls should be considered a
// revert-state-and-consume-all-gas operation, no checks on
// specific errors should ever be performed. The interpreter makes
// sure that any errors generated are to be considered faulty code.
//
// The EVM should never be reused and is not thread safe.
type EVM struct {
	// Context provides auxiliary blockchain related information
	Context
	// StateDB gives access to the underlying state
	StateDB StateDB
	// Depth is the current call stack
	depth int

	chainID *big.Int

	tracer Tracer
	// global (to this context) ethereum virtual machine
	// used throughout the execution of the tx.
	interpreters []Interpreter
	interpreter  Interpreter
	// abort is used to abort the EVM calling operations
	// NOTE: must be set atomically
	abort int32
	// callGasTemp holds the gas available for the current call. This is needed because the
	// available gas is calculated in gasCall* according to the 63/64 rule and later
	// applied in opCall*.
	callGasTemp uint64

	accountDB *account.AccountDB
}

// NewEVM returns a new EVM. The returned EVM is not thread safe and should
// only ever be used *once*.
func NewEVM(ctx Context, statedb StateDB) *EVM {
	evm := &EVM{
		Context:      ctx,
		StateDB:      statedb,
		interpreters: make([]Interpreter, 0, 1),
	}
	if ctx.BlockNumber != nil {
		evm.chainID = common.GetChainId(ctx.BlockNumber.Uint64())
	}

	config := LogConfig{false, false, false, false}
	evm.tracer = NewStructLogger(&config, logger)

	// vmConfig.EVMInterpreter will be used by EVM-C, it won't be checked here
	// as we always want to have the built-in EVM as the failover option.
	evm.interpreters = append(evm.interpreters, NewEVMInterpreter(evm))
	evm.interpreter = evm.interpreters[0]

	return evm
}

// NewEVM returns a new EVM. The returned EVM is not thread safe and should
// only ever be used *once*.
func NewEVMWithNFT(ctx Context, statedb StateDB, accountDB *account.AccountDB) *EVM {
	evm := NewEVM(ctx, statedb)
	evm.accountDB = accountDB

	return evm
}

// Cancel cancels any running EVM operation. This may be called concurrently and
// it's safe to be called multiple times.
func (evm *EVM) Cancel() {
	atomic.StoreInt32(&evm.abort, 1)
}

// Cancelled returns true if Cancel has been called
func (evm *EVM) Cancelled() bool {
	return atomic.LoadInt32(&evm.abort) == 1
}

// Interpreter returns the current interpreter
func (evm *EVM) Interpreter() Interpreter {
	return evm.interpreter
}

// Call executes the contract associated with the addr with the given input as
// parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
func (evm *EVM) Call(caller ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, logs []*types.Log, err error) {
	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(CallCreateDepth) {
		return nil, gas, nil, ErrDepth
	}
	// Fail if we're trying to transfer more than the available balance
	if value.Sign() != 0 && !evm.Context.CanTransfer(evm.StateDB, caller.Address(), value) {
		return nil, gas, nil, ErrInsufficientBalance
	}
	snapshot := evm.StateDB.Snapshot()
	p, isPrecompile := evm.precompile(addr)

	if !evm.StateDB.Exist(addr) {
		if !isPrecompile && value.Sign() == 0 {
			// Calling a non existing account, don't do anything, but ping the tracer
			//if evm.vmConfig.Debug && evm.depth == 0
			//if evm.depth == 0 {
			//	evm.tracer.CaptureStart(caller.Address(), addr, false, input, gas, value)
			//	evm.tracer.CaptureEnd(ret, 0, 0, nil)
			//}
			return nil, gas, nil, nil
		}
		evm.StateDB.CreateAccount(addr)
	}
	evm.Transfer(evm.StateDB, caller.Address(), addr, value)

	if isPrecompile {
		ret, gas, err = RunPrecompiledContract(p, input, gas)
	} else {
		// Initialise a new contract and set the code that is to be used by the EVM.
		// The contract is a scoped environment for this execution context only.
		code := evm.StateDB.GetCode(addr)
		if len(code) == 0 {
			ret, err = nil, nil // gas is unchanged
		} else {
			addrCopy := addr
			// If the account has no code, we can abort here
			// The depth-check is already done, and precompiles handled above
			contract := NewContract(caller, AccountRef(addrCopy), value, gas)
			contract.SetCallCode(&addrCopy, evm.StateDB.GetCodeHash(addrCopy), code)
			ret, logs, err = run(evm, contract, input, false)
			gas = contract.Gas
		}
	}
	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != ErrExecutionReverted {
			gas = 0
		}
	}
	return ret, gas, logs, err
}

// CallCode executes the contract associated with the addr with the given input
// as parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
//
// CallCode differs from Call in the sense that it executes the given address'
// code with the caller as context.
func (evm *EVM) CallCode(caller ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, logs []*types.Log, err error) {
	//if evm.vmConfig.NoRecursion && evm.depth > 0 {
	//	return nil, gas, nil
	//}

	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(CallCreateDepth) {
		return nil, gas, nil, ErrDepth
	}
	// Fail if we're trying to transfer more than the available balance
	// Note although it's noop to transfer X ether to caller itself. But
	// if caller doesn't have enough balance, it would be an error to allow
	// over-charging itself. So the check here is necessary.
	if !evm.Context.CanTransfer(evm.StateDB, caller.Address(), value) {
		return nil, gas, nil, ErrInsufficientBalance
	}
	var snapshot = evm.StateDB.Snapshot()

	// It is allowed to call precompiles, even via delegatecall
	if p, isPrecompile := evm.precompile(addr); isPrecompile {
		ret, gas, err = RunPrecompiledContract(p, input, gas)
	} else {
		addrCopy := addr
		// Initialise a new contract and set the code that is to be used by the EVM.
		// The contract is a scoped environment for this execution context only.
		contract := NewContract(caller, AccountRef(caller.Address()), value, gas)
		contract.SetCallCode(&addrCopy, evm.StateDB.GetCodeHash(addrCopy), evm.StateDB.GetCode(addrCopy))
		ret, logs, err = run(evm, contract, input, false)
		gas = contract.Gas
	}
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != ErrExecutionReverted {
			gas = 0
		}
	}
	return ret, gas, nil, err
}

// DelegateCall executes the contract associated with the addr with the given input
// as parameters. It reverses the state in case of an execution error.
//
// DelegateCall differs from CallCode in the sense that it executes the given address'
// code with the caller as context and the caller is set to the caller of the caller.
func (evm *EVM) DelegateCall(caller ContractRef, addr common.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, logs []*types.Log, err error) {

	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(CallCreateDepth) {
		return nil, gas, nil, ErrDepth
	}
	var snapshot = evm.StateDB.Snapshot()

	// It is allowed to call precompiles, even via delegatecall
	if p, isPrecompile := evm.precompile(addr); isPrecompile {
		ret, gas, err = RunPrecompiledContract(p, input, gas)
	} else {
		addrCopy := addr
		// Initialise a new contract and make initialise the delegate values
		contract := NewContract(caller, AccountRef(caller.Address()), nil, gas).AsDelegate()
		contract.SetCallCode(&addrCopy, evm.StateDB.GetCodeHash(addrCopy), evm.StateDB.GetCode(addrCopy))
		ret, logs, err = run(evm, contract, input, false)
		gas = contract.Gas
	}
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != ErrExecutionReverted {
			gas = 0
		}
	}
	return ret, gas, logs, err
}

// StaticCall executes the contract associated with the addr with the given input
// as parameters while disallowing any modifications to the state during the call.
// Opcodes that attempt to perform such modifications will result in exceptions
// instead of performing the modifications.
func (evm *EVM) StaticCall(caller ContractRef, addr common.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, logs []*types.Log, err error) {
	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(CallCreateDepth) {
		return nil, gas, nil, ErrDepth
	}
	// We take a snapshot here. This is a bit counter-intuitive, and could probably be skipped.
	// However, even a staticcall is considered a 'touch'. On mainnet, static calls were introduced
	// after all empty accounts were deleted, so this is not required. However, if we omit this,
	// then certain tests start failing; stRevertTest/RevertPrecompiledTouchExactOOG.json.
	// We could change this, but for now it's left for legacy reasons
	var snapshot = evm.StateDB.Snapshot()

	// We do an AddBalance of zero here, just in order to trigger a touch.
	// This doesn't matter on Mainnet, where all empties are gone at the time of Byzantium,
	// but is the correct thing to do and matters on other networks, in tests, and potential
	// future scenarios
	evm.StateDB.AddBalance(addr, big0)

	if p, isPrecompile := evm.precompile(addr); isPrecompile {
		ret, gas, err = RunPrecompiledContract(p, input, gas)
	} else {
		// At this point, we use a copy of address. If we don't, the go compiler will
		// leak the 'contract' to the outer scope, and make allocation for 'contract'
		// even if the actual execution ends on RunPrecompiled above.
		addrCopy := addr
		// Initialise a new contract and set the code that is to be used by the EVM.
		// The contract is a scoped environment for this execution context only.
		contract := NewContract(caller, AccountRef(addrCopy), new(big.Int), gas)
		contract.SetCallCode(&addrCopy, evm.StateDB.GetCodeHash(addrCopy), evm.StateDB.GetCode(addrCopy))
		// When an error was returned by the EVM or when setting the creation code
		// above we revert to the snapshot and consume any gas remaining. Additionally
		// when we're in Homestead this also counts for code storage gas errors.
		ret, logs, err = run(evm, contract, input, true)
		gas = contract.Gas
	}
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != ErrExecutionReverted {
			gas = 0
		}
	}
	return ret, gas, logs, err
}

type codeAndHash struct {
	code []byte
	hash common.Hash
}

func (c *codeAndHash) Hash() common.Hash {
	if c.hash == (common.Hash{}) {
		c.hash = crypto.Keccak256Hash(c.code)
	}
	return c.hash
}

// create creates a new contract using code as deployment code.
func (evm *EVM) create(caller ContractRef, codeAndHash *codeAndHash, gas uint64, value *big.Int, address common.Address) ([]byte, common.Address, uint64, []*types.Log, error) {
	// check owner when subnet
	if common.IsSub() && 0 != evm.BlockNumber.Uint64() {
		data := [64]byte{}
		copy(data[12:], caller.Address().Bytes())
		copy(data[64-len(common.CreateWhiteListPostion):], common.CreateWhiteListPostion)
		hasher := sha3.NewLegacyKeccak256().(common.KeccakState)
		hasher.Write(data[:])
		key := [32]byte{}
		hasher.Read(key[:])

		value := evm.accountDB.GetData(common.CreateWhiteListAddr, key[:])
		if 0 == len(value) || 1 != value[len(value)-1] {
			return nil, common.Address{}, 0, nil, errSubChainNoCreate
		}
	}

	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	if evm.depth > int(CallCreateDepth) {
		return nil, common.Address{}, gas, nil, ErrDepth
	}
	if !evm.CanTransfer(evm.StateDB, caller.Address(), value) {
		return nil, common.Address{}, gas, nil, ErrInsufficientBalance
	}
	nonce := evm.StateDB.GetNonce(caller.Address())
	if !common.IsProposal006() || common.IsProposal007() {
		evm.StateDB.SetNonce(caller.Address(), nonce+1)
	}
	// We add this to the access list _before_ taking a snapshot. Even if the creation fails,
	// the access-list change should not be rolled back
	evm.StateDB.AddAddressToAccessList(address)

	// Ensure there's no existing contract already at the designated address
	contractHash := evm.StateDB.GetCodeHash(address)
	if evm.StateDB.GetNonce(address) != 0 || (contractHash != (common.Hash{}) && contractHash != emptyCodeHash) {
		return nil, common.Address{}, 0, nil, ErrContractAddressCollision
	}
	// Create a new account on the state
	snapshot := evm.StateDB.Snapshot()
	evm.StateDB.CreateAccount(address)
	evm.StateDB.SetNonce(address, 1)

	evm.Transfer(evm.StateDB, caller.Address(), address, value)

	// Initialise a new contract and set the code that is to be used by the EVM.
	// The contract is a scoped environment for this execution context only.
	contract := NewContract(caller, AccountRef(address), value, gas)
	contract.SetCodeOptionalHash(&address, codeAndHash)

	ret, logs, err := run(evm, contract, nil, false)

	// check whether the max code size has been exceeded
	maxCodeSizeExceeded := len(ret) > MaxCodeSize
	// if the contract creation ran successfully and no errors were returned
	// calculate the gas required to store the code. If the code could not
	// be stored due to not enough gas set an error and let it be handled
	// by the error checking condition below.
	if err == nil && !maxCodeSizeExceeded {
		createDataGas := uint64(len(ret)) * CreateDataGas
		if common.IsProposal026() {
			createDataGas = createDataGas * common.GasMagnification
		}
		if contract.UseGas(createDataGas) {
			evm.StateDB.SetCode(address, ret)
			logger.Debugf("Create data gas:%d", createDataGas)
		} else {
			err = ErrCodeStoreOutOfGas
		}
	}

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if maxCodeSizeExceeded || (err != nil && err != ErrCodeStoreOutOfGas) {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != ErrExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	// Assign err if contract code size exceeds the max while the err is still empty.
	if maxCodeSizeExceeded && err == nil {
		err = ErrMaxCodeSizeExceeded
	}
	return ret, address, contract.Gas, logs, err

}

// Create creates a new contract using code as deployment code.
func (evm *EVM) Create(caller ContractRef, code []byte, gas uint64, value *big.Int) (ret []byte, contractAddr common.Address, leftOverGas uint64, logs []*types.Log, err error) {
	contractAddr = crypto.CreateAddress(caller.Address(), evm.StateDB.GetNonce(caller.Address()))
	return evm.create(caller, &codeAndHash{code: code}, gas, value, contractAddr)
}

// Create2 creates a new contract using code as deployment code.
//
// The different between Create2 with Create is Create2 uses sha3(0xff ++ msg.sender ++ salt ++ sha3(init_code))[12:]
// instead of the usual sender-and-nonce-hash as the address where the contract is initialized at.
func (evm *EVM) Create2(caller ContractRef, code []byte, gas uint64, endowment *big.Int, salt *uint256.Int) (ret []byte, contractAddr common.Address, leftOverGas uint64, logs []*types.Log, err error) {
	codeAndHash := &codeAndHash{code: code}
	contractAddr = crypto.CreateAddress2(caller.Address(), common.Hash(salt.Bytes32()), codeAndHash.Hash().Bytes())
	return evm.create(caller, codeAndHash, gas, endowment, contractAddr)
}

// AuthCall executes the contract associated with the addr with the given input
// as parameters.
// transfer and gas will be paid by sponsor
// It reverses the state in case of an execution error.
func (evm *EVM) AuthCall(sponsor common.Address, caller ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, logs []*types.Log, err error) {
	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(CallCreateDepth) {
		return nil, gas, nil, ErrDepth
	}
	// Fail if we're trying to transfer more than the available balance
	if value.Sign() != 0 && !evm.Context.CanTransfer(evm.StateDB, sponsor, value) {
		return nil, gas, nil, ErrInsufficientBalance
	}
	//authcall caller's nonce increase
	nonce := evm.StateDB.GetNonce(caller.Address())
	evm.StateDB.SetNonce(caller.Address(), nonce+1)

	snapshot := evm.StateDB.Snapshot()
	p, isPrecompile := evm.precompile(addr)

	if !evm.StateDB.Exist(addr) {
		if !isPrecompile && value.Sign() == 0 {
			// Calling a non existing account, don't do anything, but ping the tracer
			//if evm.vmConfig.Debug && evm.depth == 0
			return nil, gas, nil, nil
		}
		evm.StateDB.CreateAccount(addr)
	}
	evm.Transfer(evm.StateDB, sponsor, addr, value)

	if isPrecompile {
		ret, gas, err = RunPrecompiledContract(p, input, gas)
	} else {
		// Initialise a new contract and set the code that is to be used by the EVM.
		// The contract is a scoped environment for this execution context only.
		code := evm.StateDB.GetCode(addr)
		if len(code) == 0 {
			ret, err = nil, nil // gas is unchanged
		} else {
			addrCopy := addr
			// If the account has no code, we can abort here
			// The depth-check is already done, and precompiles handled above
			contract := NewContract(caller, AccountRef(addrCopy), value, gas)
			contract.SetCallCode(&addrCopy, evm.StateDB.GetCodeHash(addrCopy), code)
			ret, logs, err = run(evm, contract, input, false)
			gas = contract.Gas
		}
	}
	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != ErrExecutionReverted {
			gas = 0
		}
	}
	return ret, gas, logs, err
}
