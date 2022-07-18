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
	"com.tuntun.rocket/node/src/middleware/log"
	"math/big"
	"sync/atomic"
)

const (
	Version           = "1.0.8"
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
		Proposal001Block: 894116,
		Proposal002Block: 3353000,
		Proposal003Block: 3830000,
		Proposal004Block: 5310000,
		Proposal005Block: 10293600,
		Proposal006Block: 16733000,
		Proposal007Block: 16082000,
		Proposal008Block: 16082000,
		Proposal009Block: 16733000,
	}

	robinChainConfig = ChainConfig{
		ChainId:          "9527",
		NetworkId:        "9527",
		OriginalChainId:  "9527",
		Proposal001Block: 0,
		Proposal002Block: 2802000,
		Proposal003Block: 3380000,
		Proposal004Block: 5310000,
		Proposal005Block: 10003000,
		Proposal006Block: 12582000,
		Proposal007Block: 14261000,
		Proposal008Block: 16058000,
		Proposal009Block: 16740000,

		mainNodeContract: HexToAddress("0x3a8467bEcb0B702c5c6343c8A3Ccb11acE0e8816"),
	}

	devNetChainConfig = ChainConfig{
		ChainId:          "9500",
		NetworkId:        "9500",
		Dsn:              "readonly:Tuntun123456!@tcp(api.tuntunhz.com:3336)/rpservice_dev?charset=utf8&parseTime=true&loc=Asia%2FShanghai",
		PHub:             "ws://gate.tuntunhz.com:8899",
		PubHub:           "ws://gate.tuntunhz.com:8888",
		OriginalChainId:  "9500",
		mainNodeContract: HexToAddress("0x27B01A9E699F177634f480Cc2150425009Edc5fD"),

		Proposal001Block: 300,
		Proposal002Block: 338000,
		Proposal003Block: 920000,
		Proposal004Block: 5310000,
		Proposal005Block: 1000,
		Proposal006Block: 0,
		Proposal007Block: 0,
		Proposal008Block: 0,
		Proposal009Block: 0,
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
	Proposal002Block uint64
	Proposal003Block uint64
	Proposal004Block uint64
	Proposal005Block uint64
	Proposal006Block uint64
	Proposal007Block uint64
	Proposal008Block uint64
	Proposal009Block uint64

	mainNodeContract Address
}

func InitChainConfig(env string) {
	if env == ENV_DEV {
		LocalChainConfig = devNetChainConfig
	} else if env == ENV_MAINNET {
		LocalChainConfig = mainNetChainConfig
	} else {
		LocalChainConfig = robinChainConfig
	}

	localChainInfo = chainInfo{
		currentBlockHeight: atomic.Value{},
	}
	localChainInfo.currentBlockHeight.Store(uint64(0))
	blockHeightLogger = log.GetLoggerByIndex(log.BlockHeightConfig, "")
}

func GetChainId(height uint64) *big.Int {
	chainIdStr := ChainId(height)
	chainId, _ := big.NewInt(0).SetString(chainIdStr, 10)
	return chainId
}

func ChainId(height uint64) string {
	if IsProposal001(height) {
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

func IsProposal002() bool {
	return isForked(LocalChainConfig.Proposal002Block, GetBlockHeight())
}

func IsProposal003() bool {
	return isForked(LocalChainConfig.Proposal003Block, GetBlockHeight())
}

func IsProposal004() bool {
	return isForked(LocalChainConfig.Proposal004Block, GetBlockHeight())
}

func IsProposal005() bool {
	return isForked(LocalChainConfig.Proposal005Block, GetBlockHeight())
}

// user nonce
func IsProposal006() bool {
	return isForked(LocalChainConfig.Proposal006Block, GetBlockHeight())
}

func IsProposal007() bool {
	return isForked(LocalChainConfig.Proposal007Block, GetBlockHeight())
}

func IsProposal008() bool {
	return isForked(LocalChainConfig.Proposal008Block, GetBlockHeight())
}

func IsProposal009() bool {
	return isForked(LocalChainConfig.Proposal009Block, GetBlockHeight())
}

func isForked(base uint64, height uint64) bool {
	return height >= base
}

func MainNodeContract() Address {
	return LocalChainConfig.mainNodeContract
}
