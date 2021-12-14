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
	Version           = "1.0.0"
	ProtocolVersion   = 1
	ConsensusVersion  = 1
	ENV_DEV           = "dev"
	ENV_TESTNET_ROBIN = "test"
	ENV_MAINNET       = "mainnet"
)

var (
	mainNetChainConfig = ChainConfig{ChainId: "8888", NetworkId: "8888", Dsn: "readonly:Readonly>123456@tcp(ds.rangersprotocol.com:6666)/rpservice?charset=utf8&parseTime=true&loc=Asia%2FShanghai", PHub: "wss://mainnet.rangersprotocol.com/phub", PubHub: "wss://mainnet.rangersprotocol.com/pubhub"}
	robinChainConfig   = ChainConfig{ChainId: "9527", NetworkId: "9527"}
	devNetChainConfig  = ChainConfig{ChainId: "9500", NetworkId: "9500", Dsn: "readonly:Tuntun123456!@tcp(api.tuntunhz.com:3336)/rpservice_dev?charset=utf8&parseTime=true&loc=Asia%2FShanghai", PHub: "ws://gate.tuntunhz.com:8899", PubHub: "ws://gate.tuntunhz.com:8888"}

	LocalChainConfig ChainConfig
)

type ChainConfig struct {
	ChainId   string
	NetworkId string
	PHub      string
	PubHub    string
	Dsn       string
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
func IsDEV() bool {
	return LocalChainConfig.ChainId == devNetChainConfig.ChainId
}
func IsMainnet() bool {
	return LocalChainConfig.ChainId == mainNetChainConfig.ChainId
}
