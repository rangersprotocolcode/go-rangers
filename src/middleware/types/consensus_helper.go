// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"math/big"

	"com.tuntun.rocket/node/src/common"
)

type GenesisInfo struct {
	Group  Group
	VrfPKs [][]byte
	Pks    [][]byte
}

/*
	共识接口合集
*/
type ConsensusHelper interface {
	//generate genesis group and member pk info
	GenerateGenesisInfo() []*GenesisInfo

	//vrf prove to value
	VRFProve2Value(prove *big.Int) *big.Int

	//bonus for proposal a block
	ProposalBonus() *big.Int

	//bonus for packing one bonus transaction
	PackBonus() *big.Int

	//generate verify hash of the block for current node
	VerifyHash(b *Block) common.Hash

	//check the prove root hash for weight node when add block on chain
	CheckProveRoot(bh *BlockHeader) (bool, error)

	//check the new block
	//mainly verify the cast legality, group signature
	VerifyNewBlock(bh *BlockHeader, preBH *BlockHeader) (bool, error)

	//verify the blockheader: mainly verify the group signature
	VerifyBlockHeader(bh *BlockHeader) (bool, error)

	VerifyGroupSign(groupPubkey []byte, blockHash common.Hash, sign []byte) (bool, error)

	//check group
	CheckGroup(g *Group) (bool, error)

	VerifyGroupForFork(g *Group, preGroup *Group, parentGroup *Group, baseBlock *Block) (bool, error)
}
