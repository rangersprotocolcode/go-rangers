package core

import (
	"encoding/json"
	"math/big"
	"time"
	"x/src/common"
	"x/src/consensus/groupsig"
	"x/src/consensus/vrf"
	"x/src/middleware/types"
	"x/src/service"
	"x/src/storage/account"
	"x/src/storage/trie"
)

const ChainDataVersion = 2

var EmptyHash = common.Hash{}

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
	service.NFTManagerInstance.PublishNFTSet(service.NFTManagerInstance.GenerateNFTSet("tuntunhz", "tuntun", "t", "hz", "hz", 0, "10000"), stateDB)
	stateDB.SetFT(common.HexToAddress("0x69564f3eccc4aedabde33bd5cb350b9829deced1"), "official-ETH.ETH", big.NewInt(10000000000))
	stateDB.SetFT(common.HexToAddress("0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54"), "official-ETH.ETH", big.NewInt(10000000000))

	stateDB.SetBalance(common.HexToAddress("0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443"), big.NewInt(100000000000000000))
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
		MinerManagerImpl.addMiner(miner, accountdb)
	}
}

func addTestMiners(accountdb *account.AccountDB) {
	minerInfoList := make([]TestMiner, 0)
	err := json.Unmarshal([]byte(testMinerInfo), &minerInfoList)
	if err != nil {
		panic("json unmarshal error:" + err.Error())
	}

	miners := make([]*types.Miner, 0)
	for i := 0; i < len(minerInfoList); i++ {
		if i < 4 {
			continue
		}

		info := minerInfoList[i]

		var pubkey groupsig.Pubkey
		pubkey.SetHexString(info.BPk)
		pubkeyByte := pubkey.Serialize()
		vrfPubkey := vrf.Hex2VRFPublicKey(info.VrfPubkey)
		id := info.ID

		if 4 <= i && i < 10 {
			proposer := &types.Miner{Id: id, PublicKey: pubkeyByte, VrfPublicKey: vrfPubkey, Type: common.MinerTypeProposer, Stake: common.ProposerStake * 2}
			miners = append(miners, proposer)
			continue
		}
		proposer := &types.Miner{Id: id, PublicKey: pubkeyByte, VrfPublicKey: vrfPubkey, Type: common.MinerTypeValidator, Stake: common.ValidatorStake * 4}
		miners = append(miners, proposer)
	}
	addMiners(miners, accountdb)
}

var testMinerInfo = `[{"BPk":"0x1f93119d345cbda059cc3b984f8f8e0be98cd0b4b5479e38beac5866c11cd6112c5a95f9f8fc54f5d003127cbd61e137320846e14aa6dccc4e681982bef08e17094e1b30137b8c4124ebf1c088bbd897104065e664bda1ff015b0881759adf7a1d3e4f439dae249c33f4a9684f5d04f9c0b05236c5ebafbc23ca891cea2cad43","ID":"ZCDkZ8d1FOCUcafYTgVSwTtelxkvUjwF05cNfuI79EM=","VrfPubkey":"0xa8aa3970cc71877853f88dd9d54337e38d3b36e336c90148696fb8cb0876fcbb"},{"BPk":"0x04fd9d1e30a8627a1d27fbc62be906cf0a3d78ddad98e71096cf1fe06379a1231958e8757cf630348b9989f0612ceffd5e41b962abaacaf3e9cbd5295c1d27f911da7912cebb18232fd743a6a273d22fd15a87667692cff53a89c07c78e1b377104539171bf7b76cc7222ef0ff4f75b9936c449ae29401abe6614711fae894b9","ID":"n2fo53hfSJoeVKj/jnynhZ/PxcLO8ieNW7ZSigxcYJ4=","VrfPubkey":"0x7edf7b3ca891d98dbc42d955bda45f84478b7ae5c6c6e2be6c865ef29bb699b8"},{"BPk":"0x01d3c0708e58d77b94ab6fa8af991a854e57fd3e491d6cf17b47f9a2b052e3e5027dc9e56aca4b232dbc6a860d34a17f16cd5b1edcdadfb33e54077a8c27fc6d265a3ea7a039db31ba8db0165987d1fb8fbf5e161c2bd67abee3a879dca35aac29cbaf949c6c57ac7fe315038e35976a08416e020a671afcc0e2c910a8662dbe","ID":"RFFzqzloFJH2iOi1sR8/UQQc4NBbXd11zMhvTDNDpBg=","VrfPubkey":"0x26b3b538813e6249a6ed4df7ff6311437b4573d347ba05ba92be2bfd972c5d4c"},{"BPk":"0x2209aac94b5f1eee89abd8260fb41b01a5dcdc75b94f81d0cb62227dcab165cb0acb61932491a896558b7a2120471c8d4dbd4219963b55ae215d20411011ee91245dde1f76e98937b2602951998b68e3baa2c1cb666c6893c5dfe4e62685f7fa0fad2195d91edd0a88ae2cf73839c4536dd293a4733832f826687b821815cfec","ID":"pei57J2kD577yhD7UlXKbx0B0mJDJS1oqME0FsYPCjc=","VrfPubkey":"0x832cb19963192fbf5d4f0f3c308247c06abd7ab12baea734527c73f81ee10f9d"},{"BPk":"0x2810cf8caaa0dbfbcc4ffb624b93cecbb66619adcc98283aa1c4ab75ee3b68da08816755c6b70de62661f60d67766427942e1cf7307d51a0f3af7aa9a0faab91180a67932bc0e3e9146167a6c69628e1d21c91e22d020855c0368f935363f8fb207b2edcfcfb4bec8dead2f8a599f6c97b09d75f78425485c491b68132c27715","ID":"mlrcS4PtQnL4rwxGaGqThwE5GuNXa3eJHiq050OPRC4=","VrfPubkey":"0x0f0ef3349784e308fe7622b673f3fef6b68be91ef6498d7249b95e6158a126d5"},{"BPk":"0x09467c909bc34eaa1df2521032d8a2c7e5d66c56a54a1d0197f16bc6700974ea2ba5c664e2109d6946a2ae2bbf7f02465e8a6f703a9af3ddb411e8ca3779d9330563e5d432c9de1c07892285e0d4dfc1a7acdf13f8e410e0bfee77cf817ae6b12e3a19eb0ab08f3ca11155fd0f5283694188655a69a5923644f713fd500a39c5","ID":"RLVvlGbu87dk3KhJMgnrCXDq7mRqmC+qvk8QEDEGO8o=","VrfPubkey":"0x911280f477752fb7febcb89bec871322febd241ab204d8857a59a0967a8c154b"},{"BPk":"0x0cf2f010890a5a9653a1d2d42d4d5c74c13bf86f2d7d4561cc024d29b9a4c3ee081756ac94e910d0ce1c47146ee09e465d20da977f888768df659ecef09fcfa7196e7db9ede2d7c57a065f40f284b635e7567d6307caa2d2955d99a41361cf45042a152be20f4539190abc515e2cf0d80041110998b941dd3cc5019ee3ed9ebd","ID":"NfIko97/PYlDRa09O/4IM6d2zKH21F1DRmzc4S6YDps=","VrfPubkey":"0x35409069761d6e943cd5376ce7517da0880fc074f119b5383388021b6b00b235"},{"BPk":"0x0675bdfe959ab354975b2be00b43d91dc8c0f45a8e367825688232260cea4d5301f84aac83ea4dad786f38ff17e232975229df64cebcb523ee59534d10e05549109612c9b6019c933c9624cc7b395cf9d7ed98a212aef07c87f5e2642d59d7381d61597deb92fb8b62b38ba210c776c1976a61afa8a4ffdaddaadc3d6e9b6d6b","ID":"a6KJdvc1IVmiz9EuqoHZqvkcsWQOwfe5jEdB3DsNgu8=","VrfPubkey":"0xc3518bb73351527af9875506e12c275b8d9a9b4a8f53451161417b40e091d0d0"},{"BPk":"0x14a8d7d104e6bcfee0361750a042bfe6c18ab6a0232b22f8db989b065ee872fd1eadd42a2118d5c0c48c09c7b9f3d33756eb6f53ed0c90ae34c9d766bd41529704a3026bdcb80b76540ba90ef2e1f48cdc74b40cf8757e3b16db7b7719b9ce5f254d4b4623c39d4f382c87eb0c79de62ca84e8c04ebdac96d38799a673c7bcd7","ID":"VGdZk2Rr/pctd9vZ1UhJ02FJd8GOUcnl0ebhpYB9pG8=","VrfPubkey":"0x4d0a00e82228706363fe1a2e7daa5af077a2f7ee7d830dec3d003bdec328e866"},{"BPk":"0x1d3229558b2618323d3415564cfcd51453a97c55e18d0351cac7fa9dbb7029530dcf38db4ebdcb067515c209879618144d1963629ad7ac067ccbeb90176ac1321703d301ee142ada78cdbcc7fae79a4e1b4862c28fc5c19e512b8f64591d1ac72885b87c45ef577645e9e6aee003da8bf007c2734461eee19723627ab69864ba","ID":"IQEVBzF7VkvWhz3RbMieV0ILpXUtkWX5CIDdFvomDBg=","VrfPubkey":"0x306df4b621733693d80bb4df0224e08f31057ac97cf62f45267f17e67d67fcd4"},{"BPk":"0x18288c1032e3d1599de6d0b031d7d643d20b73d72ab1a0b9a89fbd118374caa2266becd776a7e9bd9271ccf7bc494db8a2e432fbe22fdc2adcc2322cbbbe88be3035a230582db7f6bee156664a452a467d3f44b22a8306416dec6a7657a8fe8712bc34944f2bb352cf7aa7905487a4602ad4df16a37241fbde4c89e6cc1c9677","ID":"CcX5jHIz1xxj2Wd2DWvVFyM8hP8kmKYUHhnkff541tE=","VrfPubkey":"0x0d79423e39b149f91a3102eee16f0c7fe6603976403cadc922851f51d5b387ee"},{"BPk":"0x119145b56eb489c34221a2b8a422bfe700330ab8b28ae4abb523bbd4c9cba1052a599a0964a37c1c36a5436658400deb6dc57a7b71171af44457c132379edd362e2c932d534107c0ce6be7e92178f458552c1604db67264d9883edc97e3ad1c61ba80bdbfa2459be8fb2edae4941e30bab12fb84189f9047eb416edd39585bde","ID":"33htTMfZCQGA/wp/EOhwFq2LnNvEXVVttyvsXc80GHQ=","VrfPubkey":"0x061538f3624b302243bad8cdd8f1ae415dcdcf83c7eac7b5c637db93359e9160"},{"BPk":"0x21c7e09d18e489e4e5b8fb982cde5d1d7b907dd3ed2fa2672d61192b58c893310db28a14ba97392a641eb09dd0b175cef5879c26196d41cc42b050d683cab70813ed3bfa74cbfcb8dd6ba5e3819427bd520a0e88e8893702853a295a3093a33f058ea2e752aa9223a86c952898ca10dfb3aee75888a88270ce0799b87d8a0a4c","ID":"JaAERJEEGgP9f36D1zgubg4vkJk6dOKedpibwxqW2lk=","VrfPubkey":"0xa5c58f560cc3db35eb92ddc881cf7454932af586d84fb5cf352ad194c251eb1e"},{"BPk":"0x2f0d672c5a412d1ca8d6296c631aa39a17c78b66a7c0b01359c92724cab667c824647f27ad0685b045f61206e5fe86e58ecb3690c48d03a0d7c36ca71ffe1feb24214cbcff0a1d6e18f7a0aea40472ac5d73fb2249681753ba76d054991b65841c9c2b2090c4fcf41548ab4f248d5048f8b0fb28570d0ca557245f992991327b","ID":"4XNip7NRl9tjmaBazyGmk1JZ8duaJ0werSW4iVNPhzU=","VrfPubkey":"0x9b39b7d072b5b1b3605e669cbfd97a88b92819560de888709143499fb30aa1f7"},{"BPk":"0x04f47ed704d950ac54f82b74994bf5c963a52853682ea5388f9efe57db8aace90261a95abd8bd07e34511deb4719c6069b890b7f62bc492fd78a45ee9b8ad3892fd94317bee56bd002ec29ca7bbbf0be89b7bc6503f55f95573ff46eb240a2ec2422abf42756ddb06aebf03fbf11ca29c7f8a080b8ac148440ce5fbb106244dd","ID":"ac5bCqKUA/QPSO1z0672PzNH1BuSiGrKtF/O1MU8NEA=","VrfPubkey":"0xa431c039d50bd513797a70fd76ccedbbee8724881dc2bf56d5eabe4b903ff5ad"},{"BPk":"0x2f6021694b6f2ce6556949243a39b55c6046b9c0605c52dba6769bff51aecf5d2d4d549f77d92282d3428c2647f60d5d196a4bdd23f0b5fa15dd7c229f893ece01d80baf85508525a5c7eb998de94771c455aaade9a18192a8b36110359af9c02b4b9c843e271097c18f7fcc8da9c0a1cdfe235c57cbbf906e67fa836e4888cb","ID":"JU2gOy/gO46oxvSwVJ3SHtltiAineE2BqBkf9yAx9os=","VrfPubkey":"0x3be78a8e96c9953061a042bda84ac7e464b34d1fbbe3830d4e9e126e5c90905b"},{"BPk":"0x01b58dfde9c0c8a9d9b829a845178d232c4e79c2707a3891fab4c0729c9f5960071bda64ceab3de8a1a82456076365b5867bc87977172d287d62c424f55d8a11132b3109ac042f319c4166cb02d2ac8315b5c3ce3e131374c341c5f0d12898d9117f36bb7ff1f936e3f2e88032e8636aec2d4965d71f3b9a6ace053a1e245f2d","ID":"4WBhkSaXgYx/1Dr5FhnDLmdDlOmucaiqnac36GEX20I=","VrfPubkey":"0x8be835e6e156588ee6c1cba72b68c4d1c0a36c73a7dacf8ea52f29fb2012bf76"},{"BPk":"0x1d12dd342f7fe5daac049cead7be6dd136febb746e4156a9a31477f2f822f1f120a4d3a50a1ad325f832dc9331c0a1634dc4ef3733f99479eb6128668636946d2f4a2300cb527c441b8bbfc5f0fcf2494ff261d2c2ac529aed409a916ed99192216902a2575150f7e486c6e4c52aefa0fd7211dc33b7aeb2da7bde23acade5ed","ID":"9WOWU2ie7oXZ0Wub9mnSVfnZCj5tey2VP+LqOWRBPDk=","VrfPubkey":"0x07671d1375e5c301d1cef1439d12684486ef22bf17c3054f2797ecf29d059541"},{"BPk":"0x074ec04e59ccd10b0a9460acb396116b1145f6899a41dba50482afa64cdfa34125a2c22336070000ef19467c667a250897696d45cd6e6bebfa4f3900a17cbfbf2cb02593ce24a1fb47fd0f0198018fd8feae55e71b14313510830ac8c2a06c8308734cd5c907e417800f8868e1b5d1bf612b29d3dc6ce40b0bbeee2f9a9fac32","ID":"x2Zg0hdb/SFyrHP6Xpo3PtDzhqtCDTUF7Mq2ha37KjY=","VrfPubkey":"0x4a705a6ce31e6997f97275805b85595892a9cc58699ce413128753328d9bcec8"},{"BPk":"0x1c92d08ba7c20b2cab77bead324b9aa475d45f5931b7fe3f3bd3ed21f67ec4af29d0292eb1ac9eb74196c91b4ddb24392b02532967c90f8cf259be91505a44352266f77a0ff3ac6cdf7eedbba5ca95741a39ff648429f7fde60af4c5327b6c460d6bb23e0be73012e8a28119407173dc08bd7d5271ae08089233f79b687e3de5","ID":"XslO9Zv/yTBVgE0A1gpPk7fS6Q4fc3ZgXB+XLeudxB8=","VrfPubkey":"0xc8ae3c5c4b53ee055e02dc47353c4fcbe864e5da1059eb65799521ff317c41b3"},{"BPk":"0x022d678de8c683b5b682ee044c192ec20418ced937e3f4e76b9f4bf7a62d6eda205cae2de7f14f9968b1502612dfd183a0f9e2ff35e740f103a827bdd41692c501a48409066dc5c4a5c51b20da1e643519d82c5636b56af8d6b3657fa27e934e0939f26a2c5420ca9d560e0f055057b057ee1c3d441ffb7977885fd665dc7518","ID":"4AuBpQMTUZrujpjvwsI0LVY0G7C2vqop7+1YCZtkGcQ=","VrfPubkey":"0x08218643264c20fff892ebd763ddfc7e3949f60f853a39c8c55a50be69976436"},{"BPk":"0x2357b66ca2f5b41fa15177bb3fe817f577e843119c4d7070769d01e66f5c11701559ef5ac097f80c08c21e8305dfcfe82c2a0a6f32cd4dd04e314fe398feaa762b960ae4e01b6800d1da0c05943529d1dcb545a4042fe6814e8caf301b5ae0942f28864c773ea5e845e5fa71f461826094f4c941d907598f7c202b6d7b288f30","ID":"fx+NX2UTkqb2jEWrEaMRu1UYtesAnWVlUJyPeu2z1Do=","VrfPubkey":"0x73994e6d03c321e5be11433989a60dcc7abab3d6e8219fe54ddc5840f2be8c0d"},{"BPk":"0x09e6fd673039d788302661c8ae2ca0158ed976814d2fcc8303b6a600934b04ba0cb5f111176124f7ddf5a8b61cfe32c83575cd00120e5c44760d11cd4e84a8791560a81f397b1a167f0ce49e0835b350f0f19b786342799bda7843e103799e292abc6af9626e74d35337355f15bd6ca7ab1e1dd7e3bd9765418ba750a825108f","ID":"BXS8ZLwS5gD2vJ+0QHUhzZm7MIbO0924OONMpzUFue8=","VrfPubkey":"0x57531ae9a5c2b2b77612cf6523f1e17c414baebeace33012ac043223964b8b13"},{"BPk":"0x2eb5b8dc711b5040b677f9f9b276730073868fadd2339bf58653bd67587c1a9c1692bf09da4dd80a877cb4d298929967688ee4a6eaaa51cd3574b7448da30b922e22b23acc2e093ec4f90a7d83428d62be1d5d8dc8275e3a99acb7d5dc83d0cc01355151077cddc8e0f507cd70fea471bd16e5f80542315b062e0c54e55487c3","ID":"p4YyApRKmpIFjRBi8Kry9jOaZ/JrxmPomia0po9B7gs=","VrfPubkey":"0xb97cc4df1236639f8067e50590c950a7ef53ed0b28b44df6e81f65994135b456"},{"BPk":"0x1465015069016039edbc09aff2d586a5c28cb1388bb508d93ab029ff1fcb09f213a768921e545163dd07842feb64b222800fdf4a31f50a88c1d58ff1fc4ed461238e707bc3b8703fb65621fd39f3f9e38d86e95103826c8c7249016ded40529111a0eafe46c2ecea388d29ad3681b31edae8148c3bd75aea5efa8b71df0c3f5b","ID":"jAMMwfpVpRQjMdcsPp0vybBQujopPdbCevN4PDeYJd4=","VrfPubkey":"0x2dc698f9fb9a880ea484e487347ffdbe5b3d52b65895da7586ae3807ef92440a"},{"BPk":"0x0e08578d1c531bc54b0a6a4b9b8a7e3ec4f22378cb3a37f8644122a9c005e53e17c70cc649a70411550cd18a088ee6fda73542b17827376588fe1a5ee3db314612b6967d998d001cfbfe5000f1277495862270fb91aff4ab440b03c03fa66e4f2c7693db5aa5c62f43bdfcc2d87b378268aff4b0b25c53536a772b38e5df3124","ID":"z5MjVAgckgCZSuuH0IF4c40lOKDlKC9Mh/BObr92hRk=","VrfPubkey":"0xf26896dddc5a56919800f516732db39aec5b50b6260be18c851ded8fd5f98ae8"},{"BPk":"0x0fa31be4292d888ca8612f5c03a98f6322c4b59fd0c33e6ac545943a88578640241094d3dc51a2223ded799dedfc3b44726feecec3d689cd3a0e633eab51dbea2a5f362c905f38087ab720bd28eeeb6956a7e38eb1c74f07d5aef9c11b7e5d7b1fc391d1ae1570206311a5bedec3d6650b510fa94f72a00705bcd5a034b28c59","ID":"0hslrqp+o8aAgJPw1hHvo60Og7QRS9Jle/7SoWxB7Wo=","VrfPubkey":"0x12d7ff96db66ffecc0efdf287b98fd01b0505f772c0222c92e9fe17b51d9c72f"},{"BPk":"0x20d73db6b650f483548edbf37b949d01941fb9a4e7594c5b8f773da3e831f52d2111a4f736f381344ba66950f67f0b5bdef3fccc5a83b3c1cc724461692b8a232b55ae1057dd039904c191da8166bcef5d01d1f3e0ccf6ad4ae50b78ab7d9e941dd82cfbb8760709af71780396cbae4185f3c4e9d8ee99134bd2dd7f1535eda9","ID":"ObYnkqkGhK9IZgUnazPYWUfE7Y5xdoascYSIQH2zNHc=","VrfPubkey":"0xa317c860e08c17eefbe7742a6a5feb0e30beebdb877b8fd1049bd290778c9f1d"},{"BPk":"0x1d405c7c0bed520ecc2a672ae84d48e1d81951c1e67e2c26ab14f2efa231b27e0423f962459736024a745f522cf129011b762090a5781339220888d4042d870c095ab8766c68c77011770ab480d348bd91d9d5fff1e47a6299c51f13b683cc1a140a70b69891a4ec661480a09404f1c719c90781d88fb80cbc6e170b73ee51d9","ID":"UTYS0GBbNLImNTEbWmEMANscfKlypw/Qhgs0gkuyDNU=","VrfPubkey":"0x63d0ad20904445833aa910035325f61f0076fe6b359f6fc3ef1fafa756529d78"},{"BPk":"0x17f76bc3896b27c4ab4151857b3f1bbbe997c8d7c52793204ace7de2fc2a58e716fc4479472fd3d68625dc7480108aa37e789d1b77d9ffc974aa31e63897216a133173cd38f816fa720552fe93fda22dc03ce91e5659fcdf222b99e8b4b17bb11bfc9b8050e4aadd3f37b23f1ab7f89dafb797aa7c32de4bedfddc1c3f553d17","ID":"Z4xsg7fef1R/xLwWCGgHITGv1jgYzSzxHpNRxqKAVYk=","VrfPubkey":"0x5951af84899c717fddf59bad54739d050f90d4141a11b7aa097247ca1c0da110"},{"BPk":"0x16ac5541d1be1b47ce5800ad5782bdec2b7830d69d8f9484512fbb618f306e8d2510bc8d35e38cf4ddacd25f8edab75e50e47db34c5570edad2bb3c7d09ac64b2ea5522e3725885fbccb44672cc9b70dfd015610880e16c3d20c8d18ba83e3d80d24d33c66de627bf817065b0f5c93d24ae147136e197080c23badb101cc9819","ID":"9wZgxi3J/ZE+qOQb74rUopqBADjvJf1m7X2BhPgW/+s=","VrfPubkey":"0xe0a1d0b45a08abe2679e28867e34f6e55e0328529b4e35d4a8532924cd1bc3e5"},{"BPk":"0x1fad82f7722c5ceb223ce74e7c0194c0f984c5841f231467d56713bd0ad414d52dc238499f7c7ad5aae6720f52cc91dbc1d7643ac4b17ef6726a11037cd6cabd03c0e76476028224097a0f6bf0d5d67b3f891e720659913d70a04671830fe2fd106f31c724b1ccf3822898427a02dabb0f016da65aa5f97c13091543646f1ef5","ID":"EdeErihZsDPBipjc/miPpaZoGqDDlf2It+y6thFnvaM=","VrfPubkey":"0x0d012999165d607c3a76011e0a3c5ede55dd5a31069badbe4d730550b41cc81a"},{"BPk":"0x168c82f5bc6bc0bcf0066c66f892a8d6aa7e68b0b621745e3d7b07e8c5113c7716cbad42f1a644534a173fd311adaf16353bebdeb899f77a384123801d8fed1613acd07480be5a7c32ff2dbbcac91903b0d79cc9530487569f24ff100e85be212fcddf0018cd5538c25c77f9394381273d7d21e2450031cd33c16b772988c594","ID":"/A5gqL+kwfcMTxZT2HdZqEifqTnXiSm2nWn/iqMAzjQ=","VrfPubkey":"0xe910a1a20bb849a801f9de843429d50aacec25fcae77328d756857a404705b5e"},{"BPk":"0x1e6595422db9e0247cde97bbc4d1b6f186e06b86dc4b9a8788c8dd366114bd8327086d2f58dc2461e8f2eb8e11547de7fa9130a68a447c1827199be9658459d306544bcb6312370630941d3e6661532b8fabc9ad51e8551cfa729b5a5b875a38078a802eee1a8cd63ca9115ac1ac6ab0ba7c62644ba9bd9aecfe8a1972295a78","ID":"uE9Rfgtq1E1PtmCcPAsaxihBYk+fq5YBco4p87Bj5kY=","VrfPubkey":"0x6de18ab8d7d3e616443678f1ec3900b96d47316eb04481105e17d7dcb8cc4fff"},{"BPk":"0x1129b4da752cdd70482813d725a8c9c0bbaa2320de521d1a5a1e60142172c49d0cd06c27bb86be5d95f71e668ec923fdce47ae2d64ea4df9a319269a0c45c12b2ef02e2a6642f47910def8df1401ae0f1599919e38e0cbf57d545733804ad26e15d7a28c292851553c8c1f8b55282ac8787eeb51c965e6e4956fb470ae36e252","ID":"mI4+yfXuXViOeHIRhmNoQqouBFaDmfqP5khHZOpUXHA=","VrfPubkey":"0x237acc6c1390dde5feb17111748a1c81233d3065ba391157a86406788c60e09c"},{"BPk":"0x058fad573f1e8f4f09c1f7408575fca656a9aeac761ad7b2dca866662a9591b10f03b54cae6c70fd98b9eae0c58c618847774d945d5e336b35f249025893a9590a7a0965940b040c6807f0904e827080cd3bd983f8002d1e4d10324618bc0fc000b03e99ba9c400b254e184920d9b30567898e53d83b8d66c1ebd8bb745a9c6d","ID":"TL9lHXAQenihRtCBKfYFQTs9ugvVQcEbWrkU8y5sAcA=","VrfPubkey":"0x8c388ad68c6f1a6b90f9538dcfd613c56581c5d8b955b8c4ad1384366d3777e0"},{"BPk":"0x294b1d4c9bf605aaf8b781e6eee29b363a3c02acef64efeacc0394855af60e4815eef8877ab48e9442ac69e250017f15e5c2be120aa150f07de73a9929a61314273fdc5c3b88f36b131dd1ef73d51243fac2a48cae1d47dcaa9e31cbcdd45e19081407fef02f79569f8410e3ae78c8e21853019b0befb5e6dcbd9623e9f78eed","ID":"i0dllVgRzvSSB+XB1O3WyDu1P+tVbzQu+VpVKgRJfiY=","VrfPubkey":"0xbf182f4436dddfabb7d8fe257200f80aed87ed5511510586be796d44f9f5e070"},{"BPk":"0x1e5b83f8936422c1f11229e4659fd59f740726e86ba82cf39f6278a85052b7ce14bb5cdca9a4a0d2f8852a17d70cfc0f4dd40427e5431a075d977ea15cda072a23940b110da5ead6f72ed197a47f22b777848a6b629cb0afce6451888a52544906d2d0bdfc7a1de2999e8363f3f2abcd36605cd73e0177008093d56d8901ff8d","ID":"yyh+rcRo3Ja0kzntB1gGvwxHo/diMvWrwPyi8Cd25ko=","VrfPubkey":"0xa3d568d76c4266c75088fd0ac77986cfc563ad45fc3001e7ce1d4ab11b524912"},{"BPk":"0x0c368e7407c9044e7549fcf27367bcf0f4ee726ae8ac5947f79ab7476ae2da0305187cdcee740908822975a78e0ec154e1a781d7c9888f5fec9ba3c5171f177f29fe7e7ea3eaa7ffe171ec65788164780dcd7e15e3ee1b8878335aec36c2f65d1aaadf65392ca7827c175c31f9c4138ac040b03f08f27a1601dbafb282b5df57","ID":"R4gCL+5puL8ofAtpuQ1Adz+xo6QlH68/XAGBzcP7eKs=","VrfPubkey":"0xda3b6a6a2f617e17013731ebc626143a2b221dd31328cf57af9c882c343dbafb"},{"BPk":"0x0f16afae572c5d78d6d9a2898ed94b915eac040f734ba8f2c1ebef77a562419e15f9f23d3caa876121e3e027e26a78024f50556e3491278f966e98d953e7d5f8214a5dcb72c0c2347a5a8388524ec8de99aeda79ff9b187d483effb834114a531b933de3e00f362d3a379334e0f6a53ea0da12cd51c3aedb24dd5f12ad1de754","ID":"uQGKQv49xF2rgJpn8hBh8ZI2yPtdeZUxiYng2eCOlEE=","VrfPubkey":"0x564feb6d6879a8ab7e7d219a40ba6cc61d92295606d6699f11e7fadd5872cf9d"}]`

type TestMiner struct {
	BPk       string
	VrfPubkey string
	ID        []byte
}
