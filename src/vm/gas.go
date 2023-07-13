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
	"github.com/holiman/uint256"
)

// Gas costs
const (
	GasQuickStep   uint64 = 2
	GasFastestStep uint64 = 3
	GasFastStep    uint64 = 5
	GasMidStep     uint64 = 8
	GasSlowStep    uint64 = 10
	GasExtStep     uint64 = 20
)

// callGas returns the actual gas cost of the call.
//
// The cost of gas was changed during the homestead price change HF.
// As part of EIP 150 (TangerineWhistle), the returned gas is gas - base * 63 / 64.
func callGas(isEip150 bool, availableGas, base uint64, callCost *uint256.Int) (uint64, error) {
	if isEip150 {
		availableGas = availableGas - base
		gas := availableGas - availableGas/64
		// If the bit length exceeds 64 bit we know that the newly calculated "gas" for EIP150
		// is smaller than the requested amount. Therefore we return the new gas instead
		// of returning an error.
		if !callCost.IsUint64() || gas < callCost.Uint64() {
			return gas, nil
		}
	}
	if !callCost.IsUint64() {
		return 0, ErrGasUintOverflow
	}

	return callCost.Uint64(), nil
}

// authCallGas returns the actual gas cost of the auth call.
func authCallGas(availableGas, base uint64, callCost *uint256.Int) (uint64, error) {
	logger.Debugf("[authCallGas]availableGas:%d,base:%d,callcost:%d", availableGas, base, callCost)
	availableGas = availableGas - base
	gas := availableGas - availableGas/64

	if callCost == nil || !callCost.IsUint64() || callCost.IsZero() {
		return gas, nil
	}
	if gas < callCost.Uint64() {
		//implement different from 3074,just give max remain gas,may be sub_call use less than remaining_gas
		//be the same with call here
		logger.Debugf("[authCallGas]return:%d", callCost.Uint64())
		return gas, nil
	}
	logger.Debugf("[authCallGas]return:%d", callCost.Uint64())
	return callCost.Uint64(), nil
}
