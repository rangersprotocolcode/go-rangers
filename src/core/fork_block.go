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
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"bytes"
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/db"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"errors"
	"github.com/oleiade/lane"
)

var (
	verifyGroupNotOnChainErr = errors.New("Verify group not on group chain")
	verifyBlockErr           = errors.New("verify block error")
)

type blockChainFork struct {
	rcvLastBlock bool
	header       uint64

	current     uint64
	latestBlock *types.BlockHeader
	pending     *lane.Queue

	db     db.Database
	logger log.Logger
}

func newBlockChainFork(commonAncestor types.Block) *blockChainFork {
	fork := &blockChainFork{header: commonAncestor.Header.Height, current: commonAncestor.Header.Height, latestBlock: commonAncestor.Header, logger: syncLogger}
	fork.rcvLastBlock = false

	fork.pending = lane.NewQueue()
	fork.db = refreshBlockForkDB(commonAncestor)
	fork.insertBlock(&commonAncestor)
	return fork
}

func (fork *blockChainFork) rcv(block *types.Block, isLastBlock bool) (needMore bool) {
	if block != nil {
		fork.pending.Enqueue(block)
	}
	fork.rcvLastBlock = isLastBlock
	if isLastBlock || fork.pending.Size() >= syncedBlockCount {
		return false
	}
	return true
}

func (fork *blockChainFork) triggerOnFork(groupFork *groupChainFork) (err error, current *types.Block) {
	fork.logger.Debugf("Trigger block on fork..")
	for !fork.pending.Empty() {
		current = fork.pending.Head().(*types.Block)
		err = fork.addBlockOnFork(current, groupFork)
		if err != nil {
			fork.logger.Debugf("Block on fork failed!%s,%d-%d", current.Header.Hash.String(), current.Header.Height, current.Header.TotalQN)
			break
		}
		fork.logger.Debugf("Block on fork success!%s,%d-%d", current.Header.Hash.String(), current.Header.Height, current.Header.TotalQN)
		fork.pending.Pop()
	}

	if err == verifyGroupNotOnChainErr || err == common.ErrSelectGroupInequal {
		fork.logger.Debugf("Trigger block on fork paused. waiting group..")
	}
	return
}

func (blockFork *blockChainFork) triggerOnChain(chain *blockChain) bool {
	middleware.LockBlockchain("block chain fork triggerOnChain")
	defer middleware.UnLockBlockchain("block chain fork triggerOnFork")

	localTopHeader := chain.latestBlock
	syncLogger.Debugf("Trigger block on chain...Local chain:%d-%d,fork:%d-%d", localTopHeader.Height, localTopHeader.TotalQN, blockFork.latestBlock.Height, blockFork.latestBlock.TotalQN)
	if blockFork.latestBlock.TotalQN < localTopHeader.TotalQN {
		return true
	}

	var commonAncestor *types.BlockHeader
	for height := blockFork.current; height <= blockFork.latestBlock.Height; height++ {
		forkBlock := blockFork.getBlock(height)
		chainBlockHeader := chain.QueryBlockHeaderByHeight(height, true)
		if forkBlock == nil || chainBlockHeader == nil {
			break
		}
		if chainBlockHeader.Hash != forkBlock.Header.Hash {
			break
		}
		commonAncestor = forkBlock.Header
	}

	if commonAncestor == nil {
		syncLogger.Debugf("[TriggerBlockOnChain]common ancestor is nil.")
		return true
	}
	syncLogger.Debugf("[TriggerBlockOnChain]. common ancestor:%d", commonAncestor.Height)
	if blockFork.latestBlock.TotalQN == localTopHeader.TotalQN && chain.nextPvGreatThanFork(commonAncestor, *blockFork) {
		return true
	}

	if blockFork.current == blockFork.header {
		chain.removeFromCommonAncestor(commonAncestor)
		blockFork.current++
	}
	for blockFork.current <= blockFork.latestBlock.Height {
		forkBlock := blockFork.getBlock(blockFork.current)
		if forkBlock == nil {
			blockFork.logger.Debugf("block fork get nil block.height:%d", blockFork.current)
			return false
		}
		success, _ := tryAddBlockOnChain(chain, forkBlock)
		if success {
			blockFork.current++
			continue
		} else {
			return false
		}
	}
	return true
}

func (fork *blockChainFork) destroy() {
	for i := fork.header; i <= fork.latestBlock.Height; i++ {
		fork.logger.Debugf("[destroy]fork delete block %d", i)
		fork.deleteBlock(i)
	}
	fork.db.Delete([]byte(blockCommonAncestorHeightKey))
	fork.db.Delete([]byte(latestBlockHeightKey))
}

func (fork *blockChainFork) getBlockByHash(hash common.Hash) *types.Block {
	bytes, _ := fork.db.Get(hash.Bytes())
	if bytes == nil || len(bytes) == 0 {
		return nil
	}
	block, err := types.UnMarshalBlock(bytes)
	if err != nil {
		syncLogger.Errorf("Fail to umMarshal block, error:%s", err.Error())
	}
	return block
}

func (fork *blockChainFork) addBlockOnFork(coming *types.Block, groupFork *groupChainFork) error {
	if coming == nil || !fork.verifyOrder(coming) || !fork.verifyHash(coming) || !fork.verifyTxRoot(coming) {
		return verifyBlockErr
	}
	var group *types.Group
	group = groupChainImpl.GetGroupById(coming.Header.GroupId)
	if group == nil && groupFork != nil {
		group = groupFork.getGroupById(coming.Header.GroupId)
	}
	if group == nil {
		fork.logger.Debugf("Verify group not on group chain.Group id:%s", common.ToHex(coming.Header.GroupId))
		return verifyGroupNotOnChainErr
	}

	if !fork.verifyGroupSign(coming, group.PubKey) {
		return verifyBlockErr
	}

	if result, err := fork.verifyMemberLegal(coming); !result {
		fork.logger.Debugf("Block cast or verify member illegal.error:%s", err.Error())
		if err != common.ErrSelectGroupInequal {
			return verifyBlockErr
		}
		return err
	}
	verifyResult, state := fork.verifyStateAndReceipt(coming)
	if !verifyResult {
		return verifyBlockErr
	}
	fork.saveState(state)

	fork.insertBlock(coming)
	fork.latestBlock = coming.Header
	return nil
}

func (fork *blockChainFork) insertBlock(block *types.Block) error {
	blockByte, err := types.MarshalBlock(block)
	if err != nil {
		fork.logger.Errorf("Fail to marshal block, error:%s", err.Error())
		return err
	}
	err = fork.db.Put(generateHeightKey(block.Header.Height), blockByte)
	if err != nil {
		fork.logger.Errorf("Fail to insert db, error:%s", err.Error())
		return err
	}

	err = fork.db.Put(block.Header.Hash.Bytes(), blockByte)
	if err != nil {
		fork.logger.Errorf("Fail to insert db, error:%s", err.Error())
		return err
	}
	err = fork.db.Put([]byte(latestBlockHeightKey), utility.UInt64ToByte(block.Header.Height))
	if err != nil {
		fork.logger.Errorf("Fail to insert db, error:%s", err.Error())
		return err
	}
	fork.logger.Debugf("set latestBlockHeightKey:%d", block.Header.Height)
	return nil
}

func (fork *blockChainFork) getBlock(height uint64) *types.Block {
	bytes, _ := fork.db.Get(generateHeightKey(height))
	if bytes == nil || len(bytes) == 0 {
		return nil
	}
	block, err := types.UnMarshalBlock(bytes)
	if err != nil {
		syncLogger.Errorf("Fail to umMarshal block, error:%s", err.Error())
	}
	return block
}

func (fork *blockChainFork) deleteBlock(height uint64) bool {
	block := fork.getBlock(height)
	if block != nil {
		err := fork.db.Delete(block.Header.Hash.Bytes())
		if err != nil {
			logger.Errorf("Fail to delete block, error:%s", err.Error())
			return false
		}
	}

	err := fork.db.Delete(generateHeightKey(height))
	if err != nil {
		logger.Errorf("Fail to delete block, error:%s", err.Error())
		return false
	}
	return true
}

func (fork *blockChainFork) verifyOrder(coming *types.Block) bool {
	if coming.Header.PreHash != fork.latestBlock.Hash {
		fork.logger.Debugf("Order error! coming pre:%s,fork latest:%s", coming.Header.PreHash.Hex(), fork.latestBlock.Hash.Hex())
		return false
	}
	return true
}

func (fork *blockChainFork) verifyHash(coming *types.Block) bool {
	genHash := coming.Header.GenHash()
	if coming.Header.Hash != genHash {
		fork.logger.Debugf("Hash error! coming hash:%s,gen:%s", coming.Header.Hash.Hex(), genHash.Hex())
		return false
	}
	return true
}

func (fork *blockChainFork) verifyTxRoot(coming *types.Block) bool {
	txTree := calcTxTree(coming.Transactions)
	if !bytes.Equal(txTree.Bytes(), coming.Header.TxTree.Bytes()) {
		fork.logger.Errorf("Tx root error! coming:%s gen:%s", coming.Header.TxTree.Bytes(), txTree.Hex())
		return false
	}
	return true
}

func (fork *blockChainFork) verifyStateAndReceipt(coming *types.Block) (bool, *account.AccountDB) {
	var height uint64 = 0
	if coming.Header.Height > 1 {
		height = coming.Header.Height - 1
	}
	preBlock := fork.getBlock(height)
	if preBlock == nil {
		fork.logger.Errorf("Pre block nil !")
		return false, nil
	}
	fork.logger.Debugf("pre state root:%s", preBlock.Header.StateTree.String())
	state, err := middleware.AccountDBManagerInstance.GetAccountDBByHash(preBlock.Header.StateTree)
	if err != nil {
		fork.logger.Errorf("Fail to new statedb, error:%s", err)
		return false, state
	}
	vmExecutor := newVMExecutor(state, coming, "fork")
	stateRoot, _, _, receipts := vmExecutor.Execute()

	if stateRoot != coming.Header.StateTree {
		fork.logger.Errorf("State root error!coming:%s gen:%s", coming.Header.StateTree.Hex(), stateRoot.Hex())
		return false, state
	}
	fork.logger.Debugf("state root correct:%s", stateRoot.Hex())
	receiptsTree := calcReceiptsTree(receipts)
	if receiptsTree != coming.Header.ReceiptTree {
		fork.logger.Errorf("Receipt root error!coming:%s gen:%s", coming.Header.ReceiptTree.Hex(), receiptsTree.Hex())
		return false, state
	}
	return true, state
}

func (fork *blockChainFork) verifyGroupSign(coming *types.Block, groupPubkey []byte) bool {
	result, err := consensusHelper.VerifyGroupSign(groupPubkey, coming.Header.Hash, coming.Header.Signature)
	if err != nil {
		fork.logger.Errorf("Verify group sign error:%s", err.Error())
	}
	return result
}

func (fork *blockChainFork) verifyMemberLegal(coming *types.Block) (bool, error) {
	preBlock := fork.getBlockByHash(coming.Header.PreHash)
	if preBlock == nil {
		fork.logger.Debugf("Fork get pre block nil! coming pre:%s", coming.Header.PreHash.Hex())
		return false, verifyBlockErr
	}

	if fork.getBlockByHash(coming.Header.Hash) != nil {
		fork.logger.Debugf("Coming block existed on the fork! coming hash:%s", coming.Header.Hash.Hex())
		return false, verifyBlockErr
	}
	return consensusHelper.VerifyMemberInfo(coming.Header, preBlock.Header)
}

func (fork *blockChainFork) saveState(state *account.AccountDB) error {
	if state == nil {
		return nil
	}
	root, err := state.Commit(true)
	if err != nil {
		fork.logger.Errorf("State commit error:%s", err.Error())
		return err
	}
	fork.logger.Debugf("commit state root:%s", root.Hex())

	trieDB := middleware.AccountDBManagerInstance.GetTrieDB()
	err = trieDB.Commit(root, false)
	if err != nil {
		fork.logger.Errorf("Trie commit error:%s", err.Error())
		return err
	}
	return nil
}

func refreshBlockForkDB(commonAncestor types.Block) db.Database {
	db, _ := db.NewDatabase(blockForkDBPrefix)

	startBytes, _ := db.Get([]byte(blockCommonAncestorHeightKey))
	start := utility.ByteToUInt64(startBytes)
	endBytes, _ := db.Get([]byte(latestBlockHeightKey))
	end := utility.ByteToUInt64(endBytes)
	syncLogger.Debugf("refreshBlockForkDB start:%d,end:%d", start, end)
	for i := start; i <= end+1; i++ {
		bytes, _ := db.Get(generateHeightKey(i))
		if len(bytes) > 0 {
			block, err := types.UnMarshalBlock(bytes)
			if err == nil && block != nil {
				db.Delete(block.Header.Hash.Bytes())
			}
		}
		db.Delete(generateHeightKey(i))
	}

	db.Put([]byte(blockCommonAncestorHeightKey), utility.UInt64ToByte(commonAncestor.Header.Height))
	db.Put([]byte(latestBlockHeightKey), utility.UInt64ToByte(commonAncestor.Header.Height))
	return db
}

func tryAddBlockOnChain(chain *blockChain, forkBlock *types.Block) (success bool, dependOnGroup bool) {
	validateCode, consensusVerifyResult := chain.consensusVerify(forkBlock)
	if !consensusVerifyResult {
		syncLogger.Debugf("[TriggerBlockOnChain]block verify error.height:%d,code %d", forkBlock.Header.Height, validateCode)
		if validateCode == types.DependOnGroup {
			return false, true
		} else {
			return false, false
		}
	}
	var result types.AddBlockResult
	result = blockChainImpl.addBlockOnChain(forkBlock)
	if result == types.AddBlockSucc {
		syncLogger.Debugf("add block on chain success.%s,%d-%d", forkBlock.Header.Hash.String(), forkBlock.Header.Height, forkBlock.Header.TotalQN)
		return true, false
	}
	syncLogger.Debugf("add block on chain failed.%s,%d-%d", forkBlock.Header.Hash.String(), forkBlock.Header.Height, forkBlock.Header.TotalQN)
	return false, false
}

func (p *syncProcessor) GetBlockHash(height uint64) common.Hash {
	header := p.GetBlockHeader(height)
	if header != nil {
		return header.Hash
	}
	return common.Hash{}
}
