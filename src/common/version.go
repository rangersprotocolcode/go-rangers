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

package common

import "math/big"

const (
	Version           = "0.9"
	ProtocolVersion   = 1
	ConsensusVersion  = 1
	ENV_DEV           = "dev"
	ENV_TESTNET_ROBIN = "test"
)

//used　to distinguish different network env
var NetworkId = "9500"

//used　to distinguish different fork
var ChainId = "9500"

func GetChainId() *big.Int {
	chainId, _ := big.NewInt(0).SetString(ChainId, 10)
	return chainId
}

func InitChainId(env string) {
	if env == ENV_DEV {
		ChainId = "10000"
		NetworkId = "10000"
	} else if env == ENV_TESTNET_ROBIN {
		ChainId = "9527"
		NetworkId = "9527"
	}
}
