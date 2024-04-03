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
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/utility"
	"encoding/json"
)

func (chain *blockChain) verifyBlock(bh *types.BlockHeader, txs []*types.Transaction, setHash bool) ([]common.Hashes, int8) {
	start := utility.GetTime()
	logger.Infof("verifyBlock. hash:%v,height:%d,preHash:%v,len header tx:%d,len tx:%d", bh.Hash.String(), bh.Height, bh.PreHash.String(), len(bh.Transactions), len(txs))
	defer func() {
		logger.Infof("verifyBlock end. hash:%v,height:%d,preHash:%v,len header tx:%d,len tx:%d, cost: %s", bh.Hash.String(), bh.Height, bh.PreHash.String(), len(bh.Transactions), len(txs), utility.GetTime().Sub(start).String())
	}()

	// use cache before verify
	if chain.verifiedBlocks.Contains(bh.Hash) {
		return nil, 0
	}

	pre := chain.queryBlockHeaderByHash(bh.PreHash)
	if nil == pre {
		if txs != nil {
			chain.futureBlocks.Add(bh.PreHash, &types.Block{Header: bh, Transactions: txs})
		}
		return nil, 2
	}

	if common.IsProposal008() {
		for _, tx := range txs {
			if chain.transactionPool.GetExecuted(tx.Hash) != nil {
				logger.Debugf("tx has already on chain:%s", tx.Hash.String())
				return nil, -1
			}
		}
	}

	miss, missingTx, transactions := chain.missTransaction(*bh, txs)
	if miss {
		return missingTx, 1
	}

	requestIds := getRequestIdFromTransactions(transactions, pre.RequestIds)
	if requestIds["fixed"] != bh.RequestIds["fixed"] {
		logger.Debugf("request id diff, %v, %v, localPre: %v", requestIds["fixed"], bh.RequestIds["fixed"], pre.RequestIds)
		for _, tx := range transactions {
			logger.Debugf("request id diff, tx: %v", tx)
		}
		return nil, -1
	}

	logger.Debugf("validateTxRoot,tx tree root:%v,len txs:%d,miss len:%d", bh.TxTree.Hex(), len(transactions), len(missingTx))
	if !common.IsProposal020() && !chain.validateTxRoot(bh.TxTree, transactions) {
		return nil, -1
	}

	block := types.Block{Header: bh, Transactions: transactions}
	executeTxResult, _, _ := chain.executeTransaction(&block, setHash)
	if !executeTxResult {
		return nil, -1
	}
	if len(block.Transactions) != 0 {
		chain.verifiedBodyCache.Add(block.Header.Hash, block.Transactions)
	}

	return nil, 0
}

func (chain *blockChain) hasPreBlock(bh types.BlockHeader) bool {
	return chain.HasBlockByHash(bh.PreHash)
}

func (chain *blockChain) missTransaction(bh types.BlockHeader, txs []*types.Transaction) (bool, []common.Hashes, []*types.Transaction) {
	var (
		missing      []common.Hashes
		transactions []*types.Transaction
	)

	if nil == txs {
		transactions, missing, _ = chain.queryTxsByBlockHash(bh.Hash, bh.Transactions)
	} else {
		transactions = txs
	}

	if 0 != len(missing) {
		var castorId groupsig.ID
		error := castorId.Deserialize(bh.Castor)
		if error != nil {
			panic("Groupsig id deserialize error:" + error.Error())
		}

		hashList := make([]common.Hashes, 0)
		for _, tx := range missing {
			logger.Debugf("miss tx:%s", tx.ShortS())
			hashList = append(hashList, tx)
			if len(hashList) > 100 {
				m := &transactionRequestMessage{TransactionHashes: hashList, CurrentBlockHash: bh.Hash, BlockHeight: bh.Height, BlockPv: bh.ProveValue}
				go requestTransaction(*m, castorId.GetHexString())
				hashList = make([]common.Hashes, 0)
			}
		}

		m := &transactionRequestMessage{TransactionHashes: hashList, CurrentBlockHash: bh.Hash, BlockHeight: bh.Height, BlockPv: bh.ProveValue}
		go requestTransaction(*m, castorId.GetHexString())
		return true, missing, transactions
	}

	return false, missing, transactions
}

func (chain *blockChain) validateTxRoot(txMerkleTreeRoot common.Hash, txs []*types.Transaction) bool {
	txTree := calcTxTree(txs)

	if !bytes.Equal(txTree.Bytes(), txMerkleTreeRoot.Bytes()) {
		logger.Errorf("Fail to verify txTree, hash1:%s hash2:%s", txTree.Hex(), txMerkleTreeRoot.Hex())
		return false
	}
	return true
}

func calcTxTree(txs []*types.Transaction) common.Hash {
	if nil == txs || 0 == len(txs) {
		return emptyHash
	}

	buf := new(bytes.Buffer)
	for _, tx := range txs {
		if 0 == tx.Type {
			continue
		}
		buf.Write(tx.Hash.Bytes())
	}
	return common.BytesToHash(common.Sha256(buf.Bytes()))
}

func calcReceiptsTree(receipts types.Receipts) common.Hash {
	if nil == receipts || 0 == len(receipts) {
		return emptyHash
	}

	buf := new(bytes.Buffer)
	for _, receipt := range receipts {
		if nil == receipt {
			continue
		}

		encode, err := json.Marshal(receipt)
		if err == nil {
			buf.Write(encode)
		}
	}
	return common.BytesToHash(common.Sha256(buf.Bytes()))
}
