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
	"bytes"
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/db"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"errors"
	"github.com/oleiade/lane"
)

var verifyGroupNotOnChainErr = errors.New("Verify group not on group chain")
var verifyBlockErr = errors.New("verify block error")

type blockChainFork struct {
	enableRcvBlock bool
	rcvLastBlock   bool
	header         uint64

	currentWaitingGroupId []byte
	lastWaitingGroupId    []byte
	latestBlock           *types.BlockHeader
	pending               *lane.Queue

	db     db.Database
	lock   middleware.Loglock
	logger log.Logger
}

func newBlockChainFork(commonAncestor types.Block) *blockChainFork {
	fork := &blockChainFork{header: commonAncestor.Header.Height, latestBlock: commonAncestor.Header, logger: syncLogger}
	fork.enableRcvBlock = true
	fork.rcvLastBlock = false

	fork.pending = lane.NewQueue()
	fork.lock = middleware.NewLoglock("blockChainFork")
	fork.db = refreshBlockForkDB(commonAncestor)
	fork.insertBlock(&commonAncestor)
	return fork
}

func (fork *blockChainFork) isWaiting() bool {
	fork.lock.RLock("block chain fork is Waiting")
	defer fork.lock.RUnlock("block chain fork is Waiting")

	if fork.lastWaitingGroupId == nil || len(fork.lastWaitingGroupId) == 0 {
		return false
	}
	if fork.currentWaitingGroupId == nil || len(fork.currentWaitingGroupId) == 0 {
		return false
	}
	return bytes.Equal(fork.lastWaitingGroupId, fork.currentWaitingGroupId)
}

func (fork *blockChainFork) rcv(block *types.Block, isLastBlock bool) (needMore bool) {
	fork.lock.Lock("block chain fork rcv")
	defer fork.lock.Unlock("block chain fork rcv")

	if !fork.enableRcvBlock {
		return false
	}
	fork.pending.Enqueue(block)
	fork.rcvLastBlock = isLastBlock
	if isLastBlock || fork.pending.Size() >= syncedBlockCount {
		fork.enableRcvBlock = false
		return false
	}
	return true
}

func (fork *blockChainFork) triggerOnFork(groupFork *groupChainFork) (err error, rcvLastBlock bool, tail *types.Block) {
	fork.lock.Lock("block chain fork triggerOnFork")
	defer fork.lock.Unlock("block chain fork triggerOnFork")

	fork.logger.Debugf("Trigger block on fork..")
	var block *types.Block
	for !fork.pending.Empty() {
		block = fork.pending.Head().(*types.Block)
		err = fork.addBlockOnFork(block, groupFork)
		if err != nil {
			fork.logger.Debugf("Block on fork failed!%s,%d-%d", block.Header.Hash.String(), block.Header.Height, block.Header.TotalQN)
			break
		}
		fork.logger.Debugf("Block on fork success!%s,%d-%d", block.Header.Hash.String(), block.Header.Height, block.Header.TotalQN)
		block = fork.pending.Pop().(*types.Block)
		fork.lastWaitingGroupId = fork.currentWaitingGroupId
		fork.currentWaitingGroupId = nil
	}

	if err == verifyGroupNotOnChainErr {
		fork.lastWaitingGroupId = fork.currentWaitingGroupId
		fork.currentWaitingGroupId = block.Header.GroupId
		fork.logger.Debugf("Trigger block on fork paused. waiting group %s", common.ToHex(fork.currentWaitingGroupId))
	}

	if err != nil {
		return err, fork.rcvLastBlock, nil
	}

	if !fork.rcvLastBlock {
		fork.enableRcvBlock = true
	}

	if fork.pending.Empty() {
		return err, fork.rcvLastBlock, block
	}
	return err, fork.rcvLastBlock, nil
}

func (blockFork *blockChainFork) triggerOnChain(chain *blockChain, groupChain *groupChain, groupFork *groupChainFork) bool {
	blockFork.lock.Lock("block chain fork triggerOnChain")
	defer blockFork.lock.Unlock("block chain fork triggerOnFork")

	localTopHeader := chain.latestBlock
	syncLogger.Debugf("Trigger block on chain...Local chain:%d-%d,fork:%d-%d", localTopHeader.Height, localTopHeader.TotalQN, blockFork.latestBlock.Height, blockFork.latestBlock.TotalQN)
	if blockFork.latestBlock.TotalQN < localTopHeader.TotalQN {
		return false
	}

	var commonAncestor *types.BlockHeader
	for height := blockFork.header; height <= blockFork.latestBlock.Height; height++ {
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
		return false
	}
	syncLogger.Debugf("[TriggerBlockOnChain]. common ancestor:%d", commonAncestor.Height)
	if blockFork.latestBlock.TotalQN == localTopHeader.TotalQN && chain.nextPvGreatThanFork(commonAncestor, *blockFork) {
		return false
	}

	chain.removeFromCommonAncestor(commonAncestor)
	for height := blockFork.header + 1; height <= blockFork.latestBlock.Height; {
		forkBlock := blockFork.getBlock(height)
		if forkBlock == nil {
			return false
		}
		success, dependOnGroup := tryAddBlockOnChain(chain, forkBlock)
		if success {
			height++
			continue
		} else if !dependOnGroup {
			return false
		}
		if groupFork != nil {
			groupFork.triggerOnChain(groupChain)
		}

		success, dependOnGroup = tryAddBlockOnChain(chain, forkBlock)
		if !success {
			return false
		}
	}
	if groupFork != nil {
		groupFork.triggerOnChain(groupChain)
	}
	return true
}

func (fork *blockChainFork) destroy() {
	fork.lock.Lock("block chain fork destroy")
	defer fork.lock.Unlock("block chain fork destroy")

	for i := fork.header; i <= fork.latestBlock.Height; i++ {
		fork.deleteBlock(i)
	}
	fork.db.Delete([]byte(blockCommonAncestorHeightKey))
	fork.db.Delete([]byte(latestBlockHeightKey))
}

func (fork *blockChainFork) getBlockByHash(hash common.Hash) *types.Block {
	fork.lock.RLock("block chain fork getBlockByHash")
	defer fork.lock.RUnlock("block chain fork getBlockByHash")

	bytes, _ := fork.db.Get(hash.Bytes())
	block, _ := types.UnMarshalBlock(bytes)
	//if err != nil {
	//	logger.Errorf("Fail to umMarshal block, error:%s", err.Error())
	//}
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
	//verifyResult, state := fork.verifyStateAndReceipt(coming)
	//if !verifyResult {
	//	return verifyBlockErr
	//}
	//fork.saveState(state)

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
	return nil
}

func (fork *blockChainFork) getBlock(height uint64) *types.Block {
	bytes, _ := fork.db.Get(generateHeightKey(height))
	block, err := types.UnMarshalBlock(bytes)
	if err != nil {
		logger.Errorf("Fail to umMarshal block, error:%s", err.Error())
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
	state, err := service.AccountDBManagerInstance.GetAccountDBByHash(preBlock.Header.StateTree)
	if err != nil {
		fork.logger.Errorf("Fail to new statedb, error:%s", err)
		return false, state
	}
	vmExecutor := newVMExecutor(state, coming, "fullverify")
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

	trieDB := service.AccountDBManagerInstance.GetTrieDB()
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
	for i := start; i <= end; i++ {
		db.Delete(generateHeightKey(i))
	}

	db.Put([]byte(blockCommonAncestorHeightKey), utility.UInt64ToByte(commonAncestor.Header.Height))
	db.Put([]byte(latestBlockHeightKey), utility.UInt64ToByte(commonAncestor.Header.Height))
	return db
}

func tryAddBlockOnChain(chain *blockChain, forkBlock *types.Block) (success bool, dependOnGroup bool) {
	validateCode, consensusVerifyResult := chain.consensusVerify(forkBlock)
	if !consensusVerifyResult {
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
