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

package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"com.tuntun.rocket/node/src/consensus/vrf"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/storage/trie"
	"math/big"
	"time"
)

const ChainDataVersion = 2

var emptyHash = common.Hash{}

type GenesisProposer struct {
	MinerId     string `yaml:"minerId"`
	MinerPubKey string `yaml:"minerPubkey"`
	VRFPubkey   string `yaml:"vrfPubkey"`
}

func (chain *blockChain) insertGenesisBlock() {
	state, err := service.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	if nil == err {
		genesisBlock := genGenesisBlock(state, service.AccountDBManagerInstance.GetTrieDB(), consensusHelper.GenerateGenesisInfo())
		logger.Debugf("GenesisBlock Hash:%s,StateTree:%s", genesisBlock.Header.Hash.String(), genesisBlock.Header.StateTree.Hex())
		blockByte, _ := types.MarshalBlock(genesisBlock)
		chain.saveBlockByHash(genesisBlock.Header.Hash, blockByte)

		headerByte, err := types.MarshalBlockHeader(genesisBlock.Header)
		if err != nil {
			logger.Errorf("Marshal block header error:%s", err.Error())
		}
		chain.saveBlockByHeight(genesisBlock.Header.Height, headerByte)

		chain.updateLastBlock(state, genesisBlock, headerByte)
		chain.updateVerifyHash(genesisBlock)
	} else {
		panic("Init block chain error:" + err.Error())
	}
}

func genGenesisBlock(stateDB *account.AccountDB, triedb *trie.NodeDatabase, genesisInfo []*types.GenesisInfo) *types.Block {
	block := new(types.Block)
	pv := big.NewInt(0)
	block.Header = &types.BlockHeader{
		Height:       0,
		ExtraData:    common.Sha256([]byte("Rocket Protocol")),
		CurTime:      time.Date(2020, 4, 10, 10, 0, 0, 0, time.Local),
		ProveValue:   pv,
		TotalQN:      0,
		Transactions: make([]common.Hashes, 0), //important!!
		EvictedTxs:   make([]common.Hash, 0),   //important!!
		Nonce:        ChainDataVersion,
	}

	block.Header.RequestIds = make(map[string]uint64)
	block.Header.Signature = common.Sha256([]byte("tuntunhz"))
	block.Header.Random = common.Sha256([]byte("RocketProtocolVRF"))

	genesisProposers := getGenesisProposer()
	addMiners(genesisProposers, stateDB)

	verifyMiners := make([]*types.Miner, 0)
	for _, genesis := range genesisInfo {
		for i, member := range genesis.Group.Members {
			miner := &types.Miner{Type: common.MinerTypeValidator, Id: member, PublicKey: genesis.Pks[i], VrfPublicKey: genesis.VrfPKs[i], Stake: common.ValidatorStake * uint64(i+2)}
			verifyMiners = append(verifyMiners, miner)
		}
	}
	addMiners(verifyMiners, stateDB)

	//addTestMiners(stateDB)

	stateDB.SetNonce(common.ProposerDBAddress, 1)
	stateDB.SetNonce(common.ValidatorDBAddress, 1)

	// 测试用
	service.FTManagerInstance.PublishFTSet(service.FTManagerInstance.GenerateFTSet("tuntun", "pig", "hz", "0", "hz", "10086", 0), stateDB)
	service.NFTManagerInstance.PublishNFTSet(service.NFTManagerInstance.GenerateNFTSet("tuntunhz", "tuntun", "t", "hz", "hz", types.NFTConditions{}, 0, "10000"), stateDB)
	stateDB.SetBNT(common.HexToAddress("0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54"), "ETH.ETH", big.NewInt(10000000000))
	/**
		测试账户
		id:0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443
	    sk:0xe7260a418579c2e6ca36db4fe0bf70f84d687bdf7ec6c0c181b43ee096a84aea
	*/
	stateDB.SetBNT(common.HexToAddress("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443"), "ETH.ETH", big.NewInt(10000000000))
	stateDB.SetFT(common.HexToAddress("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443"), "SYSTEM-ETH.USDT", big.NewInt(10000000000))
	stateDB.SetBalance(common.HexToAddress("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443"), big.NewInt(1000000000000000000))

	//客户端测试使用账号
	stateDB.SetBNT(common.HexToAddress("0xe7260a418579c2e6ca36db4fe0bf70f84d687bdf7ec6c0c181b43ee096a84aea"), "ETH.ETH", big.NewInt(10000000000))
	stateDB.SetFT(common.HexToAddress("0xe7260a418579c2e6ca36db4fe0bf70f84d687bdf7ec6c0c181b43ee096a84aea"), "SYSTEM-ETH.USDT", big.NewInt(10000000000))
	stateDB.SetBalance(common.HexToAddress("0xe7260a418579c2e6ca36db4fe0bf70f84d687bdf7ec6c0c181b43ee096a84aea"), big.NewInt(1000000000000000000))

	stateDB.SetBalance(common.HexToAddress("0xa72263e15d3a48de8cbde1f75c889c85a15567b990aab402691938f4511581ab"), big.NewInt(1000000000000000000))
	/**
	自动化测试框架专用账户
	id:0x7dba6865f337148e5887d6bea97e6a98701a2fa774bd00474ea68bcc645142f2
	sk:0x083f3fb13ffa99a18283a7fd5e2f831a52f39afdd90f5310a3d8fd4ffbd00d49
	*/
	stateDB.SetBNT(common.HexToAddress("0x7dba6865f337148e5887d6bea97e6a98701a2fa774bd00474ea68bcc645142f2"), "ETH.ETH", big.NewInt(10000000000))
	stateDB.SetFT(common.HexToAddress("0x7dba6865f337148e5887d6bea97e6a98701a2fa774bd00474ea68bcc645142f2"), "SYSTEM-ETH.USDT", big.NewInt(10000000000))
	stateDB.SetBalance(common.HexToAddress("0x7dba6865f337148e5887d6bea97e6a98701a2fa774bd00474ea68bcc645142f2"), big.NewInt(1000000000000000000))

	root, _ := stateDB.Commit(true)
	triedb.Commit(root, false)
	block.Header.StateTree = common.BytesToHash(root.Bytes())
	block.Header.Hash = block.Header.GenHash()

	return block
}

func getGenesisProposer() []*types.Miner {
	genesisProposers := make([]GenesisProposer, 1)
	genesisProposer := GenesisProposer{}
	genesisProposer.MinerId = "0xe059d17139e2915d270ef8f3eee2f3e1438546ba2f06eb674dda0967846b6951"
	genesisProposer.MinerPubKey = "0x16d0b0a106e2de32b42ea4096c9e80c883c6ffa9e3f19f09cb45dfff2b02d09a3bcf95f2d0c33b7caf5db42d55d3459395c1b8d6a5d315a113edc39c4ce3a3d5269ab4a9514a998fdcc693d90a42505185270a184a07ddfb553b181be13e968480ef0df4c06cf657957b07118776a38fea3bcf758ea4491a4213719e2f6537b5"
	genesisProposer.VRFPubkey = "0x009f3b76f3e49dcdd6d2ee8421f077fd4c68c176b18e1e602a3c1f09f9272250"
	genesisProposers[0] = genesisProposer

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
			VrfPublicKey: vrfPubkey,
			ApplyHeight:  0,
			Stake:        common.ProposerStake,
			Type:         common.MinerTypeProposer,
			Status:       common.MinerStatusNormal,
		}
		miners = append(miners, &miner)
	}
	return miners
}

func addMiners(miners []*types.Miner, accountdb *account.AccountDB) {
	for _, miner := range miners {
		service.MinerManagerImpl.InsertMiner(miner, accountdb)
	}
}
