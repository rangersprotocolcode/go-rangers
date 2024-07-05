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

package core

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/consensus/vrf"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/storage/account"
	"com.tuntun.rangers/node/src/storage/trie"
	"com.tuntun.rangers/node/src/utility"
	"encoding/json"
	"math/big"
	"time"
)

var devProposerInfo = [1]string{
	`0xa9e11ce87c646ca4b0c8eb66f28a86232734d74a`,
}

var devValidatorAccounts = [3]string{
	`0x56b1fc865ad0c87f46f804145a861b38fcbafb99`,
	`0x0ef90c9cc936c2e3117d76c1ffb28391f8cebbca`,
	`0x22b00137e24a708609fdb88ee156dabe041b158b`,
}

func genDevGenesisBlock(stateDB *account.AccountDB, triedb *trie.NodeDatabase, genesisInfo []*types.GenesisInfo) *types.Block {
	block := new(types.Block)
	pv := big.NewInt(0)
	block.Header = &types.BlockHeader{
		Height:       0,
		ExtraData:    common.Sha256([]byte("Rangers Protocol")),
		CurTime:      time.Date(2024, 7, 5, 0, 0, 0, 0, time.UTC),
		ProveValue:   pv,
		TotalQN:      0,
		Transactions: make([]common.Hashes, 0), //important!!
		EvictedTxs:   make([]common.Hash, 0),   //important!!
		Nonce:        ChainDataVersion,
	}

	block.Header.RequestIds = make(map[string]uint64)
	block.Header.Signature = common.Sha256([]byte("tuntunhz"))
	block.Header.Random = common.Sha256([]byte("RangersProtocolVRF"))

	proxy := createGenesisContract(block.Header, stateDB)

	genesisProposers := getDevGenesisOneProposer()
	addMiners(genesisProposers, stateDB)

	verifyMiners := make([]*types.Miner, 0)
	for _, genesis := range genesisInfo {
		for i, member := range genesis.Group.Members {
			miner := &types.Miner{Type: common.MinerTypeValidator, Id: member, PublicKey: genesis.Pks[i], VrfPublicKey: genesis.VrfPKs[i], Stake: common.ValidatorStake * 5}
			miner.Account = common.FromHex(devValidatorAccounts[i])
			verifyMiners = append(verifyMiners, miner)
		}
	}
	addMiners(verifyMiners, stateDB)

	stateDB.SetNonce(common.ProposerDBAddress, 1)
	stateDB.SetNonce(common.ValidatorDBAddress, 1)

	two, _ := utility.StrToBigInt("2")
	stateDB.SetBalance(common.HexToAddress("0x7edd0ef9da9cec334a7887966cc8dd71d590eeb7"), two)

	// 21000000*51%-(2000*20+400*20)-2
	money, _ := utility.StrToBigInt("10661998")
	stateDB.SetBalance(proxy, money)

	addDevTestAsset(stateDB)

	root, _ := stateDB.Commit(true)
	triedb.Commit(root, false)
	block.Header.StateTree = common.BytesToHash(root.Bytes())
	block.Header.Hash = block.Header.GenHash()

	return block
}

func getDevGenesisProposer() []*types.Miner {
	miners := make([]*types.Miner, 20)
	for i, data := range devProposerInfo {
		var gp ProposerData
		json.Unmarshal(utility.StrToBytes(data), &gp)

		var minerId groupsig.ID
		minerId.SetHexString(gp.Id)

		var minerPubkey groupsig.Pubkey
		minerPubkey.SetHexString(gp.PublicKey)

		vrfPubkey := vrf.Hex2VRFPublicKey(gp.VrfPublicKey)
		miner := types.Miner{
			Id:           minerId.Serialize(),
			PublicKey:    minerPubkey.Serialize(),
			VrfPublicKey: vrfPubkey.GetBytes(),
			ApplyHeight:  0,
			Stake:        common.ProposerStake,
			Type:         common.MinerTypeProposer,
			Status:       common.MinerStatusNormal,
			Account:      common.FromHex(gp.Account),
		}
		miners[i] = &miner
	}
	return miners
}

func addDevTestAsset(stateDB *account.AccountDB) {
	valueBillion, _ := utility.StrToBigInt("1000000000")
	stateDB.SetBalance(common.HexToAddress("0x2f4f09b722a6e5b77be17c9a99c785fa7035a09f"), valueBillion)
	stateDB.SetBalance(common.HexToAddress("0x42c8c9b13fc0573d18028b3398a887c4297ff646"), valueBillion)
	//used for faucet
	stateDB.SetBalance(common.HexToAddress("0x8744c51069589296fcb7faa2f891b1f513a0310c"), valueBillion)

	stateDB.SetBalance(common.HexToAddress("0x25716527aad0ae1dd24bd247af9232dae78595b0"), valueBillion)
}

func getDevGenesisOneProposer() []*types.Miner {
	genesisProposers := make([]GenesisProposer, 2)
	genesisProposer := GenesisProposer{}
	genesisProposer.MinerId = "0x7f88b4f2d36a83640ce5d782a0a20cc2b233de3df2d8a358bf0e7b29e9586a12"
	genesisProposer.MinerPubKey = "0x16d0b0a106e2de32b42ea4096c9e80c883c6ffa9e3f19f09cb45dfff2b02d09a3bcf95f2d0c33b7caf5db42d55d3459395c1b8d6a5d315a113edc39c4ce3a3d5269ab4a9514a998fdcc693d90a42505185270a184a07ddfb553b181be13e968480ef0df4c06cf657957b07118776a38fea3bcf758ea4491a4213719e2f6537b5"
	genesisProposer.VRFPubkey = "0x009f3b76f3e49dcdd6d2ee8421f077fd4c68c176b18e1e602a3c1f09f9272250"
	genesisProposers[0] = genesisProposer

	genesisProposer2 := GenesisProposer{}
	genesisProposer2.MinerId = "0xb26612d2742ab4edd016b354725d045d6627de9b1b2d7c40ae26d2c97af21abd"
	genesisProposer2.MinerPubKey = "0x02985100db85c1ac4ecda18701179f1055ca56fd7ca7cc77663a0f202e1ac65f754ac09f9c2a0db05074b4701f609ca97b90cf093aa16c161ec384a0ab8ee39b32dc6f91e8370fe874e83fb4eb6f7630af9640cc9142e785a66b8cb6c08902397078364d10f4bece03e5d849c8b2f171160cba9efd97d03b6e68110220b18440"
	genesisProposer2.VRFPubkey = "0x9ca3d8e00ba25a26d941e0aa9625ad2765c6f5a7fc7cc11c5e97a023173786ef"
	genesisProposers[1] = genesisProposer2

	miners := make([]*types.Miner, 0)
	for _, gp := range genesisProposers {
		var minerId groupsig.ID
		minerId.SetHexString(gp.MinerId)

		var minerPubkey groupsig.Pubkey
		minerPubkey.SetHexString(gp.MinerPubKey)

		vrfPubkey := vrf.Hex2VRFPublicKey(gp.VRFPubkey)
		miner := types.Miner{
			Id:           minerId.Serialize(),
			PublicKey:    minerPubkey.Serialize(),
			VrfPublicKey: vrfPubkey.GetBytes(),
			ApplyHeight:  0,
			Stake:        common.ProposerStake,
			Type:         common.MinerTypeProposer,
			Status:       common.MinerStatusNormal,
			Account:      common.FromHex(robinProposerAccounts[0]),
		}
		miners = append(miners, &miner)
	}
	return miners
}
