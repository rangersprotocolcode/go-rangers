package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"com.tuntun.rocket/node/src/consensus/vrf"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/storage/trie"
	"com.tuntun.rocket/node/src/utility"
	"com.tuntun.rocket/node/src/vm"
	"math/big"
	"time"
)

func genGenesisBlock(stateDB *account.AccountDB, triedb *trie.NodeDatabase, genesisInfo []*types.GenesisInfo) *types.Block {
	block := new(types.Block)
	pv := big.NewInt(0)
	block.Header = &types.BlockHeader{
		Height:       0,
		ExtraData:    common.Sha256([]byte("Rangers Protocol")),
		CurTime:      time.Date(2021, 11, 29, 2, 0, 0, 0, time.UTC),
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
	createGenesisContract(block.Header, stateDB)

	genesisProposers := getGenesisProposer()
	addMiners(genesisProposers, stateDB)

	verifyMiners := make([]*types.Miner, 0)
	for _, genesis := range genesisInfo {
		for i, member := range genesis.Group.Members {
			miner := &types.Miner{Type: common.MinerTypeValidator, Id: member, PublicKey: genesis.Pks[i], VrfPublicKey: genesis.VrfPKs[i], Stake: common.ValidatorStake * uint64(i+2)}
			miner.Account = common.FromHex("0x42c8c9b13fc0573d18028b3398a887c4297ff646")
			verifyMiners = append(verifyMiners, miner)
		}
	}
	addMiners(verifyMiners, stateDB)

	stateDB.SetNonce(common.ProposerDBAddress, 1)
	stateDB.SetNonce(common.ValidatorDBAddress, 1)

	money, _ := utility.StrToBigInt("100000000")
	stateDB.SetBalance(common.HexToAddress("0x38780174572fb5b4735df1b7c69aee77ff6e9f49"), money)
	stateDB.SetBalance(common.HexToAddress("0x42c8c9b13fc0573d18028b3398a887c4297ff646"), money)

	root, _ := stateDB.Commit(true)
	triedb.Commit(root, false)
	block.Header.StateTree = common.BytesToHash(root.Bytes())
	block.Header.Hash = block.Header.GenHash()

	return block
}

func getGenesisProposer() []*types.Miner {
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
			Account:      common.FromHex("0x38780174572fb5b4735df1b7c69aee77ff6e9f49"),
		}
		miners = append(miners, &miner)
	}
	return miners
}

func createGenesisContract(header *types.BlockHeader, statedb *account.AccountDB) {
	source := "0x42c8c9b13fc0573d18028b3398a887c4297ff646"
	vmCtx := vm.Context{}
	vmCtx.CanTransfer = vm.CanTransfer
	vmCtx.Transfer = vm.Transfer
	vmCtx.GetHash = func(uint64) common.Hash { return emptyHash }

	vmCtx.Origin = common.HexToAddress(source)
	vmCtx.Coinbase = common.BytesToAddress(header.Castor)
	vmCtx.BlockNumber = new(big.Int).SetUint64(header.Height)
	vmCtx.Time = new(big.Int).SetUint64(uint64(header.CurTime.Unix()))

	vmCtx.GasPrice = big.NewInt(1)
	vmCtx.GasLimit = 30000000
	vmInstance := vm.NewEVM(vmCtx, statedb)
	caller := vm.AccountRef(vmCtx.Origin)

	_, usdtContractAddress, _, _, err := vmInstance.Create(caller, common.FromHex(usdtContractData), vmCtx.GasLimit, big.NewInt(0))
	if err != nil {
		panic("Genesis contract create error:" + err.Error())
	}
	// 0xdf764badfdf3c27753f9c4a269a850d028f01dbc
	logger.Debugf("After execute usdt contract create!Contract address:%s", usdtContractAddress.GetHexString())

	// 0xf800eddcdbd86fc46df366526f709bef33bd3d45
	_, wRpgContractAddress, _, _, err := vmInstance.Create(caller, common.FromHex(wRPGContractData), vmCtx.GasLimit, big.NewInt(0))
	statedb.AddERC20Binding(common.BLANCE_NAME, wRpgContractAddress, 3, 18)
	if err != nil {
		panic("Genesis contract create error:" + err.Error())
	}
	logger.Debugf("After execute rpg contract create! Contract address:%s", wRpgContractAddress.GetHexString())

	_, proxyContractAddress, _, _, err := vmInstance.Create(caller, common.FromHex(proxyData), vmCtx.GasLimit, big.NewInt(0))
	if err != nil {
		panic("Genesis contract create error:" + err.Error())
	}
	// address:0x27b01a9e699f177634f480cc2150425009edc5fd
	logger.Debugf("After execute proxy contract create!Contract address:%s", proxyContractAddress.GetHexString())

}
