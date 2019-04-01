package core

import (
	"x/src/common"
	"math/big"
	"x/src/middleware/types"
	"x/src/storage/account"
)

type ExecutedTransaction struct {
	Receipt     *types.Receipt
	Transaction *types.Transaction
}

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
	CastBlock(height uint64, proveValue *big.Int, proveRoot common.Hash, qn uint64, castor []byte, groupId []byte) *types.Block

	GenerateBlock(bh types.BlockHeader) *types.Block

	//验证一个铸块（如本地缺少交易，则异步网络请求该交易）
	//返回:=0, 验证通过；=-1，验证失败；=1，缺少交易，已异步向网络模块请求
	//返回缺失交易列表
	VerifyBlock(bh types.BlockHeader) ([]common.Hash, int8)

	AddBlockOnChain(source string, b *types.Block, situation types.AddBlockOnChainSituation) types.AddBlockResult

	Height() uint64

	TotalQN() uint64

	TopBlock() *types.BlockHeader

	QueryBlockByHash(hash common.Hash) *types.Block

	QueryBlock(height uint64) *types.Block

	GetBalance(address common.Address) *big.Int

	GetNonce(address common.Address) uint64

	GetTransaction(txHash common.Hash) (*types.Transaction, error)

	GetTransactionPool() TransactionPool

	Remove(block *types.Block) bool

	Clear() error

	Close()

	GetBonusManager() *BonusManager

	GetAccountDBByHash(hash common.Hash) (*account.AccountDB, error)

	GetVerifyHash(height uint64) (common.Hash, error)
}

type GroupChain interface {
	AddGroup(group *types.Group) error

	GetGroupById(id []byte) *types.Group

	GetGroupByHeight(height uint64) (*types.Group)

	LastGroup() *types.Group

	Count() uint64

	Close()

	Iterator() *GroupIterator
}

type TransactionPool interface {
	PackForCast() []*types.Transaction

	//add new transaction to the transaction pool
	AddTransaction(tx *types.Transaction) (bool, error)

	//rcv transactions broadcast from other nodes
	AddBroadcastTransactions(txs []*types.Transaction)

	//add  local miss transactions while verifying blocks to the transaction pool
	AddMissTransactions(txs []*types.Transaction)

	GetTransaction(hash common.Hash) (*types.Transaction, error)

	GetTransactionStatus(hash common.Hash) (uint, error)

	GetExecuted(hash common.Hash) *ExecutedTransaction

	GetReceived() []*types.Transaction

	TxNum() uint64

	MarkExecuted(receipts types.Receipts, txs []*types.Transaction, evictedTxs []common.Hash)

	UnMarkExecuted(txs []*types.Transaction)

	AddExecuted(tx *types.Transaction) error

	Clear()
}
