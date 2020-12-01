// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/db"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/vm/crypto"
	"fmt"
	"math"
	"math/big"
	"strings"
	"testing"
	"time"
)

// Config is a basic type specifying certain configuration flags for running
// the EVM
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
	CanTransfer CanTransferFunc
	Transfer    TransferFunc
}

// sets defaults on the config
func setDefaults(cfg *testConfig) {
	if cfg.Difficulty == nil {
		cfg.Difficulty = new(big.Int)
	}
	if cfg.Time == nil {
		cfg.Time = big.NewInt(time.Now().Unix())
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
		cfg.CanTransfer = CanTransfer
	}

	if cfg.Transfer == nil {
		cfg.Transfer = Transfer
	}
}

func mockInit() {
	common.InitConf("1.ini")
	InitVM()
}

type mockContractRef struct {
	address common.Address
}

func (m mockContractRef) Address() common.Address {
	return m.address
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
	return NewEVM(context, cfg.State)
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
	ret, _, err := vmenv.Call(
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
	cfg.State.AddAddressToAccessList(cfg.Origin)
	for _, addr := range vmenv.ActivePrecompiles() {
		cfg.State.AddAddressToAccessList(addr)
	}

	// Call the code with the given configuration.
	code, address, leftOverGas, err := vmenv.Create(
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
	cfg.State.AddAddressToAccessList(cfg.Origin)
	cfg.State.AddAddressToAccessList(address)
	for _, addr := range vmenv.ActivePrecompiles() {
		cfg.State.AddAddressToAccessList(addr)
	}

	// Call the code with the given configuration.
	ret, leftOverGas, err := vmenv.Call(
		sender,
		address,
		input,
		cfg.GasLimit,
		cfg.Value,
	)
	return ret, leftOverGas, err
}

func TestExampleExecute(t *testing.T) {
	mockInit()
	ret, _, err := mockExecute(common.Hex2Bytes("6060604052600a8060106000396000f360606040526008565b00"), nil, nil)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(ret)
	// Output:
	// [96 96 96 64 82 96 8 86 91 0]
}

func TestExecute(t *testing.T) {
	mockInit()
	ret, _, err := mockExecute([]byte{
		byte(PUSH1), 10,
		byte(PUSH1), 0,
		byte(MSTORE),
		byte(PUSH1), 32,
		byte(PUSH1), 0,
		byte(RETURN),
	}, nil, nil)
	if err != nil {
		t.Fatal("didn't expect error", err)
	}

	num := new(big.Int).SetBytes(ret)
	if num.Cmp(big.NewInt(10)) != 0 {
		t.Error("Expected 10, got", num)
	}
}

func TestCall(t *testing.T) {
	mockInit()
	database, _ := db.NewLDBDatabase("test", 0, 0)
	state, _ := account.NewAccountDB(common.Hash{}, account.NewDatabase(database))
	//address use big address 'aa' avoid precompile contract
	address := common.HexToAddress("0x0aa")
	state.SetCode(address, []byte{
		byte(PUSH1), 10,
		byte(PUSH1), 0,
		byte(MSTORE),
		byte(PUSH1), 32,
		byte(PUSH1), 0,
		byte(RETURN),
	})

	ret, _, err := mockCall(address, nil, &testConfig{State: state})
	if err != nil {
		t.Fatal("didn't expect error", err)
	}

	num := new(big.Int).SetBytes(ret)
	if num.Cmp(big.NewInt(10)) != 0 {
		t.Error("Expected 10, got", num)
	}
}

type stepCounter struct {
	inner *mdLogger
	steps int
}

func (s *stepCounter) CaptureStart(from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) error {
	return nil
}

func (s *stepCounter) CaptureState(env *EVM, pc uint64, op OpCode, gas, cost uint64, memory *Memory, stack *Stack, rStack *ReturnStack, rData []byte, contract *Contract, depth int, err error) error {
	s.steps++
	// Enable this for more output
	//s.inner.CaptureState(env, pc, op, gas, cost, memory, stack, rStack, contract, depth, err)
	return nil
}

func (s *stepCounter) CaptureFault(env *EVM, pc uint64, op OpCode, gas, cost uint64, memory *Memory, stack *Stack, rStack *ReturnStack, contract *Contract, depth int, err error) error {
	return nil
}

func (s *stepCounter) CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) error {
	return nil
}

func TestJumpSub1024Limit(t *testing.T) {
	mockInit()
	database, _ := db.NewLDBDatabase("test", 0, 0)
	state, _ := account.NewAccountDB(common.Hash{}, account.NewDatabase(database))
	//address use big address 'aa' avoid precompile contract
	address := common.HexToAddress("0x0aa")
	// Code is
	// 0 beginsub
	// 1 push 0
	// 3 jumpsub
	//
	// The code recursively calls itself. It should error when the returns-stack
	// grows above 1023
	state.SetCode(address, []byte{
		byte(PUSH1), 3,
		byte(JUMPSUB),
		byte(BEGINSUB),
		byte(PUSH1), 3,
		byte(JUMPSUB),
	})
	tracer := &stepCounter{inner: NewMarkdownLogger(nil, log.GetLoggerByIndex(log.VMLogConfig, ""))}
	vmTracer = tracer

	// Enable 2315
	_, _, err := mockCall(address, nil, &testConfig{State: state, GasLimit: 20000})
	exp := "return stack limit reached"
	if err.Error() != exp {
		t.Fatalf("expected %v, got %v", exp, err)
	}
	if exp, got := 2048, tracer.steps; exp != got {
		t.Fatalf("expected %d steps, got %d", exp, got)
	}
}

func TestReturnSubShallow(t *testing.T) {
	mockInit()
	database, _ := db.NewLDBDatabase("test", 0, 0)
	state, _ := account.NewAccountDB(common.Hash{}, account.NewDatabase(database))
	//address use big address 'aa' avoid precompile contract
	address := common.HexToAddress("0x0aa")
	// The code does returnsub without having anything on the returnstack.
	// It should not panic, but just fail after one step
	state.SetCode(address, []byte{
		byte(PUSH1), 5,
		byte(JUMPSUB),
		byte(RETURNSUB),
		byte(PC),
		byte(BEGINSUB),
		byte(RETURNSUB),
		byte(PC),
	})
	tracer := &stepCounter{inner: NewMarkdownLogger(nil, log.GetLoggerByIndex(log.VMLogConfig, ""))}
	vmTracer = tracer

	// Enable 2315
	_, _, err := mockCall(address, nil, &testConfig{State: state, GasLimit: 10000})

	exp := "invalid retsub"
	if err.Error() != exp {
		t.Fatalf("expected %v, got %v", exp, err)
	}
	if exp, got := 4, tracer.steps; exp != got {
		t.Fatalf("expected %d steps, got %d", exp, got)
	}
}

// Iterator for disassembled EVM instructions
type instructionIterator struct {
	code    []byte
	pc      uint64
	arg     []byte
	op      OpCode
	error   error
	started bool
}

// Create a new instruction iterator.
func newInstructionIterator(code []byte) *instructionIterator {
	it := new(instructionIterator)
	it.code = code
	return it
}

// Returns true if there is a next instruction and moves on.
func (it *instructionIterator) Next() bool {
	if it.error != nil || uint64(len(it.code)) <= it.pc {
		// We previously reached an error or the end.
		return false
	}

	if it.started {
		// Since the iteration has been already started we move to the next instruction.
		if it.arg != nil {
			it.pc += uint64(len(it.arg))
		}
		it.pc++
	} else {
		// We start the iteration from the first instruction.
		it.started = true
	}

	if uint64(len(it.code)) <= it.pc {
		// We reached the end.
		return false
	}

	it.op = OpCode(it.code[it.pc])
	if it.op.IsPush() {
		a := uint64(it.op) - uint64(PUSH1) + 1
		u := it.pc + 1 + a
		if uint64(len(it.code)) <= it.pc || uint64(len(it.code)) < u {
			it.error = fmt.Errorf("incomplete push instruction at %v", it.pc)
			return false
		}
		it.arg = it.code[it.pc+1 : u]
	} else {
		it.arg = nil
	}
	return true
}

// Returns any error that may have been encountered.
func (it *instructionIterator) Error() error {
	return it.error
}

// Returns the PC of the current instruction.
func (it *instructionIterator) PC() uint64 {
	return it.pc
}

// Returns the opcode of the current instruction.
func (it *instructionIterator) Op() OpCode {
	return it.op
}

// Returns the argument of the current instruction.
func (it *instructionIterator) Arg() []byte {
	return it.arg
}

// TestEip2929Cases contains various testcases that are used for
// EIP-2929 about gas repricings
func TestEip2929Cases(t *testing.T) {
	mockInit()

	id := 1
	prettyPrint := func(comment string, code []byte) {

		instrs := make([]string, 0)
		it := newInstructionIterator(code)
		for it.Next() {
			if it.Arg() != nil && 0 < len(it.Arg()) {
				instrs = append(instrs, fmt.Sprintf("%v 0x%x", it.Op(), it.Arg()))
			} else {
				instrs = append(instrs, fmt.Sprintf("%v", it.Op()))
			}
		}
		ops := strings.Join(instrs, ", ")
		fmt.Printf("### Case %d\n\n", id)
		id++
		fmt.Printf("%v\n\nBytecode: \n```\n0x%x\n```\nOperations: \n```\n%v\n```\n\n",
			comment,
			code, ops)
		mockExecute(code, nil, nil)
	}

	{ // First eip testcase
		code := []byte{
			// Three checks against a precompile
			byte(PUSH1), 1, byte(EXTCODEHASH), byte(POP),
			byte(PUSH1), 2, byte(EXTCODESIZE), byte(POP),
			byte(PUSH1), 3, byte(BALANCE), byte(POP),
			// Three checks against a non-precompile
			byte(PUSH1), 0xf1, byte(EXTCODEHASH), byte(POP),
			byte(PUSH1), 0xf2, byte(EXTCODESIZE), byte(POP),
			byte(PUSH1), 0xf3, byte(BALANCE), byte(POP),
			// Same three checks (should be cheaper)
			byte(PUSH1), 0xf2, byte(EXTCODEHASH), byte(POP),
			byte(PUSH1), 0xf3, byte(EXTCODESIZE), byte(POP),
			byte(PUSH1), 0xf1, byte(BALANCE), byte(POP),
			// Check the origin, and the 'this'
			byte(ORIGIN), byte(BALANCE), byte(POP),
			byte(ADDRESS), byte(BALANCE), byte(POP),

			byte(STOP),
		}
		prettyPrint("This checks `EXT`(codehash,codesize,balance) of precompiles, which should be `100`, "+
			"and later checks the same operations twice against some non-precompiles. "+
			"Those are cheaper second time they are accessed. Lastly, it checks the `BALANCE` of `origin` and `this`.", code)
	}

	{ // EXTCODECOPY
		code := []byte{
			// extcodecopy( 0xff,0,0,0,0)
			byte(PUSH1), 0x00, byte(PUSH1), 0x00, byte(PUSH1), 0x00, //length, codeoffset, memoffset
			byte(PUSH1), 0xff, byte(EXTCODECOPY),
			// extcodecopy( 0xff,0,0,0,0)
			byte(PUSH1), 0x00, byte(PUSH1), 0x00, byte(PUSH1), 0x00, //length, codeoffset, memoffset
			byte(PUSH1), 0xff, byte(EXTCODECOPY),
			// extcodecopy( this,0,0,0,0)
			byte(PUSH1), 0x00, byte(PUSH1), 0x00, byte(PUSH1), 0x00, //length, codeoffset, memoffset
			byte(ADDRESS), byte(EXTCODECOPY),

			byte(STOP),
		}
		prettyPrint("This checks `extcodecopy( 0xff,0,0,0,0)` twice, (should be expensive first time), "+
			"and then does `extcodecopy( this,0,0,0,0)`.", code)
	}

	{ // SLOAD + SSTORE
		code := []byte{

			// Add slot `0x1` to access list
			byte(PUSH1), 0x01, byte(SLOAD), byte(POP), // SLOAD( 0x1) (add to access list)
			// Write to `0x1` which is already in access list
			byte(PUSH1), 0x11, byte(PUSH1), 0x01, byte(SSTORE), // SSTORE( loc: 0x01, val: 0x11)
			// Write to `0x2` which is not in access list
			byte(PUSH1), 0x11, byte(PUSH1), 0x02, byte(SSTORE), // SSTORE( loc: 0x02, val: 0x11)
			// Write again to `0x2`
			byte(PUSH1), 0x11, byte(PUSH1), 0x02, byte(SSTORE), // SSTORE( loc: 0x02, val: 0x11)
			// Read slot in access list (0x2)
			byte(PUSH1), 0x02, byte(SLOAD), // SLOAD( 0x2)
			// Read slot in access list (0x1)
			byte(PUSH1), 0x01, byte(SLOAD), // SLOAD( 0x1)
		}
		prettyPrint("This checks `sload( 0x1)` followed by `sstore(loc: 0x01, val:0x11)`, then 'naked' sstore:"+
			"`sstore(loc: 0x02, val:0x11)` twice, and `sload(0x2)`, `sload(0x1)`. ", code)
	}
	{ // Call variants
		code := []byte{
			// identity precompile
			byte(PUSH1), 0x0, byte(DUP1), byte(DUP1), byte(DUP1), byte(DUP1),
			byte(PUSH1), 0x04, byte(PUSH1), 0x0, byte(CALL), byte(POP),

			// random account - call 1
			byte(PUSH1), 0x0, byte(DUP1), byte(DUP1), byte(DUP1), byte(DUP1),
			byte(PUSH1), 0xff, byte(PUSH1), 0x0, byte(CALL), byte(POP),

			// random account - call 2
			byte(PUSH1), 0x0, byte(DUP1), byte(DUP1), byte(DUP1), byte(DUP1),
			byte(PUSH1), 0xff, byte(PUSH1), 0x0, byte(STATICCALL), byte(POP),
		}
		prettyPrint("This calls the `identity`-precompile (cheap), then calls an account (expensive) and `staticcall`s the same"+
			"account (cheap)", code)
	}
}

type dummyHeader struct {
	Number     *big.Int
	ParentHash common.Hash
	Time       uint64
	Difficulty *big.Int
	GasLimit   uint64
}

type dummyChain struct {
	counter int
}

// GetHeader returns the hash corresponding to their hash.
func (d *dummyChain) getHeader(h common.Hash, n uint64) *dummyHeader {
	d.counter++
	parentHash := common.Hash{}
	s := common.LeftPadBytes(big.NewInt(int64(n-1)).Bytes(), 32)
	copy(parentHash[:], s)

	//parentHash := common.Hash{byte(n - 1)}
	//fmt.Printf("GetHeader(%x, %d) => header with parent %x\n", h, n, parentHash)
	return fakeHeader(n, parentHash)
}

func fakeHeader(n uint64, parentHash common.Hash) *dummyHeader {
	header := dummyHeader{
		Number:     big.NewInt(int64(n)),
		ParentHash: parentHash,
		Time:       1000,
		Difficulty: big.NewInt(0),
		GasLimit:   100000,
	}
	return &header
}

// GetHashFn returns a GetHashFunc which retrieves header hashes by number
func mockGetHashFn(ref *dummyHeader, chain *dummyChain) func(n uint64) common.Hash {
	// Cache will initially contain [refHash.parent],
	// Then fill up with [refHash.p, refHash.pp, refHash.ppp, ...]
	var cache []common.Hash

	return func(n uint64) common.Hash {
		// If there's no hash cache yet, make one
		if len(cache) == 0 {
			cache = append(cache, ref.ParentHash)
		}
		if idx := ref.Number.Uint64() - n - 1; idx < uint64(len(cache)) {
			return cache[idx]
		}
		// No luck in the cache, but we can start iterating from the last element we already know
		lastKnownHash := cache[len(cache)-1]
		lastKnownNumber := ref.Number.Uint64() - uint64(len(cache))

		for {
			header := chain.getHeader(lastKnownHash, lastKnownNumber)
			if header == nil {
				break
			}
			cache = append(cache, header.ParentHash)
			lastKnownHash = header.ParentHash
			lastKnownNumber = header.Number.Uint64() - 1
			if n == lastKnownNumber {
				return lastKnownHash
			}
		}
		return common.Hash{}
	}
}

// TestBlockhash tests the blockhash operation. It's a bit special, since it internally
// requires access to a chain reader.
func TestBlockhash(t *testing.T) {
	mockInit()

	// Current head
	n := uint64(1000)
	parentHash := common.Hash{}
	s := common.LeftPadBytes(big.NewInt(int64(n-1)).Bytes(), 32)
	copy(parentHash[:], s)
	header := fakeHeader(n, parentHash)

	// This is the contract we're using. It requests the blockhash for current num (should be all zeroes),
	// then iteratively fetches all blockhashes back to n-260.
	// It returns
	// 1. the first (should be zero)
	// 2. the second (should be the parent hash)
	// 3. the last non-zero hash
	// By making the chain reader return hashes which correlate to the number, we can
	// verify that it obtained the right hashes where it should

	/*

		pragma solidity ^0.5.3;
		contract Hasher{

			function test() public view returns (bytes32, bytes32, bytes32){
				uint256 x = block.number;
				bytes32 first;
				bytes32 last;
				bytes32 zero;
				zero = blockhash(x); // Should be zeroes
				first = blockhash(x-1);
				for(uint256 i = 2 ; i < 260; i++){
					bytes32 hash = blockhash(x - i);
					if (uint256(hash) != 0){
						last = hash;
					}
				}
				return (zero, first, last);
			}
		}

	*/
	// The contract above
	data := common.Hex2Bytes("6080604052348015600f57600080fd5b50600436106045576000357c010000000000000000000000000000000000000000000000000000000090048063f8a8fd6d14604a575b600080fd5b60506074565b60405180848152602001838152602001828152602001935050505060405180910390f35b600080600080439050600080600083409050600184034092506000600290505b61010481101560c35760008186034090506000816001900414151560b6578093505b5080806001019150506094565b508083839650965096505050505090919256fea165627a7a72305820462d71b510c1725ff35946c20b415b0d50b468ea157c8c77dff9466c9cb85f560029")
	// The method call to 'test()'
	input := common.Hex2Bytes("f8a8fd6d")
	chain := &dummyChain{}
	ret, _, err := mockExecute(data, input, &testConfig{
		GetHashFn:   mockGetHashFn(header, chain),
		BlockNumber: new(big.Int).Set(header.Number),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(ret) != 96 {
		t.Fatalf("expected returndata to be 96 bytes, got %d", len(ret))
	}

	zero := new(big.Int).SetBytes(ret[0:32])
	first := new(big.Int).SetBytes(ret[32:64])
	last := new(big.Int).SetBytes(ret[64:96])
	if zero.BitLen() != 0 {
		t.Fatalf("expected zeroes, got %x", ret[0:32])
	}
	if first.Uint64() != 999 {
		t.Fatalf("second block should be 999, got %d (%x)", first, ret[32:64])
	}
	if last.Uint64() != 744 {
		t.Fatalf("last block should be 744, got %d (%x)", last, ret[64:96])
	}
	if exp, got := 255, chain.counter; exp != got {
		t.Errorf("suboptimal; too much chain iteration, expected %d, got %d", exp, got)
	}
}
