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
	"x/src/service"
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
		CurTime:      time.Date(2018, 6, 14, 10, 0, 0, 0, time.Local),
		ProveValue:   pv,
		TotalQN:      0,
		Transactions: make([]common.Hashes, 0), //important!!
		EvictedTxs:   make([]common.Hash, 0),   //important!!
		Nonce:        ChainDataVersion,
	}

	block.Header.RequestIds = make(map[string]uint64)
	block.Header.Signature = common.Sha256([]byte("tuntunhz"))
	block.Header.Random = common.Sha256([]byte("RocketProtocolVRF"))

	genesisProposers := getGenesisProposer("")
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

	// 测试用
	service.FTManagerInstance.PublishFTSet(service.FTManagerInstance.GenerateFTSet("tuntun", "pig", "hz", "0", "hz", "10086", 0), stateDB)
	service.NFTManagerInstance.PublishNFTSet(service.NFTManagerInstance.GenerateNFTSet("tuntunhz", "tuntun", "t", "hz", "hz", 0, "10000"), stateDB)
	stateDB.SetFT(common.HexToAddress("0x69564f3eccc4aedabde33bd5cb350b9829deced1"), "official-ETH.ETH", big.NewInt(10000000000))
	stateDB.SetFT(common.HexToAddress("0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54"), "official-ETH.ETH", big.NewInt(10000000000))

	root, _ := stateDB.Commit(true)
	triedb.Commit(root, false)
	block.Header.StateTree = common.BytesToHash(root.Bytes())
	block.Header.Hash = block.Header.GenHash()

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
		MinerManagerImpl.AddMiner(miner, accountdb)
	}
}
