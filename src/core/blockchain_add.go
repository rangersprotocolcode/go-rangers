package core

import (
	"x/src/middleware/types"
	"x/src/storage/account"
	"x/src/common"
	"x/src/utility"
	"x/src/middleware/notify"
	"math/big"
	"errors"
)

func (chain *blockChain) consensusVerify(source string, b *types.Block) (types.AddBlockResult, bool) {
	if b == nil {
		return types.AddBlockFailed, false
	}

	if !chain.hasPreBlock(*b.Header) {
		logger.Debugf("coming block %s,%d has no pre on local chain.Forking...", b.Header.Hash.String(), b.Header.Height)
		chain.futureBlocks.Add(b.Header.PreHash, b)
		go chain.forkProcessor.requestChainPieceInfo(source, chain.latestBlock.Height)
		return types.Forking, false
	}

	if chain.queryBlockHeaderByHash(b.Header.Hash) != nil {
		return types.BlockExisted, false
	}

	if check, err := consensusHelper.CheckProveRoot(b.Header); !check {
		logger.Errorf("checkProveRoot fail, err=%v", err.Error())
		return types.AddBlockFailed, false
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

func (chain *blockChain) addBlockOnChain(source string, b *types.Block, situation types.AddBlockOnChainSituation) types.AddBlockResult {
	topBlock := chain.latestBlock
	logger.Debugf("coming block:hash=%v, preH=%v, height=%v,totalQn:%d", b.Header.Hash.Hex(), b.Header.PreHash.Hex(), b.Header.Height, b.Header.TotalQN)
	logger.Debugf("Local topHash=%v, topPreHash=%v, height=%v,totalQn:%d", topBlock.Hash.Hex(), topBlock.PreHash.Hex(), topBlock.Height, topBlock.TotalQN)

	if _, verifyResult := chain.verifyBlock(*b.Header, b.Transactions); verifyResult != 0 {
		logger.Errorf("Fail to VerifyCastingBlock, reason code:%d \n", verifyResult)
		if verifyResult == 2 {
			logger.Debugf("coming block  has no pre on local chain.Forking...", )
			go chain.forkProcessor.requestChainPieceInfo(source, chain.latestBlock.Height)
		}
		return types.AddBlockFailed
	}

	if b.Header.PreHash == topBlock.Hash {
		result, _ := chain.insertBlock(b)
		return result
	}
	if b.Header.Hash == topBlock.Hash || chain.queryBlockHeaderByHash(b.Header.Hash) != nil {
		return types.BlockExisted
	}

	if b.Header.TotalQN < topBlock.TotalQN {
		if situation == types.Sync {
			go chain.forkProcessor.requestChainPieceInfo(source, chain.latestBlock.Height)
		}
		return types.BlockTotalQnLessThanLocal
	}
	commonAncestor := chain.queryBlockHeaderByHash(b.Header.PreHash)
	logger.Debugf("commonAncestor hash:%s height:%d", commonAncestor.Hash.Hex(), commonAncestor.Height)
	if b.Header.TotalQN > topBlock.TotalQN {
		chain.removeFromCommonAncestor(commonAncestor)
		return chain.addBlockOnChain(source, b, situation)
	}
	if b.Header.TotalQN == topBlock.TotalQN {
		if chain.compareValue(commonAncestor, b.Header) {
			if situation == types.Sync {
				go chain.forkProcessor.requestChainPieceInfo(source, chain.latestBlock.Height)
			}
			return types.BlockTotalQnLessThanLocal
		}
		chain.removeFromCommonAncestor(commonAncestor)
		return chain.addBlockOnChain(source, b, situation)
	}
	go chain.forkProcessor.requestChainPieceInfo(source, chain.latestBlock.Height)
	return types.Forking
}

func (chain *blockChain) executeTransaction(block *types.Block) (bool, *account.AccountDB, types.Receipts) {
	preBlock := chain.queryBlockHeaderByHash(block.Header.PreHash)
	if preBlock == nil {
		panic("Pre block nil !!")
	}
	preRoot := common.BytesToHash(preBlock.StateTree.Bytes())
	if len(block.Transactions) > 0 {
		logger.Debugf("NewAccountDB height:%d StateTree:%s preHash:%s preRoot:%s", block.Header.Height, block.Header.StateTree.Hex(), preBlock.Hash.Hex(), preRoot.Hex())
	}
	state, err := account.NewAccountDB(preRoot, chain.stateDB)
	if err != nil {
		logger.Errorf("Fail to new statedb, error:%s", err)
		return false, state, nil
	}

	stateRoot, _, _, receipts, err, _ := chain.executor.Execute(state, block, block.Header.Height, "fullverify")
	if common.ToHex(stateRoot.Bytes()) != common.ToHex(block.Header.StateTree.Bytes()) {
		logger.Errorf("Fail to verify state tree, hash1:%x hash2:%x", stateRoot.Bytes(), block.Header.StateTree.Bytes())
		return false, state, receipts
	}
	receiptsTree := calcReceiptsTree(receipts).Bytes()
	if common.ToHex(receiptsTree) != common.ToHex(block.Header.ReceiptTree.Bytes()) {
		logger.Errorf("fail to verify receipt, hash1:%s hash2:%s", common.ToHex(receiptsTree), common.ToHex(block.Header.ReceiptTree.Bytes()))
		return false, state, receipts
	}

	chain.verifiedBlocks.Add(block.Header.Hash, &castingBlock{state: state, receipts: receipts,})
	return true, state, receipts
}

func (chain *blockChain) insertBlock(remoteBlock *types.Block) (types.AddBlockResult, []byte) {
	logger.Debugf("Insert block hash:%s,height:%d", remoteBlock.Header.Hash.Hex(), remoteBlock.Header.Height)
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
	if nil != common.DefaultLogger {
		b := accountDB.GetBalance(common.HexToAddress("aaa"))
		if b.Sign() != 0 {
			common.DefaultLogger.Errorf("check balance, balance: %s, height: %d", b.String(), remoteBlock.Header.Height)
		}

	}
	if !saveStateResult {
		return types.AddBlockFailed, nil
	}

	if !chain.updateLastBlock(accountDB, remoteBlock.Header, headerByte) {
		return types.AddBlockFailed, headerByte
	}

	chain.updateVerifyHash(remoteBlock)

	chain.updateTxPool(remoteBlock, receipts)
	chain.topBlocks.Add(remoteBlock.Header.Height, remoteBlock.Header)

	dumpTxs(remoteBlock.Transactions, remoteBlock.Header.Height)
	chain.eraseAddBlockMark()
	chain.successOnChainCallBack(remoteBlock, headerByte)
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
	err := chain.heightDB.Put(utility.UInt64ToByte(height), headerByte)
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
		b := value.(*castingBlock)
		state = b.state
		receipts = b.receipts
	} else {
		var executeTxResult bool
		executeTxResult, state, receipts = chain.executeTransaction(b)
		if !executeTxResult {
			logger.Errorf("Fail to execute txs!")
			return false, state, receipts
		}
	}

	if nil != common.DefaultLogger {
		bb := state.GetBalance(common.HexToAddress("aaa"))
		if bb.Sign() != 0 {
			common.DefaultLogger.Errorf("check balance before commit, balance: %s, height: %d", bb.String(), b.Header.Height)
		}

	}

	root, err := state.Commit(true)
	if err != nil {
		logger.Errorf("State commit error:%s", err.Error())
		return false, state, receipts
	}

	if nil != common.DefaultLogger {
		bb := state.GetBalance(common.HexToAddress("aaa"))
		if bb.Sign() != 0 {
			common.DefaultLogger.Errorf("check balance after commit, balance: %s, height: %d", bb.String(), b.Header.Height)
		}

	}

	trieDB := chain.stateDB.TrieDB()
	err = trieDB.Commit(root, false)
	if err != nil {
		logger.Errorf("Trie commit error:%s", err.Error())
		return false, state, receipts
	}
	return true, state, receipts
}

func (chain *blockChain) updateLastBlock(state *account.AccountDB, header *types.BlockHeader, headerJson []byte) bool {
	err := chain.heightDB.Put([]byte(latestBlockKey), headerJson)
	if err != nil {
		logger.Errorf("Fail to put %s, error:%s", latestBlockKey, err.Error())
		return false
	}
	chain.latestStateDB = state
	chain.latestBlock = header
	logger.Debugf("Update latestStateDB:%s height:%d", header.StateTree.Hex(), header.Height)

	if nil != common.DefaultLogger {
		b := state.GetBalance(common.HexToAddress("aaa"))
		if b.Sign() != 0 {
			common.DefaultLogger.Errorf("check balance, balance: %s, height: %d", b.String(), header.Height)
		}

	}
	return true
}

func (chain *blockChain) updateVerifyHash(block *types.Block) {
	verifyHash := consensusHelper.VerifyHash(block)
	chain.verifyHashDB.Put(utility.UInt64ToByte(block.Header.Height), verifyHash.Bytes())
}

func (chain *blockChain) updateTxPool(block *types.Block, receipts types.Receipts) {
	chain.transactionPool.MarkExecuted(receipts, block.Transactions, block.Header.EvictedTxs)
}

func (chain *blockChain) successOnChainCallBack(remoteBlock *types.Block, headerJson []byte) {
	logger.Infof("ON chain succ! height=%d,hash=%s", remoteBlock.Header.Height, remoteBlock.Header.Hash.Hex())
	notify.BUS.Publish(notify.BlockAddSucc, &notify.BlockOnChainSuccMessage{Block: *remoteBlock,})
	if value, _ := chain.futureBlocks.Get(remoteBlock.Header.Hash); value != nil {
		block := value.(*types.Block)
		logger.Debugf("Get block from future blocks,hash:%s,height:%d", block.Header.Hash.String(), block.Header.Height)
		//todo 这里为了避免死锁只能调用这个方法，但是没办法调用CheckProveRoot全量账本验证了
		chain.addBlockOnChain("", block, types.FutureBlockCache)
		return
	}
	if BlockSyncer != nil {
		topBlockInfo := TopBlockInfo{Hash: chain.latestBlock.Hash, TotalQn: chain.latestBlock.TotalQN, Height: chain.latestBlock.Height, PreHash: chain.latestBlock.PreHash}
		go BlockSyncer.sendTopBlockInfoToNeighbor(topBlockInfo)
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

	consensusLogger.Infof("%v#%s#%d,%d", "ForkAdjustRemoveCommonAncestor", commonAncestor.Hash.ShortS(), commonAncestor.Height, chain.latestBlock.Height)

	for height := chain.latestBlock.Height; height > commonAncestor.Height; height-- {
		header := chain.queryBlockHeaderByHeight(height, true)
		if header == nil {
			//logger.Debugf("removeFromCommonAncestor nil height:%d", height)
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

func (chain *blockChain) compareValue(commonAncestor *types.BlockHeader, remoteHeader *types.BlockHeader) bool {
	if commonAncestor.Height == chain.latestBlock.Height {
		return false
	}
	var localValue *big.Int
	remoteValue := consensusHelper.VRFProve2Value(remoteHeader.ProveValue)
	logger.Debugf("coming hash:%s,coming value is:%v", remoteHeader.Hash.String(), remoteValue)
	logger.Debugf("compareValue hash:%s height:%d latestheight:%d", commonAncestor.Hash.Hex(), commonAncestor.Height, chain.latestBlock.Height)
	for height := commonAncestor.Height + 1; height <= chain.latestBlock.Height; height++ {
		logger.Debugf("compareValue queryBlockHeaderByHeight height:%d ", height)
		header := chain.queryBlockHeaderByHeight(height, true)
		if header == nil {
			logger.Debugf("compareValue queryBlockHeaderByHeight nil !height:%d ", height)
			continue
		}
		localValue = consensusHelper.VRFProve2Value(header.ProveValue)
		logger.Debugf("local hash:%s,local value is:%v", header.Hash.String(), localValue)
		break
	}
	if localValue.Cmp(remoteValue) >= 0 {
		return true
	}
	return false
}

func dumpTxs(txs []*types.Transaction, blockHeight uint64) {
	if txs == nil || len(txs) == 0 {
		return
	}

	for _, tx := range txs {
		common.DefaultLogger.Debugf("Tx on chain dump:%v,block height:%d", tx, blockHeight)
	}
}
