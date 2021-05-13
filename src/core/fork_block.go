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
	waitingGroup   bool
	header         uint64

	latestBlock *types.BlockHeader
	pending     *lane.Queue

	db     db.Database
	logger log.Logger
}

func newBlockChainFork(commonAncestor types.Block) *blockChainFork {
	fork := &blockChainFork{header: commonAncestor.Header.Height, latestBlock: commonAncestor.Header, logger: syncLogger}
	fork.enableRcvBlock = true
	fork.rcvLastBlock = false
	fork.waitingGroup = false

	fork.pending = lane.NewQueue()
	fork.db = refreshBlockForkDB(commonAncestor)
	fork.insertBlock(&commonAncestor)
	return fork
}

func (fork *blockChainFork) acceptBlock(coming *types.Block) error {
	if coming == nil || !fork.verifyOrder(coming) || !fork.verifyHash(coming) || !fork.verifyTxRoot(coming) {
		return verifyBlockErr
	}
	group := groupChainImpl.GetGroupById(coming.Header.GroupId)
	if group == nil {
		fork.logger.Debugf("Verify group not on group chain.Group id:%s", common.ToHex(coming.Header.GroupId))
		return verifyGroupNotOnChainErr
	}

	if !fork.verifyGroupSign(coming) {
		return verifyBlockErr
	}
	//todo
	//verifyResult, state := fork.verifyStateAndReceipt(coming)
	//if !verifyResult {
	//	return false
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
	err := fork.db.Delete(generateHeightKey(height))
	if err != nil {
		logger.Errorf("Fail to delete block, error:%s", err.Error())
		return false
	}
	return true
}

func (fork *blockChainFork) destroy() {
	for i := fork.header; i <= fork.latestBlock.Height; i++ {
		fork.deleteBlock(i)
	}
	fork.db.Delete([]byte(blockCommonAncestorHeightKey))
	fork.db.Delete([]byte(latestBlockHeightKey))
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
	fork.logger.Debugf("state root:%s", stateRoot.String())
	receiptsTree := calcReceiptsTree(receipts)
	if receiptsTree != coming.Header.ReceiptTree {
		fork.logger.Errorf("Receipt root error!coming:%s gen:%s", coming.Header.ReceiptTree.Hex(), receiptsTree.Hex())
		return false, state
	}
	return true, state
}

func (fork *blockChainFork) verifyGroupSign(coming *types.Block) bool {
	result, err := consensusHelper.VerifyBlockHeader(coming.Header)
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
	fork.logger.Debugf("commit state root:%s", root.String())

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
