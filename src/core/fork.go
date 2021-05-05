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
)

type fork struct {
	header      uint64
	sourceMiner string
	latestBlock *types.BlockHeader

	db     db.Database
	logger log.Logger
}

func newFork(commonAncestor types.Block, sourceMiner string, logger log.Logger) *fork {
	db, err := db.NewDatabase(forkDBPrefix)
	if err != nil {
		blockSyncLogger.Debugf("Init block chain error! Error:%s", err.Error())
	}
	fork := &fork{header: commonAncestor.Header.Height, latestBlock: commonAncestor.Header, sourceMiner: sourceMiner, db: db, logger: logger}
	fork.insertBlock(commonAncestor)

	err = fork.db.Put([]byte(commonAncestorHeightKey), utility.UInt64ToByte(commonAncestor.Header.Height))
	if err != nil {
		blockSyncLogger.Debugf("Fork init put error:%s", err.Error())
	}
	err = fork.db.Put([]byte(lastestHeightKey), utility.UInt64ToByte(commonAncestor.Header.Height))
	if err != nil {
		blockSyncLogger.Debugf("Fork init put error:%s", err.Error())
	}
	return fork
}

func (fork *fork) acceptBlock(coming types.Block, sourceMiner string) bool {
	if fork.sourceMiner != sourceMiner {
		return false
	}

	if !fork.verifyOrder(coming) || !fork.verifyHash(coming) || !fork.verifyTxRoot(coming) || !fork.verifyGroupSign(coming) {
		return false
	}

	verifyResult, state := fork.verifyStateAndReceipt(coming)
	if !verifyResult {
		return false
	}
	fork.saveState(state)
	fork.insertBlock(coming)
	fork.latestBlock = coming.Header
	return true
}

func destoryFork(fork *fork) {
	if fork == nil {
		return
	}
	for i := fork.header; i <= fork.latestBlock.Height; i++ {
		fork.deleteBlock(i)
	}
	fork.db.Delete([]byte(commonAncestorHeightKey))
	fork.db.Delete([]byte(lastestHeightKey))
	fork = nil
}

func (fork *fork) verifyOrder(coming types.Block) bool {
	if coming.Header.PreHash != fork.latestBlock.Hash {
		fork.logger.Debugf("Order error! coming pre:%s,fork lastest:%s", coming.Header.PreHash.Hex(), fork.latestBlock.Hash.Hex())
		return false
	}
	return true
}

func (fork *fork) verifyHash(coming types.Block) bool {
	genHash := coming.Header.GenHash()
	if coming.Header.Hash != genHash {
		fork.logger.Debugf("Hash error! coming hash:%s,gen:%s", coming.Header.Hash.Hex(), genHash.Hex())
		return false
	}
	return true
}

func (fork *fork) verifyTxRoot(coming types.Block) bool {
	txTree := calcTxTree(coming.Transactions)
	if !bytes.Equal(txTree.Bytes(), coming.Header.TxTree.Bytes()) {
		fork.logger.Errorf("Tx root error! coming:%s gen:%s", coming.Header.TxTree.Bytes(), txTree.Hex())
		return false
	}
	return true
}

func (fork *fork) verifyStateAndReceipt(coming types.Block) (bool, *account.AccountDB) {
	//todo 这里会溢出嘛？
	preBlock := fork.getBlock(coming.Header.Height - 1)
	if preBlock == nil {
		fork.logger.Errorf("Pre block nil !")
		return false, nil
	}
	state, err := service.AccountDBManagerInstance.GetAccountDBByHash(preBlock.Header.StateTree)
	if err != nil {
		fork.logger.Errorf("Fail to new statedb, error:%s", err)
		return false, state
	}
	vmExecutor := newVMExecutor(state, &coming, "fork")
	stateRoot, _, _, receipts := vmExecutor.Execute()

	if stateRoot != coming.Header.StateTree {
		fork.logger.Errorf("State root error!coming:%s gen:%s", coming.Header.StateTree.Hex(), stateRoot.Hex())
		return false, state
	}
	receiptsTree := calcReceiptsTree(receipts)
	if receiptsTree != coming.Header.ReceiptTree {
		fork.logger.Errorf("Receipt root error!coming:%s gen:%s", coming.Header.ReceiptTree.Hex(), receiptsTree.Hex())
		return false, state
	}
	return true, state
}

func (fork *fork) verifyGroupSign(coming types.Block) bool {
	group := groupChainImpl.GetGroupById(coming.Header.GroupId)
	if group == nil {
		fork.logger.Debugf("Local group is nil.Id:%s", common.ToHex(coming.Header.GroupId))
		return false
	}
	result, _ := consensusHelper.VerifyBlockHeader(coming.Header)
	return result
}

func (fork *fork) saveState(state *account.AccountDB) error {
	if state == nil {
		return nil
	}
	root, err := state.Commit(true)
	if err != nil {
		fork.logger.Errorf("State commit error:%s", err.Error())
		return err
	}

	trieDB := service.AccountDBManagerInstance.GetTrieDB()
	err = trieDB.Commit(root, false)
	if err != nil {
		fork.logger.Errorf("Trie commit error:%s", err.Error())
		return err
	}
	return nil
}

func (fork *fork) insertBlock(block types.Block) error {
	blockByte, err := types.MarshalBlock(&block)
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

func (fork *fork) getBlock(height uint64) *types.Block {
	bytes, _ := fork.db.Get(generateHeightKey(height))
	block, err := types.UnMarshalBlock(bytes)
	if err != nil {
		logger.Errorf("Fail to ummarshal block, error:%s", err.Error())
	}
	return block
}

func (fork *fork) deleteBlock(height uint64) bool {
	err := fork.db.Delete(generateHeightKey(height))
	if err != nil {
		logger.Errorf("Fail to delete block, error:%s", err.Error())
		return false
	}
	return true
}
