// Copyright 2020 The RangersProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RangersProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RangersProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/middleware/types"
	"math/big"
	"time"
)

type GroupIterator struct {
	current *types.Group
}

type GroupForkIterator struct {
	current *types.Group
}

func GetBlockChain() BlockChain {
	return blockChainImpl
}

func GetGroupChain() GroupChain {
	return groupChainImpl
}

type BlockChain interface {
	CastBlock(timestamp time.Time, height uint64, proveValue *big.Int, proveRoot common.Hash, qn uint64, castor []byte, groupId []byte) (types.BlockHeader, bool)

	GenerateBlock(bh types.BlockHeader) *types.Block

	VerifyBlock(bh *types.BlockHeader) ([]common.Hashes, int8)

	AddBlockOnChain(b *types.Block) types.AddBlockResult

	Height() uint64

	TotalQN() uint64

	TopBlock() *types.BlockHeader

	QueryBlockByHash(hash common.Hash) *types.Block

	QueryBlock(height uint64) *types.Block

	QueryBlockHeaderByHeight(height interface{}, cache bool) *types.BlockHeader

	GetBalance(address common.Address) *big.Int

	GetTransaction(txHash common.Hash) (*types.Transaction, error)

	Close()

	GetVerifyHash(height uint64) (common.Hash, error)

	HasBlockByHash(hash common.Hash) bool

	GetBlockHash(height uint64) common.Hash
}

type GroupChain interface {
	AddGroup(group *types.Group) error

	GetGroupById(id []byte) *types.Group

	GetGroupByHeight(height uint64) *types.Group

	LastGroup() *types.Group

	Count() uint64

	Close()

	Iterator() *GroupIterator

	GetAvailableGroupsByMinerId(height uint64, minerId []byte) []*types.Group

	GetSyncGroupsById(id []byte) []*types.Group

	SaveJoinedGroup(id []byte, value []byte) bool

	GetJoinedGroup(id []byte) ([]byte, error)

	DeleteJoinedGroup(id []byte) bool

	ForkIterator() *GroupForkIterator
}
