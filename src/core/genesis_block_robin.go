package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"com.tuntun.rocket/node/src/consensus/vrf"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/storage/trie"
	"com.tuntun.rocket/node/src/utility"
	"math/big"
	"time"
)

var robinProposerAccounts = [1]string{
	`0xa9e11ce87c646ca4b0c8eb66f28a86232734d74a`,
}

var robinValidatorAccounts = [3]string{
	`0x56b1fc865ad0c87f46f804145a861b38fcbafb99`,
	`0x0ef90c9cc936c2e3117d76c1ffb28391f8cebbca`,
	`0x22b00137e24a708609fdb88ee156dabe041b158b`,
}

func genRobinGenesisBlock(stateDB *account.AccountDB, triedb *trie.NodeDatabase, genesisInfo []*types.GenesisInfo) *types.Block {
	block := new(types.Block)
	pv := big.NewInt(0)
	block.Header = &types.BlockHeader{
		Height:       0,
		ExtraData:    common.Sha256([]byte("Rangers Protocol")),
		CurTime:      time.Date(2021, 12, 15, 0, 0, 0, 0, time.UTC),
		ProveValue:   pv,
		TotalQN:      0,
		Transactions: make([]common.Hashes, 0), //important!!
		EvictedTxs:   make([]common.Hash, 0),   //important!!
		Nonce:        ChainDataVersion,
	}

	block.Header.RequestIds = make(map[string]uint64)
	block.Header.Signature = common.Sha256([]byte("tuntunhz"))
	block.Header.Random = common.Sha256([]byte("RangersProtocolVRF"))

	//创建创始合约
	proxy := createGenesisContract(block.Header, stateDB)

	genesisProposers := getRobinGenesisProposer()
	addMiners(genesisProposers, stateDB)

	verifyMiners := make([]*types.Miner, 0)
	for _, genesis := range genesisInfo {
		for i, member := range genesis.Group.Members {
			miner := &types.Miner{Type: common.MinerTypeValidator, Id: member, PublicKey: genesis.Pks[i], VrfPublicKey: genesis.VrfPKs[i], Stake: common.ValidatorStake}
			miner.Account = common.FromHex(robinValidatorAccounts[i])
			verifyMiners = append(verifyMiners, miner)
		}
	}
	addMiners(verifyMiners, stateDB)

	stateDB.SetNonce(common.ProposerDBAddress, 1)
	stateDB.SetNonce(common.ValidatorDBAddress, 1)

	// 跨链手续费地址
	two, _ := utility.StrToBigInt("2")
	stateDB.SetBalance(common.HexToAddress("0x7edd0ef9da9cec334a7887966cc8dd71d590eeb7"), two)

	// 21000000*51%-(2000*20+400*20)-2
	money, _ := utility.StrToBigInt("10661998")
	stateDB.SetBalance(proxy, money)

	addRobinTestAsset(stateDB)

	root, _ := stateDB.Commit(true)
	triedb.Commit(root, false)
	block.Header.StateTree = common.BytesToHash(root.Bytes())
	block.Header.Hash = block.Header.GenHash()

	return block
}

func getRobinGenesisProposer() []*types.Miner {
	genesisProposers := make([]GenesisProposer, 1)
	genesisProposer := GenesisProposer{}
	genesisProposer.MinerId = "0x7f88b4f2d36a83640ce5d782a0a20cc2b233de3df2d8a358bf0e7b29e9586a12"
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

func addRobinTestAsset(stateDB *account.AccountDB) {
	valueBillion, _ := utility.StrToBigInt("1000000000")
	stateDB.SetBalance(common.HexToAddress("0x2f4f09b722a6e5b77be17c9a99c785fa7035a09f"), valueBillion)
	stateDB.SetBalance(common.HexToAddress("0x42c8c9b13fc0573d18028b3398a887c4297ff646"), valueBillion)
	//used for faucet
	stateDB.SetBalance(common.HexToAddress("0x8744c51069589296fcb7faa2f891b1f513a0310c"), valueBillion)

	stateDB.SetBalance(common.HexToAddress("0x25716527aad0ae1dd24bd247af9232dae78595b0"), valueBillion)
}
