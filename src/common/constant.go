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
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package common

import (
	"errors"
	"math/big"
	"reflect"
)

const PREFIX = "0x"

const (
	PubKeyLength = 65 //1 bytes curve, 64 bytes x,y。
	SecKeyLength = 97 //65 bytes pub, 32 bytes D。
	SignLength   = 65 //32 bytes r & 32 bytes s & 1 byte recid.

	AddressLength = 20 // golang.SHA3, 160
	HashLength    = 32 // golang.SHA3, 256
	GroupIdLength = 32
)

const (
	MinerTypeValidator = 0
	MinerTypeProposer  = 1

	MinerStatusNormal = 0
	MinerStatusAbort  = 1
)

var (
	hashT    = reflect.TypeOf(Hash{})
	addressT = reflect.TypeOf(Address{})
)

var (
	ValidatorDBAddress = BigToAddress(big.NewInt(1))
	ProposerDBAddress  = BigToAddress(big.NewInt(2))
	DifficultyAddress  = BigToAddress(big.NewInt(3))
)

var (
	Big0   = big.NewInt(0)
	Big1   = big.NewInt(1)
	Big2   = big.NewInt(2)
	Big3   = big.NewInt(3)
	Big32  = big.NewInt(32)
	Big256 = big.NewInt(256)
	Big257 = big.NewInt(257)

	ErrSelectGroupNil     = errors.New("selectGroupId is nil")
	ErrSelectGroupInequal = errors.New("selectGroupId not equal")
	ErrCreateBlockNil     = errors.New("createBlock is nil")
	ErrGroupAlreadyExist  = errors.New("group already exist")
)
