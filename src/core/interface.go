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

package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"math/big"
	"time"
)

type GroupIterator struct {
	current *types.Group
}

func GetBlockChain() BlockChain {
	return blockChainImpl
}

func GetGroupChain() GroupChain {
	return groupChainImpl
}

type BlockChain interface {
	CastBlock(timestamp time.Time, height uint64, proveValue *big.Int, proveRoot common.Hash, qn uint64, castor []byte, groupId []byte) *types.Block

	GenerateBlock(bh types.BlockHeader) *types.Block

	//验证一个铸块（如本地缺少交易，则异步网络请求该交易）
	//返回:=0, 验证通过；=-1，验证失败；=1，缺少交易，已异步向网络模块请求
	//返回缺失交易列表
	VerifyBlock(bh types.BlockHeader) ([]common.Hashes, int8)

	AddBlockOnChain(source string, b *types.Block, situation types.AddBlockOnChainSituation) types.AddBlockResult

	Height() uint64

	TotalQN() uint64

	TopBlock() *types.BlockHeader

	CurrentBlock() *types.Block

	QueryBlockByHash(hash common.Hash) *types.Block

	QueryBlock(height uint64) *types.Block

	GetBalance(address common.Address) *big.Int

	GetTransaction(txHash common.Hash) (*types.Transaction, error)

	Remove(block *types.Block) bool

	Close()

	GetVerifyHash(height uint64) (common.Hash, error)

	HasBlockByHash(hash common.Hash) bool
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
}
