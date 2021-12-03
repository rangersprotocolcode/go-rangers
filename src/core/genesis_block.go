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
	"encoding/base64"
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
	miner11.Id = common.FromHex("0xa09540c8add3c5adb84d18dffce615685774f4eaecd8b59fc1bf57d0b6a351bc")
	miner11.PublicKey = common.FromHex("0x6d5701035858aca34f58931add6176dae40ca28cf2cc3599ee6b2bc13a70ccf13e5f988c4ad8ba9539146c97145b37fe8a85bf20ef9a30ac70f090a87d4d7b5139df4cb9667712b45d77d6f797edfcc1a0d4f3c1503fecf6f7b4d6aef8ca4b315aca02a4626fd154272097726cb15a73ea69f570a38452ea5866c6ecb2557369")
	miner11.VrfPublicKey, _ = base64.StdEncoding.DecodeString("GiHfDVN5ULiON57jzuAq8h9qEMcEMSGb0oP1PrFNJ2c=")
	verifyMiners = append(verifyMiners, miner11)

	miner12 := &types.Miner{Type: common.MinerTypeValidator, Stake: common.ValidatorStake}
	miner12.Account = []byte{1, 0, 0, byte(12)}
	miner12.Id = common.FromHex("0x04c07144b4139850e121724169e71076bf598af754912fb582c7a45fc0b290c6")
	miner12.PublicKey = common.FromHex("0x5834c3bc10d9219e5c9104bec0106a1f9e289d66db2c88fce9f735c2c279481b79ce5be0d1b248a8a3a3141bfcf7b5bc7df65a50327d3d8eef0cbe863ab573440c01286312add654a6ea2cb64d73a477ae31c42f1e34a71f46680836a196c0a75094efc99bf4b3f7eb9ac16f7802d2734d64ff144732b5fb62e256ae8680e5d5")
	miner12.VrfPublicKey, _ = base64.StdEncoding.DecodeString("d4mfSXPTgXgbAtPgQzLUcQnIuW6bHWB5Hyr5qVXuLyE=")
	verifyMiners = append(verifyMiners, miner12)

	miner13 := &types.Miner{Type: common.MinerTypeValidator, Stake: common.ValidatorStake}
	miner13.Account = []byte{1, 0, 0, byte(13)}
	miner13.Id = common.FromHex("0xc053fbdeff3cbbed6a8041413ef24b848b80fa32efc5ca842bdda61dbbdad4ef")
	miner13.PublicKey = common.FromHex("0x4ad5f2e89971623de8364bdca07cc967bcc398cd9b7f34d7bba1dc448a3ab4545b93687bee82ccd09bfb8f5dfefa579670d214bf5a91a5d5bd488d17d1fddcfb60af073415c11e8d410610d5af8a0db0c948b177528e3a844797a73d9816e75b23a67545881e475ecbd7ae9dc2509e445527641b2c7029d20d68eb09a434f86a")
	miner13.VrfPublicKey, _ = base64.StdEncoding.DecodeString("XheZRfneY7YsUrPG8/GaAUoLYDXgIkmjNHzQ3VyOSuw=")
	verifyMiners = append(verifyMiners, miner13)

	miner14 := &types.Miner{Type: common.MinerTypeValidator, Stake: common.ValidatorStake}
	miner14.Account = []byte{1, 0, 0, byte(14)}
	miner14.Id = common.FromHex("0xd7cdeaf89176429db91fd6094bf02ae465c2cecc1d751ebbb2ad66cb7c104859")
	miner14.PublicKey = common.FromHex("0x48e806eaf31583bff484537ae9922db32fb16563367a4fc81ab78c9bf525a4ab6e0ac6fa05fb2237c62d882d2de7b77596fcbf8423347244cebf51eb941b8596028b47f26bd6deaafdd5ccdb3b960da4fc499141c8dc41c29e205cc60eadb947362952ab7a15b9e639b99b1257ac43fd99f96e2eb4df7477a1e2e073c2a35a10")
	miner14.VrfPublicKey, _ = base64.StdEncoding.DecodeString("scdN5k3xVKF3R2kv23MTeUgzjwG1x8gFmqwqPEdNHbU=")
	verifyMiners = append(verifyMiners, miner14)

	miner15 := &types.Miner{Type: common.MinerTypeValidator, Stake: common.ValidatorStake}
	miner15.Account = []byte{1, 0, 0, byte(15)}
	miner15.Id = common.FromHex("0x8b95500aeff85eca80201e71fc03018db5383564c2e66f092aa860b10cc972ee")
	miner15.PublicKey = common.FromHex("0x537a0410abea8922dea03715b99e830bf26a5a55a33e9419202d236311a3b5ce1de752e1708431668964f23616096e88a50955075954e98511b1796c61f03a6a3ebdc651ae986aea3b7108171a40e12280ceee9330e1891fb17b255839c41296331f907f489c5657a9abff5dfcb4c147c0260bf384ff3d42d181477a842c8f9c")
	miner15.VrfPublicKey, _ = base64.StdEncoding.DecodeString("Asrqk00htciajm/I6f0J7y8M16rfO2JRAkeE5/83Ckg=")
	verifyMiners = append(verifyMiners, miner15)

	miner16 := &types.Miner{Type: common.MinerTypeValidator, Stake: common.ValidatorStake}
	miner16.Account = []byte{1, 0, 0, byte(16)}
	miner16.Id = common.FromHex("0xeb6e7487aa27f5b91bd9bef67779bc50047a9c6da9bebaa47ef7c0040a660e8e")
	miner16.PublicKey = common.FromHex("0x80e0318f5a0de440c007b1b3fbd4185b8bf6b8bb3351b78234c6bc6d859879c21aa12bb276bdfce9e3317be03c3b39af9091591598b7ea98f61468bb7021ff97423e37d1970da69e519a2f302de78403f9e39f572fc3b026aa44c1456c2f9ee463092c3726f066c25af8dadb708c8067a26477ab88bdc96a8c77d8c278f45f38")
	miner16.VrfPublicKey, _ = base64.StdEncoding.DecodeString("3b0btDxmzj6PIzusidyCMtWaLWASCGcZlq3+Qlh/1UM=")
	verifyMiners = append(verifyMiners, miner16)

	miner17 := &types.Miner{Type: common.MinerTypeValidator, Stake: common.ValidatorStake}
	miner17.Account = []byte{1, 0, 0, byte(17)}
	miner17.Id = common.FromHex("0x72cb53924aebe5ee960a71d40c36ac9330039b402d48e72863d479b0c7612528")
	miner17.PublicKey = common.FromHex("0x52632ecfda8338b66fd7a52586259da4dec6ae70fcc3e71907cc526409a84f284548d29daeb4103a956c89cfa07d48741b1d618ce729507dab356fd54855a2c942ac82df2f3c648ca89905e8504fee470b695c651a25d2c0ea812c812a511ff43f66167c332892af284294fe3c8d97ba669ef0906593eeb547e2cee291951ff1")
	miner17.VrfPublicKey, _ = base64.StdEncoding.DecodeString("KABoueNXGt8ZQAcKtc0IQgMl8Nfudf/oCGkYJHfObuM=")
	verifyMiners = append(verifyMiners, miner17)

	miner18 := &types.Miner{Type: common.MinerTypeValidator, Stake: common.ValidatorStake}
	miner18.Account = []byte{1, 0, 0, byte(18)}
	miner18.Id = common.FromHex("0x24eb2635f561fe55fc9ebf81db91ea47f8cd2c9f59a8c17184ded1c81a155de1")
	miner18.PublicKey = common.FromHex("0x49d850669d21a689e3bf1e84d23c15e848b11858ed816b4779dd8bc11ef0eb03259fc850c40b2130b6a52fbeb6903e5b25f2d90904bfe252780acfe88dab7bea7a9c40a1f37fdb9371c9e088e5b42a117429c8203828073235a3820b01d9dd845c03d6345d810f3e18212bbcf7f4360c3f78079d3e5c4ce835b8ccf251c65012")
	miner18.VrfPublicKey, _ = base64.StdEncoding.DecodeString("2y+l3X+f8cagCaa2nnpw4N+Qfe9xFX57W11r1NhEvyU=")
	verifyMiners = append(verifyMiners, miner18)

	miner19 := &types.Miner{Type: common.MinerTypeValidator, Stake: common.ValidatorStake}
	miner19.Account = []byte{1, 0, 0, byte(19)}
	miner19.Id = common.FromHex("0xb3b269d5dceb5b70d6a6fce792a81b654b2ad57119b18d7de6d09ea41d92adbc")
	miner19.PublicKey = common.FromHex("0x6663fe4bed8b302486b9647d413f1505236e069ad90f22dfedfaaf455124df0c7db59e5af6817144f1169500c469e6dbeb3ef451af044b25cbe3b18e0d7c88e07dffc0887cee7a8c12eefbad5ce7e2ff4b6c5be921e5c8358e7495cc15d8bd7a85790d768c08152647f9b7d9c6ac23336ca600291b3b3df06e5f72183d45eb5e")
	miner19.VrfPublicKey, _ = base64.StdEncoding.DecodeString("BUL2zJSUPA9cOwPEUTQmbUukmD9dLwU+d/OP7rgEcMg=")
	verifyMiners = append(verifyMiners, miner19)

	miner20 := &types.Miner{Type: common.MinerTypeValidator, Stake: common.ValidatorStake}
	miner20.Account = []byte{1, 0, 0, byte(20)}
	miner20.Id = common.FromHex("0x63671fe9bdac7aedd53fd227cec98c1fd86ac2104a6d5f58c8b8d6896f9a380c")
	miner20.PublicKey = common.FromHex("0x0e7e4c98d5b7e72ab053bcfe0f6e2dac407517ab9869c4ba24d9b879a477b3cf492e32fe3a753c220f3d0b67251f81cb81dac759ac3f972a84c8a36d3f887140011ac03757d30d8b87ad0130b26f4e428d72c0b352c87983860aed293635681a45678c4931f7d9394abd6806ad8a2f63cfc24255ca4f77d1cf71a333bae49b67")
	miner20.VrfPublicKey, _ = base64.StdEncoding.DecodeString("toFM70wLttsxxVYSfDzC60GmFfZyAifUG1wUPzd8oX0=")
	verifyMiners = append(verifyMiners, miner20)

	addMiners(verifyMiners, stateDB)
}
