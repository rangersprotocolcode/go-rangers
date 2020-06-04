package core

import "com.tuntun.rocket/node/src/middleware/types"

func (chain *blockChain) markAddBlock(blockByte []byte) bool {
	err := chain.hashDB.Put([]byte(addBlockMark), blockByte)
	if err != nil {
		logger.Errorf("Block chain put addBlockMark error:%s", err.Error())
		return false
	}
	return true
}

func (chain *blockChain) eraseAddBlockMark() {
	chain.hashDB.Delete([]byte(addBlockMark))
}

func (chain *blockChain) markRemoveBlock(block *types.Block) bool {
	blockByte, err := types.MarshalBlock(block)
	if err != nil {
		logger.Errorf("Fail to marshal block, error:%s", err.Error())
		return false
	}

	err = chain.hashDB.Put([]byte(removeBlockMark), blockByte)
	if err != nil {
		logger.Errorf("Block chain put removeBlockMark error:%s", err.Error())
		return false
	}
	return true
}

func (chain *blockChain) eraseRemoveBlockMark() {
	chain.hashDB.Delete([]byte(removeBlockMark))
}

func (chain *blockChain) ensureChainConsistency() {
	addBlockByte, _ := chain.hashDB.Get([]byte(addBlockMark))
	if addBlockByte != nil {
		block, _ := types.UnMarshalBlock(addBlockByte)
		logger.Errorf("ensureChainConsistency find addBlockMark!")
		chain.remove(block)
		chain.eraseAddBlockMark()
	}

	removeBlockByte, _ := chain.hashDB.Get([]byte(removeBlockMark))
	if removeBlockByte != nil {
		block, _ := types.UnMarshalBlock(removeBlockByte)
		logger.Errorf("ensureChainConsistency find removeBlockMark!")
		chain.remove(block)
		chain.eraseRemoveBlockMark()
	}
}
