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
	"com.tuntun.rangers/node/src/middleware/log"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"os"
	"strconv"
	"sync/atomic"
)

const (
	Version           = "1.0.23"
	ProtocolVersion   = 1
	ConsensusVersion  = 1
	ENV_DEV           = "dev"
	ENV_TESTNET_ROBIN = "robin"
	ENV_MAINNET       = "mainnet"
)

var (
	mainNetChainConfig = ChainConfig{
		ChainId:          "2025",
		NetworkId:        "2025",
		PHub:             "wss://mainnet.rangersprotocol.com/phub",
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
		Proposal010Block: math.MaxUint64, //mainnet never use proporal010
		Proposal011Block: 11750354,
		Proposal012Block: 22815000,
		Proposal013Block: 28998000,
		Proposal014Block: 48081000,
		Proposal015Block: 53015000,
		Proposal016Block: 54038500,
		Proposal017Block: 54038500,
		Proposal018Block: 55959500,
		Proposal019Block: math.MaxUint64, //mainnet never use proporal010
		Proposal021Block: 0,              //todo
		mainNodeContract: HexToAddress("0x74448149F549CD819b7173b6D67DbBEAFd2909a7"),
		MysqlDSN:         "rpservice:!890rpService@#$@tcp(172.16.0.60:6666)/service?charset=utf8&parseTime=true&loc=Asia%2FShanghai",
		JsonRPCUrl:       "https://mainnet.rangersprotocol.com/api/jsonrpc",
	}

	robinChainConfig = ChainConfig{
		ChainId:          "9527",
		NetworkId:        "9527",
		PHub:             "wss://robin.rangersprotocol.com/phub",
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
		Proposal010Block: 19632000,
		Proposal011Block: math.MaxUint64, //robin never use Proposal011
		Proposal012Block: 23120000,
		Proposal013Block: 29063000,
		Proposal014Block: 0,
		Proposal015Block: 61205000,
		Proposal016Block: 62320000,
		Proposal017Block: 62997000,
		Proposal018Block: 65795000,
		Proposal019Block: 66114000,
		Proposal021Block: 0, //todo

		mainNodeContract: HexToAddress("0x3a8467bEcb0B702c5c6343c8A3Ccb11acE0e8816"),

		MysqlDSN:   "rpservice_v2:oJ2*bA0:hB3%@tcp(192.168.0.172:5555)/rpservice_v2?charset=utf8&parseTime=true&loc=Asia%2FShanghai",
		JsonRPCUrl: "https://robin.rangersprotocol.com/api/jsonrpc",
	}

	devNetChainConfig = ChainConfig{
		ChainId:   "9500",
		NetworkId: "9500",
		PHub:      "ws://gate.tuntunhz.com:8899",

		OriginalChainId:  "9500",
		mainNodeContract: HexToAddress("0x27B01A9E699F177634f480Cc2150425009Edc5fD"),

		Proposal001Block: 0,
		Proposal002Block: 0,
		Proposal003Block: 0,
		Proposal004Block: 0,
		Proposal005Block: 0,
		Proposal006Block: 0,
		Proposal007Block: 0,
		Proposal008Block: 0,
		Proposal009Block: 0,
		Proposal010Block: 0,
		Proposal011Block: 0,
		Proposal012Block: 0,
		Proposal013Block: 0,
		Proposal014Block: 0,
		Proposal015Block: 0,
		Proposal016Block: 0,
		Proposal017Block: 0,
		Proposal018Block: 0,
		Proposal019Block: 0,
		Proposal021Block: 0,
	}

	subNetChainConfig = ChainConfig{
		ChainId:          "9500",
		NetworkId:        "9500",
		PHub:             "ws://gate.tuntunhz.com:8899",
		OriginalChainId:  "9500",
		mainNodeContract: HexToAddress("0x27B01A9E699F177634f480Cc2150425009Edc5fD"),

		Proposal001Block: 0,
		Proposal002Block: 0,
		Proposal003Block: 0,
		Proposal004Block: 0,
		Proposal005Block: 0,
		Proposal006Block: 0,
		Proposal007Block: 0,
		Proposal008Block: 0,
		Proposal009Block: 0,
		Proposal010Block: 0,
		Proposal011Block: 0,
		Proposal012Block: 0,
		Proposal013Block: 0,
		Proposal014Block: 0,
		Proposal015Block: 0,
		Proposal016Block: 0,
		Proposal017Block: 0,
		Proposal018Block: 0,
		Proposal019Block: 0,
		Proposal021Block: 0,
	}

	LocalChainConfig ChainConfig

	filename = "genesis.json"

	Genesis *GenesisConf
)

type ChainConfig struct {
	ChainId   string
	NetworkId string

	PHub string

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
	Proposal010Block uint64
	Proposal011Block uint64
	Proposal012Block uint64
	Proposal013Block uint64
	Proposal014Block uint64
	Proposal015Block uint64
	Proposal016Block uint64
	Proposal017Block uint64
	Proposal018Block uint64
	Proposal019Block uint64
	Proposal021Block uint64

	mainNodeContract Address

	MysqlDSN   string
	JsonRPCUrl string
}

func initChainConfig(env string) {
	if env == ENV_DEV {
		LocalChainConfig = devNetChainConfig
	} else if env == ENV_MAINNET {
		LocalChainConfig = mainNetChainConfig
	} else if env == ENV_TESTNET_ROBIN {
		LocalChainConfig = robinChainConfig
	} else {
		LocalChainConfig = subNetChainConfig
	}

	localChainInfo = chainInfo{
		currentBlockHeight: atomic.Value{},
	}
	localChainInfo.currentBlockHeight.Store(uint64(0))
	blockHeightLogger = log.GetLoggerByIndex(log.BlockHeightConfig, strconv.Itoa(InstanceIndex))

	Genesis = getGenesisConf(filename)
	if nil == Genesis {
		fmt.Println("no genesisConf, using default")
	} else if 0 != len(Genesis.ChainId) {
		LocalChainConfig.NetworkId = Genesis.ChainId
		LocalChainConfig.ChainId = Genesis.ChainId
	}
}

// reading genesis info
func getGenesisConf(name string) *GenesisConf {
	file, err := os.Open(name)
	if err != nil {
		fmt.Println("no such file: " + name)
		return nil
	}

	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println("file: " + name + "reading error: " + err.Error())
		return nil
	}

	fmt.Println("get genesis.json")

	var genesisConf GenesisConf
	err = json.Unmarshal(content, &genesisConf)
	if err != nil {
		fmt.Println("file: " + name + " reading error: " + err.Error())
		return nil
	}

	return &genesisConf
}

func GetChainId(height uint64) *big.Int {
	var chainIdStr string
	if nil != Genesis && 0 != len(Genesis.ChainId) {
		chainIdStr = Genesis.ChainId
	} else {
		chainIdStr = ChainId(height)
	}

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
func IsSub() bool {
	return Genesis != nil
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

func IsProposal012() bool {
	return isForked(LocalChainConfig.Proposal012Block, GetBlockHeight())
}

func IsProposal013() bool {
	return isForked(LocalChainConfig.Proposal013Block, GetBlockHeight())
}

func IsProposal014() bool {
	return isForked(LocalChainConfig.Proposal014Block, GetBlockHeight())
}

func IsProposal015() bool {
	return isForked(LocalChainConfig.Proposal015Block, GetBlockHeight())
}

func IsProposal016() bool {
	return isForked(LocalChainConfig.Proposal016Block, GetBlockHeight())
}

func IsProposal017() bool {
	return isForked(LocalChainConfig.Proposal017Block, GetBlockHeight())
}

func IsProposal018() bool {
	return isForked(LocalChainConfig.Proposal018Block, GetBlockHeight())
}

func IsProposal021() bool {
	return isForked(LocalChainConfig.Proposal021Block, GetBlockHeight())
}

func isForked(base uint64, height uint64) bool {
	return height >= base
}

func MainNodeContract() Address {
	return LocalChainConfig.mainNodeContract
}
