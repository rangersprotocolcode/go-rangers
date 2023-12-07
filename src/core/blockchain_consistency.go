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

import "com.tuntun.rangers/node/src/middleware/types"

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
