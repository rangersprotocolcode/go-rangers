package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"com.tuntun.rocket/node/src/consensus/vrf"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/storage/trie"
	"com.tuntun.rocket/node/src/utility"
	"encoding/json"
	"math/big"
	"time"
)

var devProposerInfo = [20]string{
	// tuntunhz
	`{"id":"0x5fdcc98ef4dced468dd26c766dccad27737bace6edd770dab9311360207cc9b5","publicKey":"0x682d47c3cd2b5a47785a80332e7fbb538e47f971641cda734106383f0cec167b3a97b5c6b471ce19679b3aebcf779e8d9fb5724f8a23707bd22ee92f2df9886b8bf97d59e395cafb7b79d7deacabc2c984b34d3b3c23cb9bd08a088fa3d20ab83923d63337012cd19ce13b0aa38d604e10afd017b8c6a6e5ec3d58baa0cf2062","vrfPublicKey":"0xe4265029b9c682150f8d0c9c6f4898e5fe875a8bba67ac27329ff6cd279c8e52","account":"0xa9e11ce87c646ca4b0c8eb66f28a86232734d74a"}`,
	`{"id":"0x5f89899999d5f8b2ec456502b0468bb259c8e5b953e26eee8dfec079666200d1","publicKey":"0x2d1c451547171c6ebd9ee3b9554ce0dd272e9a0a8ddb121de2db1b1f6d6154ff4c98cf52d2d2a5cd7562fac7e52dc38245713d762ff591c9ec94d50f61b426785baa5b0a4a0571488cde8fee8e3e020e966853d2ee95b377b7b5f33eb2e88019100083f5165908d06861b0d45d308a989ace09c60ed83894e898c65ccf71d3b2","vrfPublicKey":"0xdc38cd56a43d046283c940d6da802bdc80e177fd88ee6207bc35d2acc08ab89c","account":"0xb059937233c6a66b99bc698dcec7b3bbe897f031"}`,
	`{"id":"0x035bd043510b1c6a9f268679c9f99b9cffbc41364cabf4d4d1d5be6405a2394d","publicKey":"0x8ddf604c81d24e7c4089a6f4213e1e316428a9c97ede58dada2c3ca5bb0b9b1b0dc8f58f0db0d6a634086dcde7bd24ca7718cd281ecd2bd5ad1eda25c649a042108d7ff8091af896439256b5b603312da5bf907b5e58feb9f8e0b319f8b26ed574ffa1af30b361fbdcc9615d238c638f670065968a0d806c70eb1c132ef8a955","vrfPublicKey":"0x06b4085d0fdad7a9cbe1e6e4ac7c89bacb0e771fd22adb745520404e2547b4a0","account":"0x616d48ef7ac53c919acf78a2cfe59df689b4ba04"}`,
	// mixmarvel
	`{"id":"0x2b01929013cdebed34a6aa99abb8ea5f6676585198882894e99e4df448e75de2","publicKey":"0x29031d130391efeb70eeb00f9ff63be56cd69dababf5278e19e31cefee1577490a0cf0c5ca52c983803c55a64ecf346c688f7caae34d8fde66814beb643ef5c31378a9600a3aa349d6cfc639c132dea52abe546023b74fe252bb0608fa9fadb90a0a74c4a1940dd57de7e9b0dad1a0f5aa2e2a571831c732038a721fa2527f0f","vrfPublicKey":"0x329d3d43fe09b7898118144e7c7c521f898c4b5e31d5c97682a0e324aa477fa0","account":"0xf2b86a27c3cf595f1dfc77adfb9588e32b244186"}`,
	`{"id":"0xd78a2def374f66d0a5ddc5a26faed78510593a513ce2377f5e7d53c38d754b9e","publicKey":"0x7ff57a42d59fc7cfaf72e5546c108ae9832ae67af24d3e6dd2c6d9d1e6c2dc6253b4938c42eaf6dccc35715782cdae6825de7a743ce623d6f37e37b2f4f0bc4e2b0033e86e975f7ed31aa1348449c6a2580a18297ddb61da458071ddbf3c5fdb778f78a8a4b4c38359f6576f9cb36a89a92831fae30d34fd52317ea8d9c9461c","vrfPublicKey":"0x76505334bf29cb25b3d71ad6ab2b7eaa7cc4873d5886b1d9f3162c101e57a3a9","account":"0xfd20013b456865246cd8f36ef43be0030e673ddd"}`,
	`{"id":"0x5b8fa4914cf8eca1714647e623e2fede83e09c3f8c3742b45b8f51fd485ce22e","publicKey":"0x50c7fdbdf84840cbb35b469ae8e7a1b9dd0ec62526fb2c234cd8fb706f9ec48b6dc04ee422ffd15707dd3e9b229bcfab9f7cb58aff73fd25d7c06e03a40dd19b75b1446ee4d946e25a912b779b6496d5e0b439dd73b68eb89575bea417ff32d7429814b950a18e544c2522279e2e675d4fc1e10a26cef05b584e6e2218ca9b98","vrfPublicKey":"0xd8f9a28323ba50c6597e7f03d3956029ba428084d12cb8707c90eee29f7eca45","account":"0xfcc7d5aa2942894234a942272fba50951c7fd0c0"}`,
	// other
	`{"id":"0x3b1cd6589e4b62ad98ebf2ece94ef2253e9befe1b6322aecd342b9ea9f1ea0fa","publicKey":"0x414b0965e58657db146e599273c082357e878102c625c8f1df906ba30d28eaec2fa931f331cabe4e0f1d2afb25f9b10ee736ddccb4dba0de198dc181fbb4d71f3cb0ea66fff092a065de79d0bf81386a8250e12cd52653644e8ca61e5f5a2c4d5d50c5c4ccb2818672f6009c47c6a10ed2daf2a8b858e5af5d5f61fdb8e29be8","vrfPublicKey":"0x97bcc0ab5707fde0147997110473226ada5ee94df8e714066d44d35d0d8b12b7","account":"0xaf59f00a17123d5c223c98ed71360c02064acbf7"}`,
	`{"id":"0x76aa6b30168c4d169effeb6934dc2d83923e934ddc9461b05e71f32463edc39e","publicKey":"0x138ebaa6e587614cbcbde396c66d192a64a0617c40296329d1c703e67055b74628287b0ba6017b6d27972bf4a1de67c0e41c2bf5a0488aa0b54ba4208f8de43b445266c648ea9563eeb17372310af2a289c8dc7af14c330bd816c36b9783365e66779812b760f18ddc7dd70adacc03997c485e27767fe02556070c360407c871","vrfPublicKey":"0xc63b7f1f03bac64b2de75dd5bcb06e7f03f759de9a466e384de8f6e6f59bdf94","account":"0xb8a91cb3cbdc1909d88dedb138362b1050f743ca"}`,
	`{"id":"0x98c23e0baa546e2ead8aa46daf8b7cc2dc3422a541bf15eabd4411ba1cfde61f","publicKey":"0x5a4a200b405079c4352812e870bdf1634912a081ee0debcbb3875cb265ba02db4bb64894b732eee8f353d607652e092bc6d0404e1f4636ca23210b0429f272b34ab9f376d77a7b716ed862d011e0c505912fa4fb6507589af4d5dadda5322c3a2625b326dfcfc205a56d8e9e953b294b8b19934df62bc55685d7438d7431a79b","vrfPublicKey":"0x3eed59dda3fa4ec7096e7563ef60b8e89847993bab91a2f29b8cd1c7674a6ba4","account":"0x0e7f47cd54aa66b69049ee543605b39876830ce6"}`,
	`{"id":"0x844bfe9ef3b798bd558256ec5955d5d20ea461e3ef99b0421c7e584caae0b0fc","publicKey":"0x833ba4b000717253b7395bf186b9456dcc5c116566e8fceea25a25e4e1ccad0837cc56e62c7af2a369ee4f7c9002c788d3ed2092519a129c47bfdd81b32b36366f0d21c8e9305b434e76b05adece1f71ed5a280b42350c7d557aa1bdb919b33b2293b99f3fe10d0c4546392b4b3f3a1a1c18b537ca11256f6b122f5b0c4876c5","vrfPublicKey":"0x55996c2426d3551cd032c6fcee8ce64244717b501a307f3bd71ba3611a48652b","account":"0x361973b04435cd72c0162601b7f65745656f7613"}`,
	`{"id":"0x68e1cb6a6c3c467860713d7de82a007dde8cedf5990f01823fb549fa2343a216","publicKey":"0x547bb351407a5cdc7c74b080b3659fe25f2e6ae3939e315942e26acec1948ba8669177026f4897e407bf5f321a2e1e66f7e079288fc466053f49a17a5fc02bca85661295f30cc5b4a62bd002818a55ee5c396bd1c293ab6d33c43ff9378204c96515cbc154e8c6f8a58d438862a7bfa58a6ac4c3887fda599203baa2a765e7ea","vrfPublicKey":"0x41fe4c0825c0b8289d6c324dea3ceb3c28f5226de3c6f7f99282010dc4d92054","account":"0x156a3f76ffe55fa7f2bae306b6da05ddaf769298"}`,
	`{"id":"0x802753165930ab14274f774f5be411538474b9a345a6b1803c531ad9e9615e38","publicKey":"0x099eea4abe1e1f99a894fe7b84255038bbe20c461815684d84a9af2b43e2114a3346e8facdccb05d01ca4b5f807399e6c551e0e808871645ce242f508d9ec229813d816fbcc2add5422d5a6c5e7cc3b5458d384a971ddeb864a44e7c2d61aff7304a9495bd6548fead4e5d50e86199bebec05fa33a3393f4972d5f7d3b9be4ec","vrfPublicKey":"0x8cff8563ddffb9ec125e3f01336f8ce8cde1b55a260c51d531b160f9be569401","account":"0xc419c0861a0653e7a7def0a52130b42bd53b6569"}`,
	`{"id":"0x41b76935e4ce7016aca1f75e8de85d3fa2c92e16175754b65fc53a8ba00f6de9","publicKey":"0x4bb94a781da02fb0627356b345baaa0debb644b4e9ac5acf1d63b6fefd18390d778c0cac5ae997c9b99aad95a03f42aba0d98885c905cce7951a8c129902a1fe5f14f35af14675ef61140d0e29a08be235f72f16cd0d3efcbd4fb3fea91bf8881b5056f30901685d0c9ab948250028c5c78397661b91d91359cea6fed8873b26","vrfPublicKey":"0xdd04769ffc8c856863b2358b8fa068ebb70161a75335517994aaa969534f92f2","account":"0x2868a56e20e167c07e97f214f730bf33c67ba0f4"}`,
	`{"id":"0x82333d4a115ada7cb79d08f98ddffe87bdf77029871ef8e769dc965ae6dcf5b9","publicKey":"0x61eefe6921f3928346947642a8ce91570af149bf965356dcb223b446fbf0d65a3bfe30797f8b2032e6fe13a5fed9008b8997099003beceb70c335d0ebaaadd275b19a189305af7d7e06abdb592e80f7cbeade37472e830692fc31ab95422605b40979d32c0b0720b9c8916c6660b2992af159e5e65fae946c118b3aaf6bf453b","vrfPublicKey":"0x380dcd523a69eee3a988b771dad517b1524c87f2bd89f37445d5706f3052a535","account":"0x90ffbc662a1d2853026ef75f8f43aab3594e22ff"}`,
	`{"id":"0x858978878fb7a268bb9ade492bb71fbd563032aeb6a6e484ad838f656684831d","publicKey":"0x1a439166c1156ed3df81b10a0e4b1abb8f274cce2b10ad197dbd8aec3f31f05a07ab2c9b7448b4028329936ab8ca10fd57e190842feca846da67a4101ea95a43304414603119b3e6da714601b20e3cb4827b1ddf56e45b70b5591d8bf0d6f9a76dda40e423852547f07b3a6baa8231f27c6a8a9d3138d16253b43d422ffa8623","vrfPublicKey":"0xfed763fd12228f830f019d519ddcffc007bcdfc7e55f799d1cc7a1868cd5be22","account":"0x0c8ee0183058494b9c06619c6cd7840c9ce05b9d"}`,
	`{"id":"0x9b1a7648051e3543c3782d52b6aa73ce264d2a2656e380aa594ef0b55aa30c86","publicKey":"0x78f174351e5cc28f62c4a20daf031036f34b52c94cb9903cb903e7d3dbdd1cf062f98be1a3c399706940afb9b864e4085f0509d1be861d2d8e202955b92f5f9f87f1fb1042e2729204cbcad41e35ae90eef3881e1d3ea03910ce689f03094a461e2335f3a14d5d140a263b9ea2ad64052776097940a71477f6ada5b653788970","vrfPublicKey":"0x5776c0051ed094059658237b042a51187d80c6a789c585e38e6295b274820666","account":"0x6d4d0432b83201e5bb7f22075655f2ebe637f0eb"}`,
	`{"id":"0xd76038619814e2bf67cd5234c2decd4f04862d9b140bff8e510582a36ac383d0","publicKey":"0x6250353baedc56732a9a389284bd70fa45076f93ae0c6779c7b82cdce933b54945903c13d1676961be8e446c938ed4c8c3c2468f095e286c83f4da67a7562f9f690d020ae36745237c1f562f6238d21e52b73487e6cdaa50f529a67f8573d0323b393bec50119b966187ba8d346a8edf4defba99b89e8ff4c204f910514a3b99","vrfPublicKey":"0x710ce9e32e99afd2e64d40396594d8d1827788ec79ec68b530549e1829a3044b","account":"0x84e056985c45e4de9f28499fb23facde92999309"}`,
	`{"id":"0x5c448198a659a5f3a4fb77bc58e5405940b852d429d13d2656fc510f8f11265a","publicKey":"0x5d7bdfb953f53c03db1e298e00b50254b1768c5ebc0f4d96b9e13ea216ece2da07a01e0e13cbf8774798edbffe7af1230b461e3081a7dbb636e0266a31242e25377fe958fcd6ad8d3ae229dbfabd7b26435c302732d5ceadba5d03834226e57f83a15c785916632abcb3afa64616b8b95c14abee67c8e85ee299c540213ce892","vrfPublicKey":"0x73a3e39aa0f782dc46944b64b7ab4172a47eb6b701e01ece8cffd2139f77e1c4","account":"0xbfc5f31e40b42dad579e4fc433cda987e44064c1"}`,
	`{"id":"0x303f57dc44068f22e81b00f3c759e1a31e371ca119f8ca135345b0851e8e20f4","publicKey":"0x6dadeedcc2787fbea8fcb19edff9420b60e672ecdf7170f50644c680e7abdb1825fc5c77c379654b5574fd894c8afcebbb6f648ba299459337dd3952afbf9fb65f2b348f970408ee5d3befeccf9d65a08273b7d04c1be3dbc42227e7bb2a49982f4a85cd02dddf1b4f5635e3feac068733d85c2c021dee63cfa1c998f5a56420","vrfPublicKey":"0xda82febd7cb9c7f3508ed34c0ea48868d37b3ac91cf21ca637c4b1a0a73d7201","account":"0x033e61266a278a7248229ba565b6c236b7d30b7c"}`,
	`{"id":"0x8c08ca4477b26f8ba5b96f7c76dc5bbea702cccebdcd71f91c96c973dc5677d2","publicKey":"0x445323c2443adc2aaab0aa46e638491979815835d14e21d2e1233e9ed70a18e81d40a331fc4be1a0d4b09c43525fdc1aa6c7201dc80172c0caa85d50b6e14d980973c7911d7c458ebd1bdd91a4b9088811b8b88b9146b9ec9f97abfc1e960dde1e2a31aa4e0bad1a59708d250f83b9e4a6feefc7ca6fe7235a654517ca9af077","vrfPublicKey":"0x2bc74c2338e7c3b0611380529d30a0f338c551ec493b4a4f3bec004a398ba7f2","account":"0xf88c7763aca3fc3d46867c8c605d4a636ddc43dc"}`,
}

var devValidatorAccounts = [20]string{
	//other
	`0x0e05d86e7943d7f041fabde02f25d53a2aa4cc29`,
	`0xe9b59d7af13bf6d3f838da7f73c2e369802ea211`,
	`0x1aab2207e31dff81240fc4976c301ab0a0e0da26`,
	`0x46ee7dafba4797d76565a730c64ab92b08b47eb5`,
	`0xda686b9b5ad32404a2d1d9b1e4f84e67f72a884a`,
	`0xaeab1151cb42756cbd409269154c0461c9c3df3b`,
	`0x7f0746723b141b79b802eb48eb556178fd622201`,
	`0xaba4ede96364f129baa98f112f636842a676b9bf`,
	`0x1f3a2e8f07be5839005c6d1997358684066f593f`,
	`0xfcda6f6d16d0c31234c51dc15bf71b3242b09763`,
	`0xc2eb24387458baebcf7f8517c2a38cf96fba704f`,
	`0xfae00bc664af03a99ec3a4ae3194e4bdac093450`,
	`0x7759e04cc420a6d5c12aca77e045e82ca6a55730`,
	`0x8192d698b4fa840a33e4a44aeee32f3671c8956e`,
	// tuntunhz
	`0x56b1fc865ad0c87f46f804145a861b38fcbafb99`,
	`0x0ef90c9cc936c2e3117d76c1ffb28391f8cebbca`,
	`0x22b00137e24a708609fdb88ee156dabe041b158b`,
	// mixmarvel
	`0x8cd20feb1b8c7e5378ab1b0f6d68846b29d4f0be`,
	`0xb4ca0fbec728a32845d4c44fdfb3df05645f5229`,
	`0x6ca0685b1f337ee1503ed83d2299b925adc9b804`,
}

func genDevGenesisBlock(stateDB *account.AccountDB, triedb *trie.NodeDatabase, genesisInfo []*types.GenesisInfo) *types.Block {
	block := new(types.Block)
	pv := big.NewInt(0)
	block.Header = &types.BlockHeader{
		Height:       0,
		ExtraData:    common.Sha256([]byte("Rangers Protocol")),
		CurTime:      time.Date(2023, 04, 10, 0, 0, 0, 0, time.UTC),
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

	genesisProposers := getDevGenesisProposer()
	addMiners(genesisProposers, stateDB)

	verifyMiners := make([]*types.Miner, 0)
	for _, genesis := range genesisInfo {
		for i, member := range genesis.Group.Members {
			miner := &types.Miner{Type: common.MinerTypeValidator, Id: member, PublicKey: genesis.Pks[i], VrfPublicKey: genesis.VrfPKs[i], Stake: common.ValidatorStake}
			miner.Account = common.FromHex(devValidatorAccounts[i])
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
