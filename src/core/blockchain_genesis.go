package core

import (
	"x/src/common"
	"math/big"
	"x/src/middleware/types"
	"time"
	"x/src/storage/trie"
	"x/src/storage/account"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"x/src/consensus/groupsig"
	"x/src/consensus/vrf"
)

const ChainDataVersion = 2
const defaultGenesisProposerPath = "genesis_proposer.yaml"

var emptyHash = common.Hash{}

type GenesisProposer struct {
	MinerId     string `yaml:"minerId"`
	MinerPubKey string `yaml:"minerPubkey"`
	VRFPubkey   string `yaml:"vrfPubkey"`
}

func (chain *blockChain) insertGenesisBlock() {
	state, err := account.NewAccountDB(common.Hash{}, chain.stateDB)
	if nil == err {
		genesisBlock := genGenesisBlock(state, chain.stateDB.TrieDB(), consensusHelper.GenerateGenesisInfo())
		logger.Debugf("GenesisBlock Hash:%s,StateTree:%s", genesisBlock.Header.Hash.String(), genesisBlock.Header.StateTree.Hex())
		blockByte, _ := types.MarshalBlock(genesisBlock)
		chain.saveBlockByHash(genesisBlock.Header.Hash, blockByte)

		headerByte, err := types.MarshalBlockHeader(genesisBlock.Header)
		if err != nil {
			logger.Errorf("Marshal block header error:%s", err.Error())
		}
		chain.saveBlockByHeight(genesisBlock.Header.Height, headerByte)

		chain.updateLastBlock(state, genesisBlock.Header, headerByte)
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
		ExtraData:    common.Sha256([]byte("tas")),
		CurTime:      time.Date(2018, 6, 14, 10, 0, 0, 0, time.Local),
		ProveValue:   pv,
		TotalQN:      0,
		Transactions: make([]common.Hash, 0), //important!!
		EvictedTxs:   make([]common.Hash, 0), //important!!
		Nonce:        ChainDataVersion,
	}

	block.Header.Signature = common.Sha256([]byte("tas"))
	block.Header.Random = common.Sha256([]byte("tas_initial_random"))

	tenThousandTasBi := big.NewInt(0).SetUint64(1000000000 * 10000)

	//管理员账户
	stateDB.SetBalance(common.HexStringToAddress("0xf77fa9ca98c46d534bd3d40c3488ed7a85c314db0fd1e79c6ccc75d79bd680bd"), big.NewInt(0).SetUint64(1000000000*(5000000)))
	stateDB.SetBalance(common.HexStringToAddress("0xb055a3ffdc9eeb0c5cf0c1f14507a40bdcbff98c03286b47b673c02d2efe727e"), big.NewInt(0).SetUint64(1000000000*(5000000)))

	//创世账户
	for _, genesis := range genesisInfo {
		for _, mem := range genesis.Group.Members {
			addr := common.BytesToAddress(mem)
			stateDB.SetBalance(addr, tenThousandTasBi)
		}
	}

	stateDB.SetBalance(common.HexStringToAddress("0xe75051bf0048decaffa55e3a9fa33e87ed802aaba5038b0fd7f49401f5d8b019"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xd3d410ec7c917f084e0f4b604c7008f01a923676d0352940f68a97264d49fb76"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x9d2961d1b4eb4af2d78cb9e29614756ab658671e453ea1f6ec26b4e918c79d02"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xa7a4b347a626d2353050328763f74cd85f83173d6886b24501c39b5c456446d2"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xf664571c43c13c49ccf032b9d823bedc9f62e5a5858a80d42cd5ee6d897237f0"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x86805452068d92d647ae9b7a048a09515c875d71eb3d0d3d788c3f3334aaf124"), tenThousandTasBi)

	stage := stateDB.IntermediateRoot(false)
	logger.Debugf("GenesisBlock Stage1 Root:%s", stage.Hex())
	miners := make([]*types.Miner, 0)

	for _, genesis := range genesisInfo {
		for i, member := range genesis.Group.Members {
			miner := &types.Miner{Id: member, PublicKey: genesis.Pks[i], VrfPublicKey: genesis.VrfPKs[i], Stake: 1000000000 * (100)}
			miners = append(miners, miner)
		}
	}

	MinerManagerImpl.addGenesesVerifier(miners, stateDB)
	genesisProposers := getGenesisProposer("")
	MinerManagerImpl.addGenesesProposer(genesisProposers, stateDB)

	stage = stateDB.IntermediateRoot(false)
	logger.Debugf("GenesisBlock Stage2 Root:%s", stage.Hex())
	stateDB.SetNonce(common.BonusStorageAddress, 1)
	stateDB.SetNonce(common.HeavyDBAddress, 1)
	stateDB.SetNonce(common.LightDBAddress, 1)

	root, _ := stateDB.Commit(true)
	logger.Debugf("GenesisBlock final Root:%s", root.Hex())
	triedb.Commit(root, false)
	block.Header.StateTree = common.BytesToHash(root.Bytes())
	block.Header.Hash = block.Header.GenHash()

	logger.Debugf("GenesisBlock %+v", block.Header)
	return block
}

func getGenesisProposer(path string) []*types.Miner {
	if path == "" {
		path = defaultGenesisProposerPath
	}
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		panic("Init genesis proposer error:" + err.Error())
	}

	var genesisProposers []GenesisProposer
	err = yaml.UnmarshalStrict(bytes, &genesisProposers)
	if err != nil {
		panic("Unmarshal member info error:" + err.Error())
	}
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
			Stake:        1000000000 * (100),
			Type:         types.MinerTypeHeavy,
			Status:       types.MinerStatusNormal,
		}
		miners = append(miners, &miner)
	}
	return miners
}
