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
	"com.tuntun.rangers/node/src/middleware/db"
	"com.tuntun.rangers/node/src/middleware/types"
	"fmt"
	lru "github.com/hashicorp/golang-lru"
	"math/big"
	"os"
	"sort"
	"strconv"

	"com.tuntun.rangers/node/src/middleware"
	"com.tuntun.rangers/node/src/service"
	"com.tuntun.rangers/node/src/storage/account"
	"com.tuntun.rangers/node/src/utility"
	"encoding/binary"
	"math"
	"time"
)

const (
	latestBlockKey = "bcurrent"

	addBlockMark    = "addBlockMark"
	removeBlockMark = "removeBlockMark"

	hashDBPrefix       = "block"
	heightDBPrefix     = "height"
	verifyHashDBPrefix = "verifyHash"

	topBlocksCacheSize = 100
)

var blockChainImpl *blockChain

type blockChain struct {
	init bool

	latestBlock *types.BlockHeader
	requestIds  map[string]uint64

	topBlocks         *lru.Cache
	futureBlocks      *lru.Cache
	verifiedBlocks    *lru.Cache
	verifiedBodyCache *lru.Cache

	hashDB       db.Database
	heightDB     db.Database
	verifyHashDB db.Database

	transactionPool service.TransactionPool
}

type castingBlock struct {
	state    *account.AccountDB
	receipts types.Receipts
}

func initBlockChain() error {
	chain := &blockChain{}
	chain.transactionPool = service.GetTransactionPool()

	var err error
	chain.topBlocks, err = lru.New(100)
	if err != nil {
		logger.Errorf("Init cache error:%s", err.Error())
		return err
	}

	chain.futureBlocks, err = lru.New(topBlocksCacheSize)
	if err != nil {
		logger.Errorf("Init cache error:%s", err.Error())
		return err
	}

	chain.verifiedBlocks, err = lru.New(20)
	if err != nil {
		logger.Errorf("Init cache error:%s", err.Error())
		return err
	}

	chain.verifiedBodyCache, err = lru.New(10)
	if err != nil {
		logger.Errorf("Init cache error:%s", err.Error())
		return err
	}

	chain.hashDB, err = db.NewDatabase(hashDBPrefix)
	if err != nil {
		logger.Debugf("Init block chain error! Error:%s", err.Error())
		return err
	}
	chain.heightDB, err = db.NewDatabase(heightDBPrefix)
	if err != nil {
		logger.Debugf("Init block chain error! Error:%s", err.Error())
		return err
	}
	chain.verifyHashDB, err = db.NewDatabase(verifyHashDBPrefix)
	if err != nil {
		logger.Debugf("Init block chain error! Error:%s", err.Error())
		return err
	}

	chain.latestBlock = chain.QueryBlockHeaderByHeight([]byte(latestBlockKey), false)
	if chain.latestBlock == nil {
		chain.insertGenesisBlock()
	} else {
		chain.ensureChainConsistency()

		state, err := middleware.AccountDBManagerInstance.GetAccountDBByHash(chain.latestBlock.StateTree)
		if nil != err {
			panic(err)
		}
		middleware.AccountDBManagerInstance.SetLatestStateDB(state, chain.latestBlock.RequestIds, chain.latestBlock.Height)
		logger.Debugf("refreshed latestStateDB, state: %v, height: %d", chain.latestBlock.StateTree, chain.latestBlock.Height)

		if !chain.versionValidate() {
			fmt.Println("Illegal data version! Please delete the directory d0 and restart the program!")
			os.Exit(0)
		}
		chain.buildCache(topBlocksCacheSize)
		common.SetBlockHeight(chain.latestBlock.Height)
	}
	chain.init = true
	blockChainImpl = chain

	return nil
}

func (chain *blockChain) CastBlock(timestamp time.Time, height uint64, proveValue *big.Int, proveRoot common.Hash, qn uint64, castor []byte, groupid []byte) (types.BlockHeader, bool) {
	middleware.PerfLogger.Infof("start cast block. last: %v height: %v", utility.GetTime().Sub(timestamp), height)
	defer middleware.PerfLogger.Infof("end cast block. last: %v height: %v", utility.GetTime().Sub(timestamp), height)

	middleware.RLockBlockchain("castblock")

	latestBlock := chain.latestBlock
	if latestBlock == nil {
		logger.Errorf("Block chain lastest block is nil!")
		middleware.RUnLockBlockchain("castblock")
		return types.BlockHeader{}, false
	}
	if height <= latestBlock.Height {
		logger.Errorf("Fail to cast block: height problem. height:%d, local height:%d", height, latestBlock.Height)
		middleware.RUnLockBlockchain("castblock")
		return types.BlockHeader{}, false
	}

	state, err := middleware.AccountDBManagerInstance.GetAccountDBByHash(latestBlock.StateTree)
	if err != nil {
		logger.Errorf("Fail to new account db while casting block!Latest block height:%d,error:%s", latestBlock.Height, err.Error())
		middleware.RUnLockBlockchain("castblock")
		return types.BlockHeader{}, false
	}

	txs := types.Transactions(chain.transactionPool.PackForCast(height, state))
	if 0 != len(txs) {
		sort.Sort(txs)
	}
	middleware.RUnLockBlockchain("castblock")

	bh := types.BlockHeader{
		CurTime:    timestamp,
		Height:     height,
		ProveValue: proveValue,
		Castor:     castor,
		GroupId:    groupid,
		TotalQN:    latestBlock.TotalQN + qn,
		StateTree:  common.BytesToHash(latestBlock.StateTree.Bytes()),
		PreHash:    latestBlock.Hash,
		PreTime:    latestBlock.CurTime,
	}
	bh.RequestIds = getRequestIdFromTransactions(txs, latestBlock.RequestIds)

	middleware.PerfLogger.Infof("fin cast object. last: %v height: %v", utility.GetTime().Sub(timestamp), height)

	block := new(types.Block)
	block.Transactions = txs

	if common.IsProposal020() {
		transactionHashes := make([]common.Hashes, len(txs))
		for i, transaction := range txs {
			hashes := common.Hashes{}
			hashes[0] = transaction.Hash
			hashes[1] = transaction.SubHash
			transactionHashes[i] = hashes
		}
		bh.Transactions = transactionHashes
		bh.TxTree = calcTxTree(txs)
		bh.ReceiptTree = common.Hash{}
		bh.StateTree = common.Hash{}
		bh.EvictedTxs = make([]common.Hash, 0)
		bh.Hash = bh.GenHash()

		bh2 := *&bh
		block.Header = &bh2
		go chain.runTransactions(block, state, height, timestamp)
	} else {
		block.Header = &bh
		chain.runTransactions(block, state, height, timestamp)
	}

	return bh, true
}

func (chain *blockChain) runTransactions(block *types.Block, state *account.AccountDB, height uint64, timestamp time.Time) {
	executor := newVMExecutor(state, block, "casting")
	stateRoot, evictedTxs, transactions, receipts := executor.Execute()
	middleware.PerfLogger.Infof("fin execute txs. last: %v height: %v", utility.GetTime().Sub(timestamp), height)

	if !common.IsProposal020() || common.IsProposal023() {
		transactionHashes := make([]common.Hashes, len(transactions))
		block.Transactions = transactions
		for i, transaction := range transactions {
			hashes := common.Hashes{}
			hashes[0] = transaction.Hash
			hashes[1] = transaction.SubHash
			transactionHashes[i] = hashes

		}
		block.Header.Transactions = transactionHashes
		block.Header.TxTree = calcTxTree(block.Transactions)
	}

	block.Header.EvictedTxs = evictedTxs
	middleware.PerfLogger.Infof("fin calcTxTree. last: %v height: %v", utility.GetTime().Sub(timestamp), height)

	block.Header.StateTree = stateRoot
	block.Header.ReceiptTree = calcReceiptsTree(receipts)
	block.Header.Hash = block.Header.GenHash()
	middleware.PerfLogger.Infof("fin calcReceiptsTree. last: %v height: %v", utility.GetTime().Sub(timestamp), height)

	chain.verifiedBlocks.Add(block.Header.Hash, &castingBlock{state: state, receipts: receipts})
	if len(block.Transactions) != 0 {
		chain.verifiedBodyCache.Add(block.Header.Hash, block.Transactions)
	}

	logger.Debugf("Casting block %d,hash: %v,qn: %d,tx: %d,tx tree root: %v,prove value: %v,state tree root:%s",
		height, block.Header.Hash.String(), block.Header.TotalQN, len(block.Transactions), block.Header.TxTree.Hex(),
		consensusHelper.VRFProve2Value(block.Header.ProveValue), block.Header.StateTree.String())
}

func getRequestIdFromTransactions(transactions []*types.Transaction, lastOne map[string]uint64) map[string]uint64 {
	result := make(map[string]uint64)
	for key, value := range lastOne {
		result[key] = value
	}

	if nil != transactions && 0 != len(transactions) {
		maxRequestId := uint64(0)
		for _, tx := range transactions {
			if tx.RequestId > maxRequestId {
				maxRequestId = tx.RequestId
			}
		}

		if 0 != maxRequestId && maxRequestId > result["fixed"] {
			result["fixed"] = maxRequestId
		}
	}

	return result
}

func (chain *blockChain) GenerateBlock(bh types.BlockHeader) *types.Block {
	block := &types.Block{
		Header: &bh,
	}

	txs, missTxs, _ := chain.queryTxsByBlockHash(bh.Hash, bh.Transactions)

	if len(missTxs) != 0 {
		logger.Debugf("GenerateBlock can not get all txs,return nil block!")
		return nil
	}
	block.Transactions = txs
	return block
}

func (chain *blockChain) VerifyBlock(bh *types.BlockHeader) ([]common.Hashes, int8) {
	msg := "VerifyCastingBlock: " + strconv.FormatUint(bh.Height, 10) + " " + bh.Hash.String()
	middleware.LockBlockchain(msg)
	defer middleware.UnLockBlockchain(msg)

	return chain.verifyBlock(bh, nil, true)
}

func (chain *blockChain) Height() uint64 {
	if nil == chain.latestBlock {
		return math.MaxUint64
	}
	return chain.latestBlock.Height
}

func (chain *blockChain) TotalQN() uint64 {
	if nil == chain.latestBlock {
		return 0
	}
	return chain.latestBlock.TotalQN
}

func (chain *blockChain) AddBlockOnChain(b *types.Block) types.AddBlockResult {
	if validateCode, result := chain.consensusVerify(b); !result {
		return validateCode
	}
	middleware.LockBlockchain("AddBlockOnChain")
	defer middleware.UnLockBlockchain("AddBlockOnChain")
	return chain.addBlockOnChain(b)
}

func (chain *blockChain) QueryBlockByHash(hash common.Hash) *types.Block {
	return chain.queryBlockByHash(hash)
}

func (chain *blockChain) QueryBlock(height uint64) *types.Block {
	middleware.RLockBlockchain("QueryBlock")
	defer middleware.RUnLockBlockchain("QueryBlock")

	var b *types.Block
	bh := chain.QueryBlockHeaderByHeight(height, true)
	if nil == bh {
		return b
	}
	b = chain.queryBlockByHash(bh.Hash)
	return b
}

func (chain *blockChain) GetVerifyHash(height uint64) (common.Hash, error) {
	key := utility.UInt64ToByte(height)
	raw, err := chain.verifyHashDB.Get(key)
	logger.Debugf("Get verify hash.Height:%d,hash:%s", height, common.BytesToHash(raw).String())
	return common.BytesToHash(raw), err
}

func (chain *blockChain) TopBlock() *types.BlockHeader {
	result := chain.latestBlock
	return result
}

func (chain *blockChain) GetTransaction(txHash common.Hash) (*types.Transaction, error) {
	return chain.transactionPool.GetTransaction(txHash)
}

func (chain *blockChain) GetBalance(address common.Address) *big.Int {
	latestStateDB := middleware.AccountDBManagerInstance.GetLatestStateDB()
	if nil == latestStateDB {
		return nil
	}

	return latestStateDB.GetBalance(common.BytesToAddress(address.Bytes()))
}

func (chain *blockChain) Close() {
	chain.hashDB.Close()
	chain.heightDB.Close()
	chain.verifyHashDB.Close()
}

func (chain *blockChain) QueryBlockHeaderByHeight(height interface{}, cache bool) *types.BlockHeader {
	var key []byte
	switch height.(type) {
	case []byte:
		key = height.([]byte)
	default:
		if cache {
			h := height.(uint64)
			result, ok := chain.topBlocks.Get(h)
			if ok && nil != result {
				return result.(*types.BlockHeader)
			}
		}

		key = generateHeightKey(height.(uint64))
	}

	result, err := chain.heightDB.Get(key)
	if result != nil {
		var header *types.BlockHeader
		header, err = types.UnMarshalBlockHeader(result)
		if err != nil {
			return nil
		}

		return header
	} else {
		return nil
	}
}

func (chain *blockChain) remove(block *types.Block) bool {
	if nil == block {
		return true
	}
	hash := block.Header.Hash
	height := block.Header.Height
	logger.Debugf("remove hash:%s height:%d ", hash.Hex(), height)

	var receipts types.Receipts
	if common.IsFullNode() {
		receipts = chain.getReceipts(*block)
	}
	chain.markRemoveBlock(block)

	chain.hashDB.Delete(hash.Bytes())
	chain.heightDB.Delete(generateHeightKey(height))
	chain.verifyHashDB.Delete(utility.UInt64ToByte(height))

	chain.topBlocks.Remove(height)
	chain.verifiedBlocks.Remove(hash)

	preBlock := chain.queryBlockByHash(block.Header.PreHash)
	if preBlock == nil {
		logger.Errorf("Query nil block header by hash  while removing block! Hash:%s,height:%d, preHash :%s", hash.Hex(), height, block.Header.PreHash.Hex())
		return false
	}
	preHeader := preBlock.Header
	chain.latestBlock = preHeader

	preHeaderByte, _ := types.MarshalBlockHeader(preHeader)
	chain.heightDB.Put([]byte(latestBlockKey), preHeaderByte)

	chain.transactionPool.UnMarkExecuted(block)
	chain.eraseRemoveBlockMark()
	if chain.latestBlock != nil {
		common.SetBlockHeight(chain.latestBlock.Height)
	}

	if common.IsFullNode() {
		go chain.notifyRemovedLogs(receipts)
	}
	return true
}

func (chain *blockChain) HasBlockByHash(hash common.Hash) bool {
	result, err := chain.hashDB.Has(hash.Bytes())
	if err != nil {
		result = false
	}

	return result
}

func (chain *blockChain) GetBlockHash(height uint64) common.Hash {
	blockHeader := chain.QueryBlockHeaderByHeight(height, true)
	if blockHeader != nil {
		return blockHeader.Hash
	}
	return common.Hash{}
}

func (chain *blockChain) queryBlockByHash(hash common.Hash) *types.Block {
	result, err := chain.hashDB.Get(hash.Bytes())

	if result != nil {
		var block *types.Block
		block, err = types.UnMarshalBlock(result)
		if err != nil || &block == nil {
			return nil
		}
		return block
	} else {
		return nil
	}
}

func (chain *blockChain) queryBlockHeaderByHash(hash common.Hash) *types.BlockHeader {
	block := chain.queryBlockByHash(hash)
	if nil == block {
		return nil
	}
	return block.Header
}

func (chain *blockChain) queryTxsByBlockHash(blockHash common.Hash, txHashList []common.Hashes) ([]*types.Transaction, []common.Hashes, error) {
	if nil == txHashList || 0 == len(txHashList) {
		return nil, nil, service.ErrNil
	}

	verifiedBody, _ := chain.verifiedBodyCache.Get(blockHash)
	var verifiedTxs []*types.Transaction
	if nil != verifiedBody {
		verifiedTxs = verifiedBody.([]*types.Transaction)
	}

	txs := make([]*types.Transaction, 0)
	need := make([]common.Hashes, 0)
	var err error

	for _, hash := range txHashList {
		var tx *types.Transaction
		if verifiedTxs != nil {
			for _, verifiedTx := range verifiedTxs {
				if verifiedTx.Hash == hash[0] {
					tx = verifiedTx
					break
				}
			}
		}

		if tx == nil {
			tx, err = chain.transactionPool.GetTransaction(hash[0])
		}

		if tx == nil {
			need = append(need, hash)
		}
		txs = append(txs, tx)
	}
	return txs, need, err
}

func (chain *blockChain) versionValidate() bool {
	genesisHeader := chain.QueryBlockHeaderByHeight(uint64(0), true)
	if genesisHeader == nil {
		return false
	}
	version := genesisHeader.Nonce
	if version != ChainDataVersion {
		return false
	}
	return true
}

func (chain *blockChain) buildCache(size uint64) {
	var start uint64
	if chain.latestBlock.Height < size {
		start = 0
	} else {
		start = chain.latestBlock.Height - (size - 1)
	}

	for i := start; i < chain.latestBlock.Height; i++ {
		chain.topBlocks.Add(i, chain.QueryBlockHeaderByHeight(i, false))
	}
}

func generateHeightKey(height uint64) []byte {
	h := make([]byte, 8)
	binary.BigEndian.PutUint64(h, height)
	return h
}

func (chain *blockChain) getReceipts(b types.Block) types.Receipts {
	if value, exit := chain.verifiedBlocks.Get(b.Header.Hash); exit {
		bb := value.(*castingBlock)
		return bb.receipts
	}
	var result types.Receipts
	for _, tx := range b.Transactions {
		receipt := service.GetReceipt(tx.Hash)
		if receipt != nil {
			result = append(result, receipt)
		}
	}
	return result
}
