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
	"bytes"
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/middleware"
	"com.tuntun.rangers/node/src/middleware/notify"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/storage/account"
	"com.tuntun.rangers/node/src/utility"
	"errors"
)

func (chain *blockChain) consensusVerify(b *types.Block) (types.AddBlockResult, bool) {
	if b == nil {
		return types.AddBlockFailed, false
	}

	if !chain.hasPreBlock(*b.Header) {
		logger.Warnf("coming block %s,%d has no pre on local chain.", b.Header.Hash.String(), b.Header.Height)
		chain.futureBlocks.Add(b.Header.PreHash, b)
		return types.NoPreOnChain, false
	}

	if chain.queryBlockHeaderByHash(b.Header.Hash) != nil {
		return types.BlockExisted, false
	}

	if check, err := consensusHelper.CheckProveRoot(b.Header); !check {
		logger.Errorf("checkProveRoot fail, err=%v", err.Error())
		return types.DependOnGroup, false
	}

	groupValidateResult, err := chain.validateGroupSig(b.Header)
	if !groupValidateResult {
		if err == common.ErrSelectGroupNil || err == common.ErrSelectGroupInequal {
			logger.Infof("Add block on chain failed: depend on group!")
		} else {
			logger.Errorf("Fail to validate group sig!Err:%s", err.Error())
		}
		return types.AddBlockFailed, false
	}
	return types.ValidateBlockOk, true
}

func (chain *blockChain) addBlockOnChain(coming *types.Block) types.AddBlockResult {
	topBlock := chain.latestBlock
	comingHeader := coming.Header

	logger.Debugf("coming block: hash: %v, preH: %v, height: %v,totalQn:%d, localTopHash: %v, localPreHash: %v, localHeight: %v, localTotalQn: %d", comingHeader.Hash.Hex(), comingHeader.PreHash.Hex(), comingHeader.Height, comingHeader.TotalQN, topBlock.Hash.Hex(), topBlock.PreHash.Hex(), topBlock.Height, topBlock.TotalQN)

	if comingHeader.Hash == topBlock.Hash || chain.HasBlockByHash(comingHeader.Hash) {
		return types.BlockExisted
	}

	if _, verifyResult := chain.verifyBlock(comingHeader, coming.Transactions, false); verifyResult != 0 {
		logger.Errorf("Fail to VerifyCastingBlock, reason code:%d \n", verifyResult)
		if verifyResult == 2 {
			logger.Warnf("coming block has no pre on local chain.Forking...")
		}
		return types.AddBlockFailed
	}

	if comingHeader.PreHash == topBlock.Hash {
		result, _ := chain.insertBlock(coming)
		return result
	}

	if comingHeader.TotalQN < topBlock.TotalQN {
		return types.BlockTotalQnLessThanLocal
	}

	commonAncestor := chain.queryBlockHeaderByHash(comingHeader.PreHash)
	if commonAncestor == nil {
		logger.Warnf("Block chain query nil block!Hash:%s", comingHeader.PreHash)
		return types.AddBlockFailed
	}

	if comingHeader.TotalQN > topBlock.TotalQN {
		logger.Warnf("coming qn great than local. Remove from common ancestor and add...coming block:hash=%v, preH=%v, height=%v,totalQn:%d. Local topHash=%v, topPreHash=%v, height=%v,totalQn:%d. commonAncestor hash:%s height:%d",
			comingHeader.Hash.Hex(), comingHeader.PreHash.Hex(), comingHeader.Height, comingHeader.TotalQN, topBlock.Hash.Hex(), topBlock.PreHash.Hex(), topBlock.Height, topBlock.TotalQN, commonAncestor.Hash.Hex(), commonAncestor.Height)
		chain.removeFromCommonAncestor(commonAncestor)
		return chain.addBlockOnChain(coming)
	}

	localNextBlock := chain.QueryBlockHeaderByHeight(commonAncestor.Height+1, true)
	if localNextBlock == nil {
		logger.Warnf("Block chain query nil block!Height:%s", commonAncestor.Height+1)
		return types.AddBlockFailed
	}
	if chainPvGreatThanRemote(localNextBlock, comingHeader) {
		return types.BlockTotalQnLessThanLocal
	}

	logger.Warnf("coming pv great to local. Remove from common ancestor and add...coming block:hash=%v, preH=%v, height=%v,totalQn:%d. Local topHash=%v, topPreHash=%v, height=%v,totalQn:%d. commonAncestor hash:%s height:%d",
		comingHeader.Hash.Hex(), comingHeader.PreHash.Hex(), comingHeader.Height, comingHeader.TotalQN, topBlock.Hash.Hex(), topBlock.PreHash.Hex(), topBlock.Height, topBlock.TotalQN, commonAncestor.Hash.Hex(), commonAncestor.Height)
	chain.removeFromCommonAncestor(commonAncestor)

	return chain.addBlockOnChain(coming)
}

func (chain *blockChain) executeTransaction(block *types.Block, setHash bool) (bool, *account.AccountDB, types.Receipts) {
	preBlock := chain.queryBlockHeaderByHash(block.Header.PreHash)
	if preBlock == nil {
		panic("Pre block nil !!")
	}
	preRoot := common.BytesToHash(preBlock.StateTree.Bytes())
	if len(block.Transactions) > 0 {
		logger.Debugf("NewAccountDB height:%d StateTree:%s preHash:%s preRoot:%s", block.Header.Height, block.Header.StateTree.Hex(), preBlock.Hash.Hex(), preRoot.Hex())
	}
	state, err := middleware.AccountDBManagerInstance.GetAccountDBByHash(preRoot)
	if err != nil {
		logger.Errorf("Fail to new statedb, error:%s", err)
		return false, state, nil
	}

	vmExecutor := newVMExecutor(state, block, "fullverify")
	stateRoot, evictedTxs, transactions, receipts := vmExecutor.Execute()
	if setHash {
		block.Header.StateTree = stateRoot
	} else if common.ToHex(stateRoot.Bytes()) != common.ToHex(block.Header.StateTree.Bytes()) {
		logger.Errorf("Fail to verify state tree, hash1:%x hash2:%x", stateRoot.Bytes(), block.Header.StateTree.Bytes())
		return false, state, receipts
	}

	receiptsTree := calcReceiptsTree(receipts)
	if setHash {
		block.Header.ReceiptTree = receiptsTree
	} else if 0 != bytes.Compare(receiptsTree.Bytes(), block.Header.ReceiptTree.Bytes()) {
		logger.Errorf("fail to verify receipt, hash1:%s hash2:%s", receiptsTree.String(), block.Header.ReceiptTree.String())
		return false, state, receipts
	}

	if setHash {
		block.Header.EvictedTxs = evictedTxs
		if !common.IsProposal020() {
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

		block.Header.Hash = block.Header.GenHash()
	}

	chain.verifiedBlocks.Add(block.Header.Hash, &castingBlock{state: state, receipts: receipts})
	return true, state, receipts
}

func (chain *blockChain) insertBlock(remoteBlock *types.Block) (types.AddBlockResult, []byte) {
	logger.Debugf("Insert block hash:%s,height:%d,evicted tx len:%d", remoteBlock.Header.Hash.Hex(), remoteBlock.Header.Height, len(remoteBlock.Header.EvictedTxs))
	blockByte, err := types.MarshalBlock(remoteBlock)
	if err != nil {
		logger.Errorf("Fail to json Marshal, error:%s", err.Error())
		return types.AddBlockFailed, nil
	}

	chain.markAddBlock(blockByte)
	if !chain.saveBlockByHash(remoteBlock.Header.Hash, blockByte) {
		return types.AddBlockFailed, nil
	}

	headerByte, err := types.MarshalBlockHeader(remoteBlock.Header)
	if err != nil {
		logger.Errorf("Fail to json Marshal header, error:%s", err.Error())
		return types.AddBlockFailed, nil
	}
	if !chain.saveBlockByHeight(remoteBlock.Header.Height, headerByte) {
		return types.AddBlockFailed, nil
	}

	saveStateResult, accountDB, receipts := chain.saveBlockState(remoteBlock)

	if !saveStateResult {
		return types.AddBlockFailed, nil
	}

	chain.updateVerifyHash(remoteBlock)

	chain.updateTxPool(remoteBlock, receipts)
	chain.topBlocks.Add(remoteBlock.Header.Height, remoteBlock.Header)

	if !chain.updateLastBlock(accountDB, remoteBlock, headerByte) {
		return types.AddBlockFailed, headerByte
	}
	if chain.latestBlock != nil {
		common.SetBlockHeight(chain.latestBlock.Height)
	}
	chain.eraseAddBlockMark()
	chain.successOnChainCallBack(remoteBlock)

	return types.AddBlockSucc, headerByte
}

func (chain *blockChain) saveBlockByHash(hash common.Hash, blockByte []byte) bool {
	err := chain.hashDB.Put(hash.Bytes(), blockByte)
	if err != nil {
		logger.Errorf("Fail to put block hash %s  error:%s", hash.String(), err.Error())
		return false
	}
	return true
}

func (chain *blockChain) saveBlockByHeight(height uint64, headerByte []byte) bool {
	err := chain.heightDB.Put(generateHeightKey(height), headerByte)
	if err != nil {
		logger.Errorf("Fail to put block height:%d  error:%s", height, err.Error())
		return false
	}
	return true
}

func (chain *blockChain) saveBlockState(b *types.Block) (bool, *account.AccountDB, types.Receipts) {
	var state *account.AccountDB
	var receipts types.Receipts
	if value, exit := chain.verifiedBlocks.Get(b.Header.Hash); exit {
		bb := value.(*castingBlock)
		state = bb.state
		receipts = bb.receipts
		logger.Errorf("get verifiedBlock from cache, %s", b.Header.Hash.String())
	} else {
		var executeTxResult bool
		logger.Errorf("fail to get verifiedBlock from cache, %s", b.Header.Hash.String())

		executeTxResult, state, receipts = chain.executeTransaction(b, false)
		if !executeTxResult {
			logger.Errorf("fail to execute txs!")
			return false, state, receipts
		}
	}

	root, err := state.Commit(true)
	if err != nil {
		logger.Errorf("State commit error:%s", err.Error())
		return false, state, receipts
	}

	trieDB := middleware.AccountDBManagerInstance.GetTrieDB()
	err = trieDB.Commit(root, false)
	if err != nil {
		logger.Errorf("Trie commit error:%s", err.Error())
		return false, state, receipts
	}
	return true, state, receipts
}

func (chain *blockChain) updateLastBlock(state *account.AccountDB, block *types.Block, headerJson []byte) bool {
	header := block.Header
	err := chain.heightDB.Put([]byte(latestBlockKey), headerJson)
	if err != nil {
		logger.Errorf("Fail to put %s, error:%s", latestBlockKey, err.Error())
		return false
	}

	chain.latestBlock = header
	chain.requestIds = header.RequestIds

	middleware.AccountDBManagerInstance.SetLatestStateDB(state, block.Header.RequestIds, block.Header.Height)
	logger.Debugf("Update latestStateDB:%s height:%d", header.StateTree.Hex(), header.Height)

	return true
}

func (chain *blockChain) updateVerifyHash(block *types.Block) {
	verifyHash := consensusHelper.VerifyHash(block)
	chain.verifyHashDB.Put(utility.UInt64ToByte(block.Header.Height), verifyHash.Bytes())
	logger.Debugf("Update verify hash.Height:%d,verifyHash:%s", utility.UInt64ToByte(block.Header.Height), verifyHash.String())
}

func (chain *blockChain) updateTxPool(block *types.Block, receipts types.Receipts) {
	if common.IsFullNode() {
		go chain.notifyLogs(block.Header.Hash, receipts)
		go chain.notifyBlockHeader(block.Header)
	}
	chain.transactionPool.MarkExecuted(block.Header, receipts, block.Transactions, block.Header.EvictedTxs)
}

func (chain *blockChain) successOnChainCallBack(remoteBlock *types.Block) {
	logger.Infof("ON chain succ! height: %d,hash: %s", remoteBlock.Header.Height, remoteBlock.Header.Hash.Hex())
	notify.BUS.Publish(notify.BlockAddSucc, &notify.BlockOnChainSuccMessage{Block: *remoteBlock})
	if value, _ := chain.futureBlocks.Get(remoteBlock.Header.Hash); value != nil {
		block := value.(*types.Block)
		logger.Debugf("Get block from future blocks,hash:%s,height:%d", block.Header.Hash.String(), block.Header.Height)

		chain.addBlockOnChain(block)
		return
	}
	if SyncProcessor != nil {
		go SyncProcessor.broadcastChainInfo(chain.latestBlock)
	}
}

func (chain *blockChain) validateGroupSig(bh *types.BlockHeader) (bool, error) {
	if chain.Height() == 0 {
		return true, nil
	}
	pre := chain.queryBlockByHash(bh.PreHash)
	if pre == nil {
		return false, errors.New("has no pre")
	}
	result, err := consensusHelper.VerifyNewBlock(bh, pre.Header)
	if err != nil {
		logger.Errorf("validateGroupSig error:%s", err.Error())
		return false, err
	}
	return result, err
}

func (chain *blockChain) removeFromCommonAncestor(commonAncestor *types.BlockHeader) {
	logger.Debugf("removeFromCommonAncestor hash:%s height:%d latestheight:%d", commonAncestor.Hash.Hex(), commonAncestor.Height, chain.latestBlock.Height)
	for height := chain.latestBlock.Height; height > commonAncestor.Height; height-- {
		header := chain.QueryBlockHeaderByHeight(height, true)
		if header == nil {
			logger.Debugf("removeFromCommonAncestor nil height:%d", height)
			continue
		}
		block := chain.queryBlockByHash(header.Hash)
		if block == nil {
			continue
		}
		chain.remove(block)
		logger.Debugf("Remove local block hash:%s, height %d", header.Hash.String(), header.Height)
	}
}

func dumpTxs(txs []*types.Transaction, blockHeight uint64) {
	if txs == nil || len(txs) == 0 {
		return
	}

	txLogger.Tracef("Tx on chain dump! Block height:%d", blockHeight)
	for _, tx := range txs {
		txLogger.Tracef("Tx info;%s", tx.ToTxJson().ToString())
	}
}

func (chain *blockChain) ExecuteTransaction(block *types.Block) (bool, *account.AccountDB, types.Receipts) {
	return chain.executeTransaction(block, false)
}
