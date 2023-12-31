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
	"com.tuntun.rangers/node/src/utility"
	"fmt"
)

const (
	FTPrefix    = "f:"
	FTSetPrefix = "fs:"
)

const (
	Official  = "official"
	BNTPrefix = Official + "-"
)

const (
	ERC20BindingPrefix = "erc20-"
)

func GenerateBNTName(bntName string) string {
	return fmt.Sprintf("%s%s", BNTPrefix, bntName)
}

func FormatBNTName(bntName string) string {
	return bntName[len(BNTPrefix):]
}

func GenerateFTSetAddress(ftSetId string) Address {
	addr := fmt.Sprintf("%s%s", FTSetPrefix, ftSetId)
	return BytesToAddress(Sha256(utility.StrToBytes(addr)))
}

func GenerateFTKey(name string) string {
	return fmt.Sprintf("%s%s", FTPrefix, name)
}

func GenerateERC20Binding(name string) Address {
	addr := fmt.Sprintf("%s%s", ERC20BindingPrefix, name)
	return BytesToAddress(Sha256(utility.StrToBytes(addr)))
}
