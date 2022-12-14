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

func genSubGenesisBlock(stateDB *account.AccountDB, triedb *trie.NodeDatabase, genesisInfo []*types.GenesisInfo) *types.Block {
	block := new(types.Block)
	pv := big.NewInt(0)
	block.Header = &types.BlockHeader{
		Height:       0,
		ExtraData:    common.Sha256([]byte(common.Genesis.Name)),
		CurTime:      time.UnixMilli(common.Genesis.GenesisTime),
		ProveValue:   pv,
		TotalQN:      0,
		Transactions: make([]common.Hashes, 0), //important!!
		EvictedTxs:   make([]common.Hash, 0),   //important!!
		Nonce:        ChainDataVersion,
	}

	block.Header.RequestIds = make(map[string]uint64)
	block.Header.Signature = common.Sha256([]byte(common.Genesis.Name))
	block.Header.Random = common.Sha256([]byte(common.Genesis.Name))

	//创建创始合约
	proxy := createGenesisContract(block.Header, stateDB)

	genesisProposers := getSubGenesisProposer()
	addMiners(genesisProposers, stateDB)

	verifyMiners := make([]*types.Miner, 0)
	for _, genesis := range genesisInfo {
		for i, member := range genesis.Group.Members {
			miner := &types.Miner{Type: common.MinerTypeValidator, Id: member, PublicKey: genesis.Pks[i], VrfPublicKey: genesis.VrfPKs[i], Stake: common.ValidatorStake}
			miner.Account = common.FromHex(validatorAccounts[i])
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

	root, _ := stateDB.Commit(true)
	triedb.Commit(root, false)
	block.Header.StateTree = common.BytesToHash(root.Bytes())
	block.Header.Hash = block.Header.GenHash()

	return block
}

func getSubGenesisProposer() []*types.Miner {
	miners := make([]*types.Miner, 0)
	for _, data := range common.Genesis.ProposerInfo {
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
		miners = append(miners, &miner)
	}
	return miners
}
