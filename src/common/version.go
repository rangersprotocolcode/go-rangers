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
	ENV_MAINNET       = "mainnet"
)

var (
	mainNetChainConfig = ChainConfig{ChainId: "8888", NetworkId: "8888"}
	robinChainConfig   = ChainConfig{ChainId: "9527", NetworkId: "9527"}
	devNetChainConfig  = ChainConfig{ChainId: "9500", NetworkId: "9500"}

	LocalChainConfig ChainConfig
)

type ChainConfig struct {
	ChainId   string
	NetworkId string
}

func InitChainConfig(env string) {
	if env == ENV_DEV {
		LocalChainConfig = devNetChainConfig
	} else if env == ENV_MAINNET {
		LocalChainConfig = mainNetChainConfig
	} else {
		LocalChainConfig = robinChainConfig
	}
}

func GetChainId() *big.Int {
	chainId, _ := big.NewInt(0).SetString(LocalChainConfig.ChainId, 10)
	return chainId
}

func ChainId() string {
	return LocalChainConfig.ChainId
}

func NetworkId() string {
	return LocalChainConfig.NetworkId
}
func IsRobin() bool {
	return LocalChainConfig.ChainId == robinChainConfig.ChainId
}
