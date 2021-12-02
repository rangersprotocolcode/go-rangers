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

package account

import (
	"bytes"
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/utility"
	"golang.org/x/crypto/sha3"
	"hash"
	"strings"
)

var (
	contractKey        = utility.StrToBytes("c")
	positionKey        = utility.StrToBytes("p")
	decimalKey         = utility.StrToBytes("d")
	rpgContractAddress = common.Address{}
)

func (self *AccountDB) AddERC20Binding(name string, contract common.Address, position, decimal uint64) bool {
	address := common.GenerateERC20Binding(name)
	if self.Exist(address) {
		return false
	}

	self.SetData(address, contractKey, contract.Bytes())
	self.SetData(address, positionKey, utility.UInt64ToByte(position))
	self.SetData(address, decimalKey, utility.UInt64ToByte(decimal))
	return true
}

func (self *AccountDB) loadContractCache() {
	address := common.GenerateERC20Binding(common.BLANCE_NAME)
	value1 := common.BytesToAddress(self.GetData(address, contractKey))
	rpgContractAddress = value1
}

func (self *AccountDB) GetERC20Binding(name string) (found bool, contract common.Address, position uint64, decimal uint64) {
	if 0 == strings.Compare(name, common.BLANCE_NAME) {
		if 0 == bytes.Compare(rpgContractAddress.Bytes(), common.Address{}.Bytes()) {
			self.loadContractCache()
		}
		return true, rpgContractAddress, 3, 18
	}

	found = false
	address := common.GenerateERC20Binding(name)
	if !self.Exist(address) {
		return found, common.Address{}, 0, 0
	}

	found = true
	contract = common.BytesToAddress(self.GetData(address, contractKey))
	position = utility.ByteToUInt64(self.GetData(address, positionKey))
	decimal = utility.ByteToUInt64(self.GetData(address, decimalKey))
	return
}

func (self *AccountDB) GetERC20Key(address common.Address, position uint64) []byte {
	data := [64]byte{}
	copy(data[12:], address.Bytes())
	positionBytes := utility.UInt64ToByte(position)
	copy(data[64-len(positionBytes):], positionBytes)

	hasher := sha3.NewLegacyKeccak256().(keccakState)
	hasher.Write(data[:])
	result := [32]byte{}
	hasher.Read(result[:])
	return result[:]
}

type keccakState interface {
	hash.Hash
	Read([]byte) (int, error)
}
