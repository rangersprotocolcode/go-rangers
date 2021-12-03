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
			miner := &types.Miner{Type: common.MinerTypeValidator, Id: member, PublicKey: genesis.Pks[i], VrfPublicKey: genesis.VrfPKs[i], Stake: common.ValidatorStake}
			miner.Account = common.FromHex("0x42c8c9b13fc0573d18028b3398a887c4297ff646")
			verifyMiners = append(verifyMiners, miner)
		}
	}
	addMiners(verifyMiners, stateDB)

	if common.IsMainnet() {
		addSecondGroupValidator(stateDB)
	}
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

func addSecondGroupValidator(stateDB *account.AccountDB) {
	verifyMiners := make([]*types.Miner, 0)
	miner11 := &types.Miner{Type: common.MinerTypeValidator, Stake: common.ValidatorStake}
	miner11.Account = []byte{1, 0, 0, byte(11)}
	miner11.Id = common.FromHex("0x245f61a18215af0d2531b8e6921b13b52cd58fa43c2e19cd75c9031040ecc7fa")
	miner11.PublicKey = common.FromHex("0x5a50b13fe58348098700e60d271cab6e1aa684490a058ab8b9a26ad163345a3866895c3cab2fcccbea1e001a13aa1811fd8a03f0ae57f40d69e161cf1b89dac680a8fbdd12d6b88527ef57cfb4637ad793368f3e55da4b359dbe7c8b0625ff6836df03ac8b3f7caa0af361c4231d32c80063079b67e0dbf2508503b633d6780f")
	miner11.VrfPublicKey = common.FromHex("0xb1fe9e6464fab928195812d7f5937896549b1c87b1fd639fe4a01a98c0a04f31")
	verifyMiners = append(verifyMiners, miner11)

	miner12 := &types.Miner{Type: common.MinerTypeValidator, Stake: common.ValidatorStake}
	miner12.Account = []byte{1, 0, 0, byte(12)}
	miner12.Id = common.FromHex("0x77f144dcc7a25692a70ac7324504e764a3f6db0b3f954de87e8dc1a25467fe1a")
	miner12.PublicKey = common.FromHex("0x6da381570d6a8b3c4332da524c0d79e81f8e72389bc8d8fa26328f16d8140ad326c7f923992dd3a78131b68cdb16c2fbcd3ae0e2548224d47717d60d1a204557594d85d60ed5a026168a31fcc1c7ef32e6efa61378e4fa289265a383fcb98234211faf8da063b93e30c68aa8103fdda19f42288e5e80767cb2fd54ed5756fc9d")
	miner12.VrfPublicKey = common.FromHex("0xe70552fa4e86337786786f46b5008bf6e5c8817ca4e84d5cd057bf349180c0db")
	verifyMiners = append(verifyMiners, miner12)

	miner13 := &types.Miner{Type: common.MinerTypeValidator, Stake: common.ValidatorStake}
	miner13.Account = []byte{1, 0, 0, byte(13)}
	miner13.Id = common.FromHex("0xc8ad0be14a695432843e76485644eb41618c0307ceadd49fae57961f8ef716d1")
	miner13.PublicKey = common.FromHex("0x2aa89a13a767d34d6382c2d4298ac6461b8973d77e683ef0c7907660c412879e0a44b4236d5a465946dd0c7d956f9f65e4868d7f8103f6664549af6907a2267d1e8f407eca4230641dbe287a0675e4de10a2791a057a74c72ab39cca80727c5b451f417fbc0537306ef5f9a3ed33bf0da8964079a7752dc84d19b3ae2120d969")
	miner13.VrfPublicKey = common.FromHex("0x7a622af41d1f8485129c089261aecc542a94312de99bc0fdcaeead9846993166")
	verifyMiners = append(verifyMiners, miner13)

	miner14 := &types.Miner{Type: common.MinerTypeValidator, Stake: common.ValidatorStake}
	miner14.Account = []byte{1, 0, 0, byte(14)}
	miner14.Id = common.FromHex("0xab2d103f32136d9d635eeadfd82d6ac340788d3bf70862270ee04b82e94c79ee")
	miner14.PublicKey = common.FromHex("0x2c481e9df39a57e3a8bdd9a3efa7a526ee19315193d4c27ff444419f2b06c57a1c91faed4cf38deed76bf29a31beda72f759518f18f23b41367aa74c668c95291ce55810bbd68d0fb0b9d935913e7ec08f58e39a8f8017844a33b642f87398b331bafbaf7e194522e4da90092c9011e306a6d00aa43c7b062e9dd15e1f836b59")
	miner14.VrfPublicKey = common.FromHex("0x173670a2db53e42118b23b8132cdcbf8b72621ce433a3b2bfd1228de093b1bc6")
	verifyMiners = append(verifyMiners, miner14)

	miner15 := &types.Miner{Type: common.MinerTypeValidator, Stake: common.ValidatorStake}
	miner15.Account = []byte{1, 0, 0, byte(15)}
	miner15.Id = common.FromHex("0x138f0771a33357766ae663f4889e8b8be2654ac401214557c407868504051455")
	miner15.PublicKey = common.FromHex("0x3eb26e8666d0d0d43918e7789dde9d7f5a1bd706f5e267bcd0a2fb93e6dcaada21c21219d8928a6b5ffead0ab76e2441769f0db53831178f0d0efc9b8e0f4d153b06a313091ac104ec03105de4ee5f3d7791986dcdb5fcae552863e21e8edd7081dff99b5eb8323699f450582160737397007cf38c84eef351541469544a0c5b")
	miner15.VrfPublicKey = common.FromHex("0x1fba14757c3744d60e7b0681bad15855a9d7f3e66fed7c24fa69e090e8ef1a35")
	verifyMiners = append(verifyMiners, miner15)

	miner16 := &types.Miner{Type: common.MinerTypeValidator, Stake: common.ValidatorStake}
	miner16.Account = []byte{1, 0, 0, byte(16)}
	miner16.Id = common.FromHex("0x6ddc3604fbb86696627828f7e9f33c85df6d66e19ecd12f29d7b095da32f3ed4")
	miner16.PublicKey = common.FromHex("0x8d5d7ea074b2126a9b0a0e7e7df203d330441cf2b2eb7bfdef81d240eefd5f3452cd858212645754e1cec7b849210365485bb694786dd724f8110e3208d406773f13624a8e328f7fb25318d7513860b3cdf4d4d8ba9702562c0abf0f446a8f6f5b15e2981018b902c26c142191f8075f4bbd739f40afee349c066cc69c6b084d")
	miner16.VrfPublicKey = common.FromHex("0x38d5248e0e94929221dec49613876c51b19f026809073ae46abcb7197215e36b")
	verifyMiners = append(verifyMiners, miner16)

	miner17 := &types.Miner{Type: common.MinerTypeValidator, Stake: common.ValidatorStake}
	miner17.Account = []byte{1, 0, 0, byte(17)}
	miner17.Id = common.FromHex("0x4d2300e3c25fd4214072e4adcf599abed366b5be1cea8de3829a80d48eca4205")
	miner17.PublicKey = common.FromHex("0x27f56504b235eefab1e0fb9b107d2206d33e5f8839fb93811d05f9a531513bbd061272930f6bbebb25fb64c07b36b2e4def75904dfe8c015647e7b31cba7c6bb8be22ba72de9ba2157005215607f9183579de77683dd93b9976e757a301183667cb60e26ebc9a87af73ca5412efef0f56ac93a20de5ef8bb14efd82a3fbfd637")
	miner17.VrfPublicKey = common.FromHex("0xd9b7c8403cb24cbf48f912cf82ea28b3124e83d04240359530d15ecafc7ea95b")
	verifyMiners = append(verifyMiners, miner17)

	miner18 := &types.Miner{Type: common.MinerTypeValidator, Stake: common.ValidatorStake}
	miner18.Account = []byte{1, 0, 0, byte(18)}
	miner18.Id = common.FromHex("0x3748f337f5915aaa3bc150fa22c7aebd1e3e397e3f61fb501eeacac1a268165e")
	miner18.PublicKey = common.FromHex("0x7674b7d4cd7ccab7d745cfae515fc1a7ff5e578b7d685c287eeaf0b88b0c6e3535842179e25b464b04fe502088f8d766b73b0e2b2c95e11d070b053da8bce07a00c70710fb442c07778e386a677973bf4934ec15d43f03428bd1bd9fdb7090f812f1f222ca7db9ca17a2ea4084e31795337cf8da2028dfb2465bd9afc9b8fbcc")
	miner18.VrfPublicKey = common.FromHex("0x9ba61a65c70bcad5775908bad5cdc6aae60d1f2480c35444802b915ac81277d9")
	verifyMiners = append(verifyMiners, miner18)

	miner19 := &types.Miner{Type: common.MinerTypeValidator, Stake: common.ValidatorStake}
	miner19.Account = []byte{1, 0, 0, byte(19)}
	miner19.Id = common.FromHex("0xae003a027036ed72eac4d367fffd6979993b62f94e98a10c8d189ee462d19d84")
	miner19.PublicKey = common.FromHex("0x88cd094ef69980ceaf64532d08c33866c411dc5dc8094cb2b2880f01e43152695bbe7cdc8030bebb9c3ec49cbd476e091e649f87199c560978458903b3dceab07fbdd5df292a84b0e1ee3066e637acb7aadf43f9e3dd3e8a3d66274078bc2290141c6fa9e04338574d65605b58f6cfd5368d89388bd090b9d79117157caa13a2")
	miner19.VrfPublicKey = common.FromHex("0x7af4f2f7c143dd208892c4323e4586e703e1fa5d01991f107a5c4dfa21b4ccc8")
	verifyMiners = append(verifyMiners, miner19)

	miner20 := &types.Miner{Type: common.MinerTypeValidator, Stake: common.ValidatorStake}
	miner20.Account = []byte{1, 0, 0, byte(20)}
	miner20.Id = common.FromHex("0xbff465dd49992a79958dfa3c7e72a994558885c9d63f6ec5325701a74dde732f")
	miner20.PublicKey = common.FromHex("0x713b8c73b2b2b989176769e27e69c95436cbb9b8c0d39e2a13aecaa18f87242f74a8526cd048565d1e437772236997bbf2fda38ccc84ad6ea3efde43d701602d5ae020e69451eda4356e1be3ee36e74d1c641c3fe11d07481eb5b9d1d9cc44672d9919f6cca6bd7bd3dda6498fe0cd14cd2eb986d9161548d1ff42a4ed5f5950")
	miner20.VrfPublicKey = common.FromHex("0x43ff010d35afe2b74063aa67a8e023bc3b4f7cccf30416eeac82816056c88a7c")
	verifyMiners = append(verifyMiners, miner20)

	addMiners(verifyMiners, stateDB)
}
