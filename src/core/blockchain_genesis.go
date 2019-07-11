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

	block.Header.RequestIds = make(map[string]uint64)
	block.Header.Signature = common.Sha256([]byte("tas"))
	block.Header.Random = common.Sha256([]byte("tas_initial_random"))

	tenThousandGxCoin := big.NewInt(0).SetUint64(1000000000 * 10000)

	for _, genesis := range genesisInfo {
		for _, mem := range genesis.Group.Members {
			addr := common.BytesToAddress(mem)
			stateDB.SetBalance(addr, tenThousandGxCoin)
		}
	}

	genesisProposers := getGenesisProposer("")
	for _, proposer := range genesisProposers {
		stateDB.SetBalance(common.BytesToAddress(proposer.Id), tenThousandGxCoin)
	}

	//游戏账户充值
	//subAccount := types.SubAccount{Balance: tenThousandGxCoin}
	//stateDB.UpdateSubAccount(common.HexToAddress("0x5d6fd9f54085490457cd534d4bdf90289fae65a7"), "0x5d6fd9f54085490457cd534d4bdf90289fae65a7", subAccount)

	stage := stateDB.IntermediateRoot(false)
	logger.Debugf("GenesisBlock Stage1 Root:%s", stage.Hex())

	verifyMiners := make([]*types.Miner, 0)
	for _, genesis := range genesisInfo {
		for i, member := range genesis.Group.Members {
			miner := &types.Miner{Id: member, PublicKey: genesis.Pks[i], VrfPublicKey: genesis.VrfPKs[i], Stake: 1000000000 * (100)}
			verifyMiners = append(verifyMiners, miner)
		}
	}
	MinerManagerImpl.addGenesesVerifier(verifyMiners, stateDB)
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
