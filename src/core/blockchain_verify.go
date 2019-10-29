package core

import (
	"x/src/middleware/types"
	"x/src/common"
	"bytes"
	"x/src/consensus/groupsig"
	"x/src/middleware/serialize"
	"x/src/storage/trie"
)

func (chain *blockChain) verifyBlock(bh types.BlockHeader, txs []*types.Transaction) ([]common.Hashes, int8) {
	// use cache before verify
	//if chain.verifiedBlocks.Contains(bh.Hash) {
	//	return nil, 0
	//}

	logger.Infof("verifyBlock hash:%v,height:%d,totalQn:%d,preHash:%v,len header tx:%d,len tx:%d", bh.Hash.String(), bh.Height, bh.TotalQN, bh.PreHash.String(), len(bh.Transactions), len(txs))
	if bh.Hash != bh.GenHash() {
		logger.Debugf("Validate block hash error!")
		return nil, -1
	}

	if !chain.hasPreBlock(bh) {
		if txs != nil {
			chain.futureBlocks.Add(bh.PreHash, &types.Block{Header: &bh, Transactions: txs})
		}
		return nil, 2
	}

	miss, missingTx, transactions := chain.missTransaction(bh, txs)
	if miss {
		return missingTx, 1
	}

	logger.Debugf("validateTxRoot,tx tree root:%v,len txs:%d,miss len:%d", bh.TxTree.Hex(), len(transactions), len(missingTx))
	if !chain.validateTxRoot(bh.TxTree, transactions) {
		return nil, -1
	}

	block := types.Block{Header: &bh, Transactions: transactions}
	executeTxResult, _, _ := chain.executeTransaction(&block)
	if !executeTxResult {
		return nil, -1
	}
	if len(block.Transactions) != 0 {
		chain.verifiedBodyCache.Add(block.Header.Hash, block.Transactions)
	}
	return nil, 0
}

func (chain *blockChain) hasPreBlock(bh types.BlockHeader) bool {
	pre := chain.queryBlockHeaderByHash(bh.PreHash)
	if pre == nil {
		return false
	}
	return true
}

func (chain *blockChain) missTransaction(bh types.BlockHeader, txs []*types.Transaction) (bool, []common.Hashes, []*types.Transaction) {
	var missing []common.Hashes
	var transactions []*types.Transaction
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
		for _, tx := range missing {
			logger.Debugf("miss tx:%s", tx.ShortS())
		}
		//向CASTOR索取交易
		m := &transactionRequestMessage{TransactionHashes: missing, CurrentBlockHash: bh.Hash, BlockHeight: bh.Height, BlockPv: bh.ProveValue,}
		go requestTransaction(*m, castorId.String())
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
		buf.Write(tx.Hash.Bytes())
	}
	return common.BytesToHash(common.Sha256(buf.Bytes()))
}

func calcReceiptsTree(receipts types.Receipts) common.Hash {
	if nil == receipts || 0 == len(receipts) {
		return emptyHash
	}

	keybuf := new(bytes.Buffer)
	trie := new(trie.Trie)
	for i := 0; i < len(receipts); i++ {
		if receipts[i] != nil {
			keybuf.Reset()
			serialize.Encode(keybuf, uint(i))
			encode, _ := serialize.EncodeToBytes(receipts[i])
			trie.Update(keybuf.Bytes(), encode)
		}
	}
	hash := trie.Hash()

	return common.BytesToHash(hash.Bytes())
}
