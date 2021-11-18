package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"com.tuntun.rocket/node/src/consensus/vrf"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/storage/trie"
	"com.tuntun.rocket/node/src/utility"
	"com.tuntun.rocket/node/src/vm"
	"math/big"
	"time"
)

func genRobinGenesisBlock(stateDB *account.AccountDB, triedb *trie.NodeDatabase, genesisInfo []*types.GenesisInfo) *types.Block {
	block := new(types.Block)
	pv := big.NewInt(0)
	block.Header = &types.BlockHeader{
		Height:       0,
		ExtraData:    common.Sha256([]byte("Rocket Protocol")),
		CurTime:      time.Date(2021, 10, 30, 2, 0, 0, 0, time.UTC),
		ProveValue:   pv,
		TotalQN:      0,
		Transactions: make([]common.Hashes, 0), //important!!
		EvictedTxs:   make([]common.Hash, 0),   //important!!
		Nonce:        ChainDataVersion,
	}

	block.Header.RequestIds = make(map[string]uint64)
	block.Header.Signature = common.Sha256([]byte("tuntunhz"))
	block.Header.Random = common.Sha256([]byte("RocketProtocolVRF"))

	genesisProposers := getRobinGenesisProposer()
	addMiners(genesisProposers, stateDB)

	verifyMiners := make([]*types.Miner, 0)
	for _, genesis := range genesisInfo {
		for i, member := range genesis.Group.Members {
			miner := &types.Miner{Type: common.MinerTypeValidator, Id: member, PublicKey: genesis.Pks[i], VrfPublicKey: genesis.VrfPKs[i], Stake: common.ValidatorStake * uint64(i+2)}
			verifyMiners = append(verifyMiners, miner)
		}
	}
	addMiners(verifyMiners, stateDB)

	stateDB.SetNonce(common.ProposerDBAddress, 1)
	stateDB.SetNonce(common.ValidatorDBAddress, 1)

	//创建创始合约
	usdtContractAddress, wethContractAddress, mixContractAddress, bscUsdtContractAddress, wBNBContractAddress, bscMixContractAddress := createRobinGenesisContract(block.Header, stateDB)
	stateDB.AddERC20Binding("SYSTEM-ETH.USDT", usdtContractAddress, 2, 6)
	stateDB.AddERC20Binding("ETH.ETH", wethContractAddress, 3, 18)
	stateDB.AddERC20Binding("SYSTEM-ETH.MIX", mixContractAddress, 0, 18)
	stateDB.AddERC20Binding("SYSTEM-BSC.USDT", bscUsdtContractAddress, 2, 6)
	stateDB.AddERC20Binding("BSC.BNB", wBNBContractAddress, 3, 18)
	stateDB.AddERC20Binding("SYSTEM-BSC.MIX", bscMixContractAddress, 0, 18)

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

func createRobinGenesisContract(header *types.BlockHeader, statedb *account.AccountDB) (common.Address, common.Address, common.Address, common.Address, common.Address, common.Address) {
	source := "0x38780174572fb5b4735df1b7c69aee77ff6e9f49"
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
	logger.Debugf("After execute usdt contract create!Contract address:%s", usdtContractAddress.GetHexString())

	_, wethContractAddress, _, _, err := vmInstance.Create(caller, common.FromHex(wethContractData), vmCtx.GasLimit, big.NewInt(0))
	if err != nil {
		panic("Genesis contract create error:" + err.Error())
	}
	logger.Debugf("After execute weth contract create! Contract address:%s", wethContractAddress.GetHexString())

	_, mixContractAddress, _, _, err := vmInstance.Create(caller, common.FromHex(mixContractData), vmCtx.GasLimit, big.NewInt(0))
	if err != nil {
		panic("Genesis contract create error:" + err.Error())
	}
	logger.Debugf("After execute mix contract create! Contract address:%s", mixContractAddress.GetHexString())

	_, bscUsdtContractAddress, _, _, err := vmInstance.Create(caller, common.FromHex(usdtContractData), vmCtx.GasLimit, big.NewInt(0))
	if err != nil {
		panic("Genesis contract create error:" + err.Error())
	}
	logger.Debugf("After execute BSC usdt contract create!Contract address:%s", bscUsdtContractAddress.GetHexString())

	_, wBNBContractAddress, _, _, err := vmInstance.Create(caller, common.FromHex(wBNBContractData), vmCtx.GasLimit, big.NewInt(0))
	if err != nil {
		panic("Genesis contract create error:" + err.Error())
	}
	logger.Debugf("After execute wBNB contract create! Contract address:%s", wBNBContractAddress.GetHexString())

	_, bscMixContractAddress, _, _, err := vmInstance.Create(caller, common.FromHex(mixContractData), vmCtx.GasLimit, big.NewInt(0))
	if err != nil {
		panic("Genesis contract create error:" + err.Error())
	}
	logger.Debugf("After execute  BSC mix contract create! Contract address:%s", bscMixContractAddress.GetHexString())
	return usdtContractAddress, wethContractAddress, mixContractAddress, bscUsdtContractAddress, wBNBContractAddress, bscMixContractAddress
}

func addRobinTestAsset(stateDB *account.AccountDB) {
	valueTenThousand, _ := utility.StrToBigInt("10000")
	valueBillion, _ := utility.StrToBigInt("1000000000")
	/**
	  id:0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443
	  address:0x38780174572fb5b4735df1b7c69aee77ff6e9f49
	  sk:0xe7260a418579c2e6ca36db4fe0bf70f84d687bdf7ec6c0c181b43ee096a84aea
	*/
	stateDB.SetFT(common.HexToAddress("0x38780174572fb5b4735df1b7c69aee77ff6e9f49"), "ETH.ETH", valueTenThousand)
	stateDB.SetFT(common.HexToAddress("0x38780174572fb5b4735df1b7c69aee77ff6e9f49"), "SYSTEM-ETH.USDT", valueTenThousand)
	stateDB.SetFT(common.HexToAddress("0x38780174572fb5b4735df1b7c69aee77ff6e9f49"), "SYSTEM-ETH.MIX", valueBillion)
	stateDB.SetFT(common.HexToAddress("0x38780174572fb5b4735df1b7c69aee77ff6e9f49"), "BSC.BNB", valueTenThousand)
	stateDB.SetFT(common.HexToAddress("0x38780174572fb5b4735df1b7c69aee77ff6e9f49"), "SYSTEM-BSC.USDT", valueTenThousand)
	stateDB.SetFT(common.HexToAddress("0x38780174572fb5b4735df1b7c69aee77ff6e9f49"), "SYSTEM-BSC.MIX", valueBillion)
	stateDB.SetBalance(common.HexToAddress("0x38780174572fb5b4735df1b7c69aee77ff6e9f49"), valueBillion)

	/**
	id:0x7dba6865f337148e5887d6bea97e6a98701a2fa774bd00474ea68bcc645142f2
	address:0x2c616a97d3d10e008f901b392986b1a65e0abbb7
	sk:0x083f3fb13ffa99a18283a7fd5e2f831a52f39afdd90f5310a3d8fd4ffbd00d49
	*/
	stateDB.SetFT(common.HexToAddress("0x2c616a97d3d10e008f901b392986b1a65e0abbb7"), "ETH.ETH", valueBillion)
	stateDB.SetFT(common.HexToAddress("0x2c616a97d3d10e008f901b392986b1a65e0abbb7"), "SYSTEM-ETH.USDT", valueBillion)
	stateDB.SetFT(common.HexToAddress("0x2c616a97d3d10e008f901b392986b1a65e0abbb7"), "SYSTEM-ETH.MIX", valueBillion)
	stateDB.SetFT(common.HexToAddress("0x2c616a97d3d10e008f901b392986b1a65e0abbb7"), "BSC.BNB", valueTenThousand)
	stateDB.SetFT(common.HexToAddress("0x2c616a97d3d10e008f901b392986b1a65e0abbb7"), "SYSTEM-BSC.USDT", valueTenThousand)
	stateDB.SetFT(common.HexToAddress("0x2c616a97d3d10e008f901b392986b1a65e0abbb7"), "SYSTEM-BSC.MIX", valueBillion)
	stateDB.SetBalance(common.HexToAddress("0x2c616a97d3d10e008f901b392986b1a65e0abbb7"), valueBillion)

	/**
	address:0xb726d8add2d0da0e3497b8686e0440c1703348c6
	sk:0xe64b395c653ac649b6fd378bedfb2f93db298711b3a083229899d0c600e026d9
	*/
	stateDB.SetFT(common.HexToAddress("0xb726d8add2d0da0e3497b8686e0440c1703348c6"), "ETH.ETH", valueBillion)
	stateDB.SetFT(common.HexToAddress("0xb726d8add2d0da0e3497b8686e0440c1703348c6"), "SYSTEM-ETH.USDT", valueBillion)
	stateDB.SetFT(common.HexToAddress("0xb726d8add2d0da0e3497b8686e0440c1703348c6"), "SYSTEM-ETH.MIX", valueBillion)
	stateDB.SetFT(common.HexToAddress("0xb726d8add2d0da0e3497b8686e0440c1703348c6"), "BSC.BNB", valueTenThousand)
	stateDB.SetFT(common.HexToAddress("0xb726d8add2d0da0e3497b8686e0440c1703348c6"), "SYSTEM-BSC.USDT", valueTenThousand)
	stateDB.SetFT(common.HexToAddress("0xb726d8add2d0da0e3497b8686e0440c1703348c6"), "SYSTEM-BSC.MIX", valueBillion)
	stateDB.SetBalance(common.HexToAddress("0xb726d8add2d0da0e3497b8686e0440c1703348c6"), valueBillion)

	assetCreatTime := "1634686092119"
	service.FTManagerInstance.PublishFTSet(service.FTManagerInstance.GenerateFTSet("testFT", "alpha", "0x38780174572fb5b4735df1b7c69aee77ff6e9f49", "0", "0x38780174572fb5b4735df1b7c69aee77ff6e9f49", assetCreatTime, 0), stateDB)
	service.FTManagerInstance.PublishFTSet(service.FTManagerInstance.GenerateFTSet("testFT", "beta", "0x38780174572fb5b4735df1b7c69aee77ff6e9f49", "0", "0x38780174572fb5b4735df1b7c69aee77ff6e9f49", assetCreatTime, 0), stateDB)
	service.FTManagerInstance.PublishFTSet(service.FTManagerInstance.GenerateFTSet("testFT", "gamma", "0x38780174572fb5b4735df1b7c69aee77ff6e9f49", "0", "0x38780174572fb5b4735df1b7c69aee77ff6e9f49", assetCreatTime, 0), stateDB)
	service.FTManagerInstance.PublishFTSet(service.FTManagerInstance.GenerateFTSet("testFT", "delta", "0x38780174572fb5b4735df1b7c69aee77ff6e9f49", "0", "0x38780174572fb5b4735df1b7c69aee77ff6e9f49", assetCreatTime, 0), stateDB)
	service.FTManagerInstance.PublishFTSet(service.FTManagerInstance.GenerateFTSet("testFT", "epsilon", "0x38780174572fb5b4735df1b7c69aee77ff6e9f49", "0", "0x38780174572fb5b4735df1b7c69aee77ff6e9f49", assetCreatTime, 0), stateDB)

	//NFT Asset
	service.NFTManagerInstance.PublishNFTSet(service.NFTManagerInstance.GenerateNFTSet("262beed0-703e-417f-9258-89ad1f736982", "testNFT", "alpha", "0x38780174572fb5b4735df1b7c69aee77ff6e9f49", "0x38780174572fb5b4735df1b7c69aee77ff6e9f49", types.NFTConditions{}, 0, assetCreatTime), stateDB)
	service.NFTManagerInstance.PublishNFTSet(service.NFTManagerInstance.GenerateNFTSet("59c641ee-7b1b-444b-9f87-47889539df1f", "testNFT", "beta", "0x38780174572fb5b4735df1b7c69aee77ff6e9f49", "0x38780174572fb5b4735df1b7c69aee77ff6e9f49", types.NFTConditions{}, 0, assetCreatTime), stateDB)
	service.NFTManagerInstance.PublishNFTSet(service.NFTManagerInstance.GenerateNFTSet("0d19bbce-6c3d-4153-8168-1c7a33448fa4", "testNFT", "gamma", "0x38780174572fb5b4735df1b7c69aee77ff6e9f49", "0x38780174572fb5b4735df1b7c69aee77ff6e9f49", types.NFTConditions{}, 0, assetCreatTime), stateDB)
	service.NFTManagerInstance.PublishNFTSet(service.NFTManagerInstance.GenerateNFTSet("a919a5a0-a5ed-40b3-be4d-403985063863", "testNFT", "delta", "0x38780174572fb5b4735df1b7c69aee77ff6e9f49", "0x38780174572fb5b4735df1b7c69aee77ff6e9f49", types.NFTConditions{}, 0, assetCreatTime), stateDB)
	service.NFTManagerInstance.PublishNFTSet(service.NFTManagerInstance.GenerateNFTSet("f5626c4d-3895-4376-a5df-fc1f04e0f375", "testNFT", "epsilon", "0x38780174572fb5b4735df1b7c69aee77ff6e9f49", "0x38780174572fb5b4735df1b7c69aee77ff6e9f49", types.NFTConditions{}, 0, assetCreatTime), stateDB)
}
