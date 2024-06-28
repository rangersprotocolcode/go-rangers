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
	"github.com/holiman/uint256"
)

// rangers protocol defined InstructionSet newest----------------------------------------------------------------------------------------------
func doProposal014(instructionSet *JumpTable) {
	instructionSet[PRINTF] = &operation{
		execute:     opPrintF,
		constantGas: PrintGas,
		minStack:    minStack(0, 0),
		maxStack:    maxStack(0, 0),
	}

	instructionSet[STAKE] = &operation{
		execute:     opStake,
		constantGas: StakeGas,
		minStack:    minStack(2, 1),
		maxStack:    maxStack(2, 1),
	}

	instructionSet[UNSTAKE] = &operation{
		execute:     opUnStake,
		constantGas: UnStakeGas,
		minStack:    minStack(2, 1),
		maxStack:    maxStack(2, 1),
	}

	instructionSet[GETSTAKE] = &operation{
		execute:     opGetStake,
		constantGas: GetStake,
		minStack:    minStack(1, 1),
		maxStack:    maxStack(1, 1),
	}

	instructionSet[UNSTAKEALL] = &operation{
		execute:     opUnStakeAll,
		constantGas: UnStakeAllGas,
		minStack:    minStack(1, 1),
		maxStack:    maxStack(1, 1),
	}

	instructionSet[STAKENUM] = &operation{
		execute:     opStakeNum,
		constantGas: StakeNumGas,
		minStack:    minStack(1, 1),
		maxStack:    maxStack(1, 1),
	}

	instructionSet[AUTH] = &operation{
		execute:     opAuth,
		constantGas: AuthGas,
		dynamicGas:  gasAuth,
		minStack:    minStack(3, 1),
		maxStack:    maxStack(3, 1),
	}

	instructionSet[AUTHCALL] = &operation{
		execute:     opAuthCall,
		constantGas: WarmStorageReadCostEIP2929,
		dynamicGas:  gasAuthCall,
		minStack:    minStack(9, 1),
		maxStack:    maxStack(9, 1),
		memorySize:  memoryAuthCall,
	}
}

func doProposal022(jt *JumpTable) {
	// New opcode
	jt[BASEFEE] = &operation{
		execute:     opBaseFee,
		constantGas: GasQuickStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
	}

	jt[BLOBHASH] = &operation{
		execute:     opBlobHash,
		constantGas: GasFastestStep,
		minStack:    minStack(1, 1),
		maxStack:    maxStack(1, 1),
	}

	jt[BLOBBASEFEE] = &operation{
		execute:     opBlobBaseFee,
		constantGas: GasQuickStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
	}

	jt[TLOAD] = &operation{
		execute:     opTload,
		constantGas: WarmStorageReadCostEIP2929,
		minStack:    minStack(1, 1),
		maxStack:    maxStack(1, 1),
	}

	jt[TSTORE] = &operation{
		execute:     opTstore,
		constantGas: WarmStorageReadCostEIP2929,
		minStack:    minStack(2, 0),
		maxStack:    maxStack(2, 0),
	}

	jt[MCOPY] = &operation{
		execute:     opMcopy,
		constantGas: GasFastestStep,
		dynamicGas:  gasMcopy,
		minStack:    minStack(3, 0),
		maxStack:    maxStack(3, 0),
		memorySize:  memoryMcopy,
	}

	jt[PUSH0] = &operation{
		execute:     opPush0,
		constantGas: GasQuickStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
	}
}

// opBaseFee implements BASEFEE opcode
func opBaseFee(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	baseFee := uint256.NewInt()
	callContext.stack.push(baseFee)
	return nil, nil
}

func opBlobHash(pc *uint64, interpreter *EVMInterpreter, scope *callCtx) ([]byte, error) {
	index := scope.stack.peek()
	index.SetBytes32([]byte{})
	return nil, nil
}

// opBlobBaseFee implements BLOBBASEFEE opcode
func opBlobBaseFee(pc *uint64, interpreter *EVMInterpreter, scope *callCtx) ([]byte, error) {
	blobBaseFee := uint256.NewInt()
	scope.stack.push(blobBaseFee)
	return nil, nil
}

// opTload implements TLOAD opcode
func opTload(pc *uint64, interpreter *EVMInterpreter, scope *callCtx) ([]byte, error) {
	loc := scope.stack.peek()
	hash := common.Hash(loc.Bytes32())
	val := interpreter.evm.StateDB.GetTransientState(scope.contract.Address(), hash)
	loc.SetBytes(val.Bytes())
	return nil, nil
}

// opTstore implements TSTORE opcode
func opTstore(pc *uint64, interpreter *EVMInterpreter, scope *callCtx) ([]byte, error) {
	if interpreter.readOnly {
		return nil, ErrWriteProtection
	}
	loc := scope.stack.pop()
	val := scope.stack.pop()
	interpreter.evm.StateDB.SetTransientState(scope.contract.Address(), loc.Bytes32(), val.Bytes32())
	return nil, nil
}

// opMcopy implements the MCOPY opcode (https://eips.ethereum.org/EIPS/eip-5656)
func opMcopy(pc *uint64, interpreter *EVMInterpreter, scope *callCtx) ([]byte, error) {
	var (
		dst    = scope.stack.pop()
		src    = scope.stack.pop()
		length = scope.stack.pop()
	)
	// These values are checked for overflow during memory expansion calculation
	// (the memorySize function on the opcode).
	scope.memory.Copy(dst.Uint64(), src.Uint64(), length.Uint64())
	return nil, nil
}

// opPush0 implements the PUSH0 opcode
func opPush0(pc *uint64, interpreter *EVMInterpreter, scope *callCtx) ([]byte, error) {
	scope.stack.push(new(uint256.Int))
	return nil, nil
}

func doProposal026(jt *JumpTable) {
	for _, operator := range jt {
		if operator != nil {
			operator.constantGas = operator.constantGas * gasMagnification
		}
	}
}
