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

type (
	executionFunc func(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error)
	gasFunc       func(*EVM, *Contract, *Stack, *Memory, uint64) (uint64, error) // last parameter is the requested memory size as a uint64
	// memorySizeFunc returns the required size, and whether the operation overflowed a uint64
	memorySizeFunc func(*Stack) (size uint64, overflow bool)
)

type operation struct {
	// execute is the operation function
	execute     executionFunc
	constantGas uint64
	dynamicGas  gasFunc
	// minStack tells how many stack items are required
	minStack int
	// maxStack specifies the max length the stack can have for this operation
	// to not overflow the stack.
	maxStack int

	// memorySize returns the memory size required for the operation
	memorySize memorySizeFunc

	halts   bool // indicates whether the operation should halt further execution
	jumps   bool // indicates whether the program counter should not increment
	writes  bool // determines whether this a state modifying operation
	reverts bool // determines whether the operation reverts state (implicitly halts)
	returns bool // determines whether the operations sets the return data content
}

// JumpTable contains the EVM opcodes supported at a given fork.
type JumpTable [256]*operation

/**
EVM instruction set evolution：
HomesteadInstructionSet = FrontierInstructionSet + [DELEGATECALL]

EIP 150:
TangerineWhistleInstructionSet = HomesteadInstructionSet +[BALANCE,EXTCODESIZE,SLOAD,EXTCODECOPY,CALL,CALLCODE,DELEGATECALL]

EIP 158:
SpuriousDragonInstructionSet = TangerineWhistleInstructionSet + EXP

ByzantiumInstructionSet = SpuriousDragonInstructionSet +[STATICCALL,RETURNDATASIZE,RETURNDATACOPY,REVERT]

ConstantinopleInstructionSet = ByzantiumInstructionSet + [SHL,SHR,SAR,EXTCODEHASH,CREATE2]

IstanbulInstructionSet = ConstantinopleInstructionSet + EIP-1344 + EIP-1884 +EIP-2200

EIP 1344:
CHAINID

EIP 1884:
SELFBALANCE
gas cost changed: SLOAD BALANCE EXTCODEHASH

EIP 2200:
SLOAD constantGas
SSTORE dynamicGas

YoloV2InstructionSet = IstanbulInstructionSet + EIP-2315 + EIP-2929

EIP 2315:
BEGINSUB JUMPSUB RETURNSUB

EIP 2929:
dynamicGas:SSTORE,SLOAD,EXTCODECOPY,EXTCODESIZE,EXTCODEHASH,BALANCE,CALL,CALLCODE,STATICCALL,DELEGATECALL,SELFDESTRUCT
constantGas:SLOAD,EXTCODECOPY,EXTCODESIZE,EXTCODEHASH,BALANCE,CALL,CALLCODE,STATICCALL,DELEGATECALL,SELFDESTRUCT

Roctet Protocol VM is compatible all evm instruction set util eth-release 1.9.24
*/

func newInstructionSet() JumpTable {
	//FrontierInstructionSet
	instructionSet := JumpTable{
		STOP: {
			execute:     opStop,
			constantGas: 0,
			minStack:    minStack(0, 0),
			maxStack:    maxStack(0, 0),
			halts:       true,
		},
		ADD: {
			execute:     opAdd,
			constantGas: GasFastestStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		MUL: {
			execute:     opMul,
			constantGas: GasFastStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		SUB: {
			execute:     opSub,
			constantGas: GasFastestStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		DIV: {
			execute:     opDiv,
			constantGas: GasFastStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		SDIV: {
			execute:     opSdiv,
			constantGas: GasFastStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		MOD: {
			execute:     opMod,
			constantGas: GasFastStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		SMOD: {
			execute:     opSmod,
			constantGas: GasFastStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		ADDMOD: {
			execute:     opAddmod,
			constantGas: GasMidStep,
			minStack:    minStack(3, 1),
			maxStack:    maxStack(3, 1),
		},
		MULMOD: {
			execute:     opMulmod,
			constantGas: GasMidStep,
			minStack:    minStack(3, 1),
			maxStack:    maxStack(3, 1),
		},
		EXP: {
			execute:    opExp,
			dynamicGas: gasExpFrontier,
			minStack:   minStack(2, 1),
			maxStack:   maxStack(2, 1),
		},
		SIGNEXTEND: {
			execute:     opSignExtend,
			constantGas: GasFastStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		LT: {
			execute:     opLt,
			constantGas: GasFastestStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		GT: {
			execute:     opGt,
			constantGas: GasFastestStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		SLT: {
			execute:     opSlt,
			constantGas: GasFastestStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		SGT: {
			execute:     opSgt,
			constantGas: GasFastestStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		EQ: {
			execute:     opEq,
			constantGas: GasFastestStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		ISZERO: {
			execute:     opIszero,
			constantGas: GasFastestStep,
			minStack:    minStack(1, 1),
			maxStack:    maxStack(1, 1),
		},
		AND: {
			execute:     opAnd,
			constantGas: GasFastestStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		XOR: {
			execute:     opXor,
			constantGas: GasFastestStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		OR: {
			execute:     opOr,
			constantGas: GasFastestStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		NOT: {
			execute:     opNot,
			constantGas: GasFastestStep,
			minStack:    minStack(1, 1),
			maxStack:    maxStack(1, 1),
		},
		BYTE: {
			execute:     opByte,
			constantGas: GasFastestStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		SHA3: {
			execute:     opSha3,
			constantGas: Sha3Gas,
			dynamicGas:  gasSha3,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
			memorySize:  memorySha3,
		},
		ADDRESS: {
			execute:     opAddress,
			constantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		BALANCE: {
			execute:     opBalance,
			constantGas: BalanceGasFrontier,
			minStack:    minStack(1, 1),
			maxStack:    maxStack(1, 1),
		},
		ORIGIN: {
			execute:     opOrigin,
			constantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		CALLER: {
			execute:     opCaller,
			constantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		CALLVALUE: {
			execute:     opCallValue,
			constantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		CALLDATALOAD: {
			execute:     opCallDataLoad,
			constantGas: GasFastestStep,
			minStack:    minStack(1, 1),
			maxStack:    maxStack(1, 1),
		},
		CALLDATASIZE: {
			execute:     opCallDataSize,
			constantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		CALLDATACOPY: {
			execute:     opCallDataCopy,
			constantGas: GasFastestStep,
			dynamicGas:  gasCallDataCopy,
			minStack:    minStack(3, 0),
			maxStack:    maxStack(3, 0),
			memorySize:  memoryCallDataCopy,
		},
		CODESIZE: {
			execute:     opCodeSize,
			constantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		CODECOPY: {
			execute:     opCodeCopy,
			constantGas: GasFastestStep,
			dynamicGas:  gasCodeCopy,
			minStack:    minStack(3, 0),
			maxStack:    maxStack(3, 0),
			memorySize:  memoryCodeCopy,
		},
		GASPRICE: {
			execute:     opGasprice,
			constantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		EXTCODESIZE: {
			execute:     opExtCodeSize,
			constantGas: ExtcodeSizeGasFrontier,
			minStack:    minStack(1, 1),
			maxStack:    maxStack(1, 1),
		},
		EXTCODECOPY: {
			execute:     opExtCodeCopy,
			constantGas: ExtcodeCopyBaseFrontier,
			dynamicGas:  gasExtCodeCopy,
			minStack:    minStack(4, 0),
			maxStack:    maxStack(4, 0),
			memorySize:  memoryExtCodeCopy,
		},
		BLOCKHASH: {
			execute:     opBlockhash,
			constantGas: GasExtStep,
			minStack:    minStack(1, 1),
			maxStack:    maxStack(1, 1),
		},
		COINBASE: {
			execute:     opCoinbase,
			constantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		TIMESTAMP: {
			execute:     opTimestamp,
			constantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		NUMBER: {
			execute:     opNumber,
			constantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		DIFFICULTY: {
			execute:     opDifficulty,
			constantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		GASLIMIT: {
			execute:     opGasLimit,
			constantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		POP: {
			execute:     opPop,
			constantGas: GasQuickStep,
			minStack:    minStack(1, 0),
			maxStack:    maxStack(1, 0),
		},
		MLOAD: {
			execute:     opMload,
			constantGas: GasFastestStep,
			dynamicGas:  gasMLoad,
			minStack:    minStack(1, 1),
			maxStack:    maxStack(1, 1),
			memorySize:  memoryMLoad,
		},
		MSTORE: {
			execute:     opMstore,
			constantGas: GasFastestStep,
			dynamicGas:  gasMStore,
			minStack:    minStack(2, 0),
			maxStack:    maxStack(2, 0),
			memorySize:  memoryMStore,
		},
		MSTORE8: {
			execute:     opMstore8,
			constantGas: GasFastestStep,
			dynamicGas:  gasMStore8,
			memorySize:  memoryMStore8,
			minStack:    minStack(2, 0),
			maxStack:    maxStack(2, 0),
		},
		SLOAD: {
			execute:     opSload,
			constantGas: SloadGasFrontier,
			minStack:    minStack(1, 1),
			maxStack:    maxStack(1, 1),
		},
		SSTORE: {
			execute:    opSstore,
			dynamicGas: gasSStore,
			minStack:   minStack(2, 0),
			maxStack:   maxStack(2, 0),
			writes:     true,
		},
		JUMP: {
			execute:     opJump,
			constantGas: GasMidStep,
			minStack:    minStack(1, 0),
			maxStack:    maxStack(1, 0),
			jumps:       true,
		},
		JUMPI: {
			execute:     opJumpi,
			constantGas: GasSlowStep,
			minStack:    minStack(2, 0),
			maxStack:    maxStack(2, 0),
			jumps:       true,
		},
		PC: {
			execute:     opPc,
			constantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		MSIZE: {
			execute:     opMsize,
			constantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		GAS: {
			execute:     opGas,
			constantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		JUMPDEST: {
			execute:     opJumpdest,
			constantGas: JumpdestGas,
			minStack:    minStack(0, 0),
			maxStack:    maxStack(0, 0),
		},
		PUSH1: {
			execute:     opPush1,
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH2: {
			execute:     makePush(2, 2),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH3: {
			execute:     makePush(3, 3),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH4: {
			execute:     makePush(4, 4),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH5: {
			execute:     makePush(5, 5),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH6: {
			execute:     makePush(6, 6),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH7: {
			execute:     makePush(7, 7),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH8: {
			execute:     makePush(8, 8),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH9: {
			execute:     makePush(9, 9),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH10: {
			execute:     makePush(10, 10),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH11: {
			execute:     makePush(11, 11),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH12: {
			execute:     makePush(12, 12),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH13: {
			execute:     makePush(13, 13),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH14: {
			execute:     makePush(14, 14),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH15: {
			execute:     makePush(15, 15),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH16: {
			execute:     makePush(16, 16),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH17: {
			execute:     makePush(17, 17),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH18: {
			execute:     makePush(18, 18),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH19: {
			execute:     makePush(19, 19),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH20: {
			execute:     makePush(20, 20),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH21: {
			execute:     makePush(21, 21),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH22: {
			execute:     makePush(22, 22),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH23: {
			execute:     makePush(23, 23),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH24: {
			execute:     makePush(24, 24),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH25: {
			execute:     makePush(25, 25),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH26: {
			execute:     makePush(26, 26),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH27: {
			execute:     makePush(27, 27),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH28: {
			execute:     makePush(28, 28),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH29: {
			execute:     makePush(29, 29),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH30: {
			execute:     makePush(30, 30),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH31: {
			execute:     makePush(31, 31),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH32: {
			execute:     makePush(32, 32),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		DUP1: {
			execute:     makeDup(1),
			constantGas: GasFastestStep,
			minStack:    minDupStack(1),
			maxStack:    maxDupStack(1),
		},
		DUP2: {
			execute:     makeDup(2),
			constantGas: GasFastestStep,
			minStack:    minDupStack(2),
			maxStack:    maxDupStack(2),
		},
		DUP3: {
			execute:     makeDup(3),
			constantGas: GasFastestStep,
			minStack:    minDupStack(3),
			maxStack:    maxDupStack(3),
		},
		DUP4: {
			execute:     makeDup(4),
			constantGas: GasFastestStep,
			minStack:    minDupStack(4),
			maxStack:    maxDupStack(4),
		},
		DUP5: {
			execute:     makeDup(5),
			constantGas: GasFastestStep,
			minStack:    minDupStack(5),
			maxStack:    maxDupStack(5),
		},
		DUP6: {
			execute:     makeDup(6),
			constantGas: GasFastestStep,
			minStack:    minDupStack(6),
			maxStack:    maxDupStack(6),
		},
		DUP7: {
			execute:     makeDup(7),
			constantGas: GasFastestStep,
			minStack:    minDupStack(7),
			maxStack:    maxDupStack(7),
		},
		DUP8: {
			execute:     makeDup(8),
			constantGas: GasFastestStep,
			minStack:    minDupStack(8),
			maxStack:    maxDupStack(8),
		},
		DUP9: {
			execute:     makeDup(9),
			constantGas: GasFastestStep,
			minStack:    minDupStack(9),
			maxStack:    maxDupStack(9),
		},
		DUP10: {
			execute:     makeDup(10),
			constantGas: GasFastestStep,
			minStack:    minDupStack(10),
			maxStack:    maxDupStack(10),
		},
		DUP11: {
			execute:     makeDup(11),
			constantGas: GasFastestStep,
			minStack:    minDupStack(11),
			maxStack:    maxDupStack(11),
		},
		DUP12: {
			execute:     makeDup(12),
			constantGas: GasFastestStep,
			minStack:    minDupStack(12),
			maxStack:    maxDupStack(12),
		},
		DUP13: {
			execute:     makeDup(13),
			constantGas: GasFastestStep,
			minStack:    minDupStack(13),
			maxStack:    maxDupStack(13),
		},
		DUP14: {
			execute:     makeDup(14),
			constantGas: GasFastestStep,
			minStack:    minDupStack(14),
			maxStack:    maxDupStack(14),
		},
		DUP15: {
			execute:     makeDup(15),
			constantGas: GasFastestStep,
			minStack:    minDupStack(15),
			maxStack:    maxDupStack(15),
		},
		DUP16: {
			execute:     makeDup(16),
			constantGas: GasFastestStep,
			minStack:    minDupStack(16),
			maxStack:    maxDupStack(16),
		},
		SWAP1: {
			execute:     makeSwap(1),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(2),
			maxStack:    maxSwapStack(2),
		},
		SWAP2: {
			execute:     makeSwap(2),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(3),
			maxStack:    maxSwapStack(3),
		},
		SWAP3: {
			execute:     makeSwap(3),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(4),
			maxStack:    maxSwapStack(4),
		},
		SWAP4: {
			execute:     makeSwap(4),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(5),
			maxStack:    maxSwapStack(5),
		},
		SWAP5: {
			execute:     makeSwap(5),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(6),
			maxStack:    maxSwapStack(6),
		},
		SWAP6: {
			execute:     makeSwap(6),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(7),
			maxStack:    maxSwapStack(7),
		},
		SWAP7: {
			execute:     makeSwap(7),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(8),
			maxStack:    maxSwapStack(8),
		},
		SWAP8: {
			execute:     makeSwap(8),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(9),
			maxStack:    maxSwapStack(9),
		},
		SWAP9: {
			execute:     makeSwap(9),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(10),
			maxStack:    maxSwapStack(10),
		},
		SWAP10: {
			execute:     makeSwap(10),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(11),
			maxStack:    maxSwapStack(11),
		},
		SWAP11: {
			execute:     makeSwap(11),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(12),
			maxStack:    maxSwapStack(12),
		},
		SWAP12: {
			execute:     makeSwap(12),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(13),
			maxStack:    maxSwapStack(13),
		},
		SWAP13: {
			execute:     makeSwap(13),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(14),
			maxStack:    maxSwapStack(14),
		},
		SWAP14: {
			execute:     makeSwap(14),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(15),
			maxStack:    maxSwapStack(15),
		},
		SWAP15: {
			execute:     makeSwap(15),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(16),
			maxStack:    maxSwapStack(16),
		},
		SWAP16: {
			execute:     makeSwap(16),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(17),
			maxStack:    maxSwapStack(17),
		},
		LOG0: {
			execute:    makeLog(0),
			dynamicGas: makeGasLog(0),
			minStack:   minStack(2, 0),
			maxStack:   maxStack(2, 0),
			memorySize: memoryLog,
			writes:     true,
		},
		LOG1: {
			execute:    makeLog(1),
			dynamicGas: makeGasLog(1),
			minStack:   minStack(3, 0),
			maxStack:   maxStack(3, 0),
			memorySize: memoryLog,
			writes:     true,
		},
		LOG2: {
			execute:    makeLog(2),
			dynamicGas: makeGasLog(2),
			minStack:   minStack(4, 0),
			maxStack:   maxStack(4, 0),
			memorySize: memoryLog,
			writes:     true,
		},
		LOG3: {
			execute:    makeLog(3),
			dynamicGas: makeGasLog(3),
			minStack:   minStack(5, 0),
			maxStack:   maxStack(5, 0),
			memorySize: memoryLog,
			writes:     true,
		},
		LOG4: {
			execute:    makeLog(4),
			dynamicGas: makeGasLog(4),
			minStack:   minStack(6, 0),
			maxStack:   maxStack(6, 0),
			memorySize: memoryLog,
			writes:     true,
		},
		CREATE: {
			execute:     opCreate,
			constantGas: CreateGas,
			dynamicGas:  gasCreate,
			minStack:    minStack(3, 1),
			maxStack:    maxStack(3, 1),
			memorySize:  memoryCreate,
			writes:      true,
			returns:     true,
		},
		CALL: {
			execute:     opCall,
			constantGas: CallGasFrontier,
			dynamicGas:  gasCall,
			minStack:    minStack(7, 1),
			maxStack:    maxStack(7, 1),
			memorySize:  memoryCall,
			returns:     true,
		},
		CALLCODE: {
			execute:     opCallCode,
			constantGas: CallGasFrontier,
			dynamicGas:  gasCallCode,
			minStack:    minStack(7, 1),
			maxStack:    maxStack(7, 1),
			memorySize:  memoryCall,
			returns:     true,
		},
		RETURN: {
			execute:    opReturn,
			dynamicGas: gasReturn,
			minStack:   minStack(2, 0),
			maxStack:   maxStack(2, 0),
			memorySize: memoryReturn,
			halts:      true,
		},
		SELFDESTRUCT: {
			execute:    opSuicide,
			dynamicGas: gasSelfdestruct,
			minStack:   minStack(1, 0),
			maxStack:   maxStack(1, 0),
			halts:      true,
			writes:     true,
		},
	}
	//HomesteadInstructionSet-------------------------------------------------------------------------------------------
	instructionSet[DELEGATECALL] = &operation{
		execute:     opDelegateCall,
		dynamicGas:  gasDelegateCall,
		constantGas: CallGasFrontier,
		minStack:    minStack(6, 1),
		maxStack:    maxStack(6, 1),
		memorySize:  memoryDelegateCall,
		returns:     true,
	}

	//TangerineWhistleInstructionSet------------------------------------------------------------------------------------
	instructionSet[BALANCE].constantGas = BalanceGasEIP150
	instructionSet[EXTCODESIZE].constantGas = ExtcodeSizeGasEIP150
	instructionSet[SLOAD].constantGas = SloadGasEIP150
	instructionSet[EXTCODECOPY].constantGas = ExtcodeCopyBaseEIP150
	instructionSet[CALL].constantGas = CallGasEIP150
	instructionSet[CALLCODE].constantGas = CallGasEIP150
	instructionSet[DELEGATECALL].constantGas = CallGasEIP150

	//SpuriousDragonInstructionSet--------------------------------------------------------------------------------------
	instructionSet[EXP].dynamicGas = gasExpEIP158

	//ByzantiumInstructionSet-------------------------------------------------------------------------------------------
	instructionSet[STATICCALL] = &operation{
		execute:     opStaticCall,
		constantGas: CallGasEIP150,
		dynamicGas:  gasStaticCall,
		minStack:    minStack(6, 1),
		maxStack:    maxStack(6, 1),
		memorySize:  memoryStaticCall,
		returns:     true,
	}
	instructionSet[RETURNDATASIZE] = &operation{
		execute:     opReturnDataSize,
		constantGas: GasQuickStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
	}
	instructionSet[RETURNDATACOPY] = &operation{
		execute:     opReturnDataCopy,
		constantGas: GasFastestStep,
		dynamicGas:  gasReturnDataCopy,
		minStack:    minStack(3, 0),
		maxStack:    maxStack(3, 0),
		memorySize:  memoryReturnDataCopy,
	}
	instructionSet[REVERT] = &operation{
		execute:    opRevert,
		dynamicGas: gasRevert,
		minStack:   minStack(2, 0),
		maxStack:   maxStack(2, 0),
		memorySize: memoryRevert,
		reverts:    true,
		returns:    true,
	}

	//ConstantinopleInstructionSet--------------------------------------------------------------------------------------
	instructionSet[SHL] = &operation{
		execute:     opSHL,
		constantGas: GasFastestStep,
		minStack:    minStack(2, 1),
		maxStack:    maxStack(2, 1),
	}
	instructionSet[SHR] = &operation{
		execute:     opSHR,
		constantGas: GasFastestStep,
		minStack:    minStack(2, 1),
		maxStack:    maxStack(2, 1),
	}
	instructionSet[SAR] = &operation{
		execute:     opSAR,
		constantGas: GasFastestStep,
		minStack:    minStack(2, 1),
		maxStack:    maxStack(2, 1),
	}
	instructionSet[EXTCODEHASH] = &operation{
		execute:     opExtCodeHash,
		constantGas: ExtcodeHashGasConstantinople,
		minStack:    minStack(1, 1),
		maxStack:    maxStack(1, 1),
	}
	instructionSet[CREATE2] = &operation{
		execute:     opCreate2,
		constantGas: Create2Gas,
		dynamicGas:  gasCreate2,
		minStack:    minStack(4, 1),
		maxStack:    maxStack(4, 1),
		memorySize:  memoryCreate2,
		writes:      true,
		returns:     true,
	}

	//IstanbulInstructionSet--------------------------------------------------------------------------------------------
	//EIP 1344
	instructionSet[CHAINID] = &operation{
		execute:     opChainID,
		constantGas: GasQuickStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
	}

	//EIP1884
	instructionSet[SLOAD].constantGas = SloadGasEIP1884
	instructionSet[BALANCE].constantGas = BalanceGasEIP1884
	instructionSet[EXTCODEHASH].constantGas = ExtcodeHashGasEIP1884

	instructionSet[SELFBALANCE] = &operation{
		execute:     opSelfBalance,
		constantGas: GasFastStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
	}

	//EIP 2200
	instructionSet[SLOAD].constantGas = SloadGasEIP2200
	instructionSet[SSTORE].dynamicGas = gasSStoreEIP2200

	//YoloV2InstructionSet----------------------------------------------------------------------------------------------

	////EIP 2315
	//instructionSet[BEGINSUB] = &operation{
	//	execute:     opBeginSub,
	//	constantGas: GasQuickStep,
	//	minStack:    minStack(0, 0),
	//	maxStack:    maxStack(0, 0),
	//}
	//instructionSet[JUMPSUB] = &operation{
	//	execute:     opJumpSub,
	//	constantGas: GasSlowStep,
	//	minStack:    minStack(1, 0),
	//	maxStack:    maxStack(1, 0),
	//	jumps:       true,
	//}
	//instructionSet[RETURNSUB] = &operation{
	//	execute:     opReturnSub,
	//	constantGas: GasFastStep,
	//	minStack:    minStack(0, 0),
	//	maxStack:    maxStack(0, 0),
	//	jumps:       true,
	//}
	//
	////EIP 2929
	//instructionSet[SSTORE].dynamicGas = gasSStoreEIP2929
	//
	//instructionSet[SLOAD].constantGas = 0
	//instructionSet[SLOAD].dynamicGas = gasSLoadEIP2929
	//
	//instructionSet[EXTCODECOPY].constantGas = WarmStorageReadCostEIP2929
	//instructionSet[EXTCODECOPY].dynamicGas = gasExtCodeCopyEIP2929
	//
	//instructionSet[EXTCODESIZE].constantGas = WarmStorageReadCostEIP2929
	//instructionSet[EXTCODESIZE].dynamicGas = gasEip2929AccountCheck
	//
	//instructionSet[EXTCODEHASH].constantGas = WarmStorageReadCostEIP2929
	//instructionSet[EXTCODEHASH].dynamicGas = gasEip2929AccountCheck
	//
	//instructionSet[BALANCE].constantGas = WarmStorageReadCostEIP2929
	//instructionSet[BALANCE].dynamicGas = gasEip2929AccountCheck
	//
	//instructionSet[CALL].constantGas = WarmStorageReadCostEIP2929
	//instructionSet[CALL].dynamicGas = gasCallEIP2929
	//
	//instructionSet[CALLCODE].constantGas = WarmStorageReadCostEIP2929
	//instructionSet[CALLCODE].dynamicGas = gasCallCodeEIP2929
	//
	//instructionSet[STATICCALL].constantGas = WarmStorageReadCostEIP2929
	//instructionSet[STATICCALL].dynamicGas = gasStaticCallEIP2929
	//
	//instructionSet[DELEGATECALL].constantGas = WarmStorageReadCostEIP2929
	//instructionSet[DELEGATECALL].dynamicGas = gasDelegateCallEIP2929
	//
	//// This was previously part of the dynamic cost, but we're using it as a constantGas
	//// factor here
	//instructionSet[SELFDESTRUCT].constantGas = SelfdestructGasEIP150
	//instructionSet[SELFDESTRUCT].dynamicGas = gasSelfdestructEIP2929

	//rangers protocol defined InstructionSet----------------------------------------------------------------------------------------------

	instructionSet[PRINTF] = &operation{
		execute:     opPrintF,
		constantGas: GasExtStep,
		minStack:    minStack(0, 0),
		maxStack:    maxStack(0, 0),
	}

	instructionSet[STAKE] = &operation{
		execute:     opStake,
		constantGas: GasExtStep,
		minStack:    minStack(2, 1),
		maxStack:    maxStack(2, 1),
	}

	instructionSet[UNSTAKE] = &operation{
		execute:     opUnStake,
		constantGas: GasExtStep,
		minStack:    minStack(2, 1),
		maxStack:    maxStack(2, 1),
	}

	instructionSet[GETSTAKE] = &operation{
		execute:     opGetStake,
		constantGas: GasExtStep,
		minStack:    minStack(1, 1),
		maxStack:    maxStack(1, 1),
	}

	instructionSet[UNSTAKEALL] = &operation{
		execute:     opUnStakeAll,
		constantGas: GasExtStep,
		minStack:    minStack(1, 1),
		maxStack:    maxStack(1, 1),
	}

	instructionSet[STAKENUM] = &operation{
		execute:     opStakeNum,
		constantGas: GasExtStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
	}

	instructionSet[AUTH] = &operation{
		execute:     opAuth,
		constantGas: GasExtStep,
		minStack:    minStack(3, 1),
		maxStack:    maxStack(3, 1),
	}

	instructionSet[AUTHCALL] = &operation{
		execute:     opAuthCall,
		constantGas: GasExtStep,
		dynamicGas:  gasCall,
		minStack:    minStack(8, 1),
		maxStack:    maxStack(8, 1),
		memorySize:  memoryAuthCall,
	}

	return instructionSet
}
