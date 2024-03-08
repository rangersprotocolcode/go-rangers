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

func (chain *blockChain) markAddBlock(blockByte []byte) error {
	err := chain.hashDB.Put([]byte(addBlockMark), blockByte)
	if err != nil {
		logger.Errorf("Block chain put addBlockMark error:%s", err.Error())
	}
	return err
}

func (chain *blockChain) eraseAddBlockMark() error {
	err := chain.hashDB.Delete([]byte(addBlockMark))
	if err != nil {
		logger.Errorf("Block chain remove addBlockMark error:%s", err.Error())
	}
	return err
}

func (chain *blockChain) markRemoveBlock(blockByte []byte) error {
	err := chain.hashDB.Put([]byte(removeBlockMark), blockByte)
	if err != nil {
		logger.Errorf("Block chain put removeBlockMark error:%s", err.Error())
	}
	return err
}

func (chain *blockChain) eraseRemoveBlockMark() error {
	err := chain.hashDB.Delete([]byte(removeBlockMark))
	if err != nil {
		logger.Errorf("Block chain remove removeBlockMark error:%s", err.Error())
	}
	return err
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
