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

import (
	"math/big"
)

const (
	Version           = "1.0.0"
	ProtocolVersion   = 1
	ConsensusVersion  = 1
	ENV_DEV           = "dev"
	ENV_TESTNET_ROBIN = "test"
	ENV_MAINNET       = "mainnet"
)

var (
	mainNetChainConfig = ChainConfig{
		ChainId:          "2025",
		NetworkId:        "2025",
		Dsn:              "readonly:Readonly>123456@tcp(ds.rangersprotocol.com:6666)/rpservice?charset=utf8&parseTime=true&loc=Asia%2FShanghai",
		PHub:             "wss://mainnet.rangersprotocol.com/phub",
		PubHub:           "wss://mainnet.rangersprotocol.com/pubhub",
		OriginalChainId:  "8888",
		Proposal001Block: 1000000,
	}

	robinChainConfig = ChainConfig{
		ChainId:          "9527",
		NetworkId:        "9527",
		OriginalChainId:  "9527",
		Proposal001Block: 0,
	}

	devNetChainConfig = ChainConfig{
		ChainId:          "9500",
		NetworkId:        "9500",
		Dsn:              "readonly:Tuntun123456!@tcp(api.tuntunhz.com:3336)/rpservice_dev?charset=utf8&parseTime=true&loc=Asia%2FShanghai",
		PHub:             "ws://gate.tuntunhz.com:8899",
		PubHub:           "ws://gate.tuntunhz.com:8888",
		OriginalChainId:  "9500",
		Proposal001Block: 0,
	}

	LocalChainConfig ChainConfig
)

type ChainConfig struct {
	ChainId   string
	NetworkId string

	PHub   string
	PubHub string
	Dsn    string

	OriginalChainId  string
	Proposal001Block uint64
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

func GetChainId(forkedProposal001 bool) *big.Int {
	chainIdStr := ChainId(forkedProposal001)
	chainId, _ := big.NewInt(0).SetString(chainIdStr, 10)
	return chainId
}

func ChainId(forkedProposal001 bool) string {
	if forkedProposal001 {
		return LocalChainConfig.ChainId
	} else {
		return LocalChainConfig.OriginalChainId
	}
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

func IsProposal001(height uint64) bool {
	return isForked(LocalChainConfig.Proposal001Block, height)
}

func isForked(base uint64, height uint64) bool {
	return height >= base
}
