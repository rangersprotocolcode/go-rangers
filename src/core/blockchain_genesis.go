package core

import (
	"x/src/common"
	"math/big"
	"x/src/middleware/types"
	"time"
	"x/src/storage/trie"
	"x/src/storage/account"
)

const ChainDataVersion = 2

var emptyHash = common.Hash{}

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

func genGenesisBlock(stateDB *account.AccountDB, triedb *trie.NodeDatabase, genesisInfo *types.GenesisInfo) *types.Block {
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
	for _, mem := range genesisInfo.Group.Members {
		addr := common.BytesToAddress(mem)
		stateDB.SetBalance(addr, tenThousandTasBi)
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
	for i, member := range genesisInfo.Group.Members {
		miner := &types.Miner{Id: member, PublicKey: genesisInfo.Pks[i], VrfPublicKey: genesisInfo.VrfPKs[i], Stake: 1000000000 * (100)}
		miners = append(miners, miner)
	}
	MinerManagerImpl.addGenesesMiner(miners, stateDB)
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
