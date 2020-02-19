package core

import (
	"x/src/common"
	"math/big"
	"x/src/middleware/types"
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

	GetGroupByHeight(height uint64) (*types.Group)

	LastGroup() *types.Group

	Count() uint64

	Close()

	Iterator() *GroupIterator

	GetAvailableGroupsByMinerId(height uint64, minerId []byte) []*types.Group
}
