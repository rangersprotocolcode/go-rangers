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
	"com.tuntun.rangers/node/src/middleware/log"
	"com.tuntun.rangers/node/src/utility"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
)

var errTraceLimitReached = errors.New("the number of logs reached the specified limit")

// Storage represents a contract's storage.
type Storage map[common.Hash]common.Hash

// Copy duplicates the current storage.
func (s Storage) Copy() Storage {
	cpy := make(Storage)
	for key, value := range s {
		cpy[key] = value
	}
	return cpy
}

// StructLog is emitted to the EVM each cycle and lists information about the current internal state
// prior to the execution of the statement.
type StructLog struct {
	Pc            uint64                      `json:"pc"`
	Op            OpCode                      `json:"op"`
	Gas           uint64                      `json:"gas"`
	GasCost       uint64                      `json:"gasCost"`
	Memory        []byte                      `json:"memory"`
	MemorySize    int                         `json:"memSize"`
	Stack         []*big.Int                  `json:"stack"`
	ReturnStack   []uint32                    `json:"returnStack"`
	ReturnData    []byte                      `json:"returnData"`
	Storage       map[common.Hash]common.Hash `json:"-"`
	Depth         int                         `json:"depth"`
	RefundCounter uint64                      `json:"refund"`
	Err           error                       `json:"-"`
}

// OpName formats the operand name in a human-readable format.
func (s *StructLog) OpName() string {
	return s.Op.String()
}

// ErrorString formats the log's error as a string.
func (s *StructLog) ErrorString() string {
	if s.Err != nil {
		return s.Err.Error()
	}
	return ""
}

// MarshalJSON marshals as JSON.
func (s StructLog) MarshalJSON() ([]byte, error) {
	type StructLog struct {
		Pc      uint64 `json:"pc"`
		Op      OpCode `json:"op"`
		OpName  string `json:"opName"`
		Depth   int    `json:"depth"`
		Gas     uint64 `json:"gas"`
		GasCost uint64 `json:"gasCost"`
		Memory  string `json:"memory"`
		//MemorySize    int                        `json:"memSize"`
		Stack   []*utility.HexOrDecimal256 `json:"stack"`
		Storage string                     `json:"storage"`
		//ReturnStack   []utility.HexOrDecimal64   `json:"returnStack"`
		//ReturnData    utility.Bytes              `json:"returnData"`
		//RefundCounter uint64                     `json:"refund"`
		//Err           error                      `json:"-"`
		//ErrorString   string                     `json:"error"`
	}
	var enc StructLog
	enc.Pc = s.Pc
	enc.Op = s.Op
	enc.OpName = s.OpName()
	enc.Gas = s.Gas
	enc.GasCost = s.GasCost
	enc.Depth = s.Depth

	memoryBytes, _ := json.Marshal(printBytes(s.Memory))
	enc.Memory = string(memoryBytes)
	//enc.MemorySize = s.MemorySize
	if s.Stack != nil {
		enc.Stack = make([]*utility.HexOrDecimal256, len(s.Stack))
		for k, v := range s.Stack {
			enc.Stack[k] = (*utility.HexOrDecimal256)(v)
		}
	}

	if s.Storage != nil {
		storageBytes, _ := json.Marshal(s.Storage)
		enc.Storage = string(storageBytes)
	}

	//if s.ReturnStack != nil {
	//	enc.ReturnStack = make([]utility.HexOrDecimal64, len(s.ReturnStack))
	//	for k, v := range s.ReturnStack {
	//		enc.ReturnStack[k] = utility.HexOrDecimal64(v)
	//	}
	//}
	//enc.ReturnData = s.ReturnData
	//enc.RefundCounter = s.RefundCounter
	//enc.Err = s.Err
	//enc.ErrorString = s.ErrorString()
	return json.Marshal(&enc)
}

func printBytes(bytes []byte) []string {
	result := make([]string, 0)
	for i := 0; i < len(bytes); i += 32 {
		s := common.ToHex(bytes[i : i+32])
		result = append(result, s)
	}
	return result
}

// Tracer is used to collect execution traces from an EVM transaction
// execution. CaptureState is called for each step of the VM with the
// current VM state.
// Note that reference types are actual VM data structures; make copies
// if you need to retain them beyond the current call.
type Tracer interface {
	CaptureStart(from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) error
	CaptureState(env *EVM, pc uint64, op OpCode, gas, cost uint64, memory *Memory, stack *Stack, rStack *ReturnStack, rData []byte, contract *Contract, depth int, err error) error
	CaptureFault(env *EVM, pc uint64, op OpCode, gas, cost uint64, memory *Memory, stack *Stack, rStack *ReturnStack, contract *Contract, depth int, err error) error
	CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) error
}

// StructLogger is an EVM state logger and implements Tracer.
//
// StructLogger can capture state based on the given Log configuration and also keeps
// a track record of modified storage which is used in reporting snapshots of the
// contract their storage.
type StructLogger struct {
	cfg LogConfig

	storage map[common.Address]Storage
	logger  log.Logger
	output  []byte
	err     error
}

// NewStructLogger returns a new logger
func NewStructLogger(cfg *LogConfig, logger log.Logger) *StructLogger {
	structLogger := &StructLogger{
		storage: make(map[common.Address]Storage),
		logger:  logger,
	}
	if cfg != nil {
		structLogger.cfg = *cfg
	}
	return structLogger
}

// CaptureStart implements the Tracer interface to initialize the tracing operation.
func (l *StructLogger) CaptureStart(from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) error {
	if !create {
		l.logger.Debugf("Contract call:\nFrom: `%v`\nTo: `%v`\nData: `0x%x`\nGas: `%d`\nValue `%v` wei\n",
			from.String(), to.String(),
			input, gas, value)
	} else {
		l.logger.Debugf("Contract create:\nFrom: `%v`\nCreate at: `%v`\nData: `0x%x`\nGas: `%d`\nValue `%v` wei\n",
			from.String(), to.String(),
			input, gas, value)
	}
	return nil
}

// CaptureState logs a new structured log message and pushes it out to the environment
//
// CaptureState also tracks SLOAD/SSTORE ops to track storage change.
func (l *StructLogger) CaptureState(env *EVM, pc uint64, op OpCode, gas, cost uint64, memory *Memory, stack *Stack, rStack *ReturnStack, rData []byte, contract *Contract, depth int, err error) error {
	// Copy a snapshot of the current memory state to a new buffer
	var mem []byte
	if !l.cfg.DisableMemory {
		mem = make([]byte, len(memory.Data()))
		copy(mem, memory.Data())
	}
	// Copy a snapshot of the current stack state to a new buffer
	var stck []*big.Int
	if !l.cfg.DisableStack {
		stck = make([]*big.Int, len(stack.Data()))
		for i, item := range stack.Data() {
			stck[i] = new(big.Int).Set(item.ToBig())
		}
	}
	var rstack []uint32
	if !l.cfg.DisableStack && rStack != nil {
		rstck := make([]uint32, len(rStack.data))
		copy(rstck, rStack.data)
	}
	// Copy a snapshot of the current storage to a new container
	var storage Storage
	if !l.cfg.DisableStorage {
		// initialise new changed values storage container for this contract
		// if not present.
		if l.storage[contract.Address()] == nil {
			l.storage[contract.Address()] = make(Storage)
		}
		// capture SLOAD opcodes and record the read entry in the local storage
		if op == SLOAD && stack.len() >= 1 {
			var (
				address = common.Hash(stack.data[stack.len()-1].Bytes32())
				value   = env.StateDB.GetState(contract.Address(), address)
			)
			l.storage[contract.Address()][address] = value
		}
		// capture SSTORE opcodes and record the written entry in the local storage.
		if op == SSTORE && stack.len() >= 2 {
			var (
				value   = common.Hash(stack.data[stack.len()-2].Bytes32())
				address = common.Hash(stack.data[stack.len()-1].Bytes32())
			)
			l.storage[contract.Address()][address] = value
		}
		storage = l.storage[contract.Address()].Copy()
	}
	var rdata []byte
	if !l.cfg.DisableReturnData {
		rdata = make([]byte, len(rData))
		copy(rdata, rData)
	}
	// create a new snapshot of the EVM.
	log := StructLog{pc, op, gas, cost, mem, memory.Len(), stck, rstack, rdata, storage, depth, env.StateDB.GetRefund(), err}
	logStr, _ := log.MarshalJSON()
	l.logger.Debugf(string(logStr))
	return nil
}

// CaptureFault implements the Tracer interface to trace an execution fault
// while running an opcode.
func (l *StructLogger) CaptureFault(env *EVM, pc uint64, op OpCode, gas, cost uint64, memory *Memory, stack *Stack, rStack *ReturnStack, contract *Contract, depth int, err error) error {
	return nil
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (l *StructLogger) CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) error {
	l.output = output
	l.err = err
	l.logger.Debugf("\nOutput: `0x%x`\nConsumed gas: `%d`\nError: `%v`\n",
		output, gasUsed, err)
	return nil
}

type mdLogger struct {
	logger log.Logger
	cfg    *LogConfig
}

// NewMarkdownLogger creates a logger which outputs information in a format adapted
// for human readability, and is also a valid markdown table
func NewMarkdownLogger(cfg *LogConfig, logger log.Logger) *mdLogger {
	l := &mdLogger{logger, cfg}
	if l.cfg == nil {
		l.cfg = &LogConfig{}
	}
	return l
}

func (t *mdLogger) CaptureStart(from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) error {
	if !create {
		t.logger.Debugf("Contract call:\nFrom: `%v`\nTo: `%v`\nData: `0x%x`\nGas: `%d`\nValue `%v` wei\n",
			from.String(), to.String(),
			input, gas, value)
	} else {
		t.logger.Debugf("Contract create:\nFrom: `%v`\nCreate at: `%v`\nData: `0x%x`\nGas: `%d`\nValue `%v` wei\n",
			from.String(), to.String(),
			input, gas, value)
	}

	t.logger.Debugf(`|  Pc   |      Op     | Cost |   Stack   |   RStack  |  Refund |`)
	t.logger.Debugf(`|-------|-------------|------|-----------|-----------|---------|`)
	return nil
}

func (t *mdLogger) CaptureState(env *EVM, pc uint64, op OpCode, gas, cost uint64, memory *Memory, stack *Stack, rStack *ReturnStack, rData []byte, contract *Contract, depth int, err error) error {
	logContent := fmt.Sprintf("| %4d  | %10v  |  %3d |", pc, op, cost)

	if !t.cfg.DisableStack {
		// format stack
		var a []string
		for _, elem := range stack.data {
			a = append(a, fmt.Sprintf("%v", elem.String()))
		}
		b := fmt.Sprintf("[%v]", strings.Join(a, ","))
		logContent += fmt.Sprintf("%10v |", b)
		// format return stack
		a = a[:0]
		for _, elem := range rStack.data {
			a = append(a, fmt.Sprintf("%2d", elem))
		}
		b = fmt.Sprintf("[%v]", strings.Join(a, ","))
		logContent += fmt.Sprintf("%10v |", b)
	}
	logContent += fmt.Sprintf("%10v |", env.StateDB.GetRefund())
	logContent += fmt.Sprintf("")
	if err != nil {
		t.logger.Debugf("Error: %v\n", err)
	}
	t.logger.Debugf(logContent)
	return nil
}

func (t *mdLogger) CaptureFault(env *EVM, pc uint64, op OpCode, gas, cost uint64, memory *Memory, stack *Stack, rStack *ReturnStack, contract *Contract, depth int, err error) error {

	t.logger.Debugf("\nError: at pc=%d, op=%v: %v\n", pc, op, err)

	return nil
}

func (t *mdLogger) CaptureEnd(output []byte, gasUsed uint64, tm time.Duration, err error) error {
	t.logger.Debugf("End:\nOutput: `0x%x`\nConsumed gas: `%d`\nError: `%v`\n",
		output, gasUsed, err)
	return nil
}
