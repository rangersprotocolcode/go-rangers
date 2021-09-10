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
	"com.tuntun.rocket/node/src/middleware/types"
	"math/big"
)

func (chain *blockChain) getChainPiece(sourceChainHeight uint64) []*types.BlockHeader {
	chain.lock.Lock("getChainPiece")
	defer chain.lock.Unlock("getChainPiece")

	localHeight := chain.latestBlock.Height
	var endHeight uint64 = 0
	if localHeight < sourceChainHeight {
		endHeight = localHeight
	} else {
		endHeight = sourceChainHeight
	}

	var height uint64 = 0
	if sourceChainHeight > blockChainPieceLength {
		height = sourceChainHeight - blockChainPieceLength
	}
	chainPiece := make([]*types.BlockHeader, 0)
	for ; height <= endHeight; height++ {
		header := chain.QueryBlockHeaderByHeight(height, true)
		chainPiece = append(chainPiece, header)
	}
	return chainPiece
}

func (chain *blockChain) getSyncedBlock(reqHeight uint64) []*types.Block {
	chain.lock.Lock("getSyncedBlock")
	defer chain.lock.Unlock("getSyncedBlock")

	result := make([]*types.Block, 0)
	count := 0
	for i := reqHeight; i <= chain.latestBlock.Height; i++ {
		if count >= syncedBlockCount {
			break
		}

		header := chain.QueryBlockHeaderByHeight(i, true)
		if header == nil {
			syncLogger.Errorf("Block chain get nil block!Height:%d", i)
			break
		}
		block := chain.queryBlockByHash(header.Hash)
		if block == nil {
			syncLogger.Errorf("Block chain get nil block!Height:%d", i)
			break
		}
		result = append(result, block)
		count++
	}
	return result
}

func (chain *blockChain) nextPvGreatThanFork(commonAncestor *types.BlockHeader, fork blockChainFork) bool {
	commonAncestorHeight := commonAncestor.Height
	if commonAncestorHeight < fork.latestBlock.Height && commonAncestorHeight < chain.latestBlock.Height {
		forkBlock := fork.getBlock(commonAncestorHeight + 1)
		chainBlockHeader := chain.QueryBlockHeaderByHeight(commonAncestorHeight+1, true)
		if forkBlock != nil && chainBlockHeader != nil {
			return chainPvGreatThanRemote(chainBlockHeader, forkBlock.Header)
		}
	}
	return true
}

func chainPvGreatThanRemote(chainNextBlock *types.BlockHeader, remoteBlock *types.BlockHeader) bool {
	logger.Debugf("[ComparePV]coming block:%s-%d,coming value is:%v", remoteBlock.Hash.String(), remoteBlock.Height, remoteBlock.ProveValue)
	logger.Debugf("[ComparePV]local next block:%s-%d,local value is:%v", chainNextBlock.Hash.String(), chainNextBlock.Height, chainNextBlock.ProveValue)
	compareValue := chainNextBlock.ProveValue.Cmp(remoteBlock.ProveValue)
	if compareValue > 0 {
		return true
	}
	if compareValue < 0 {
		return false
	}

	chainNextHashBig := new(big.Int).SetBytes(chainNextBlock.Hash.Bytes())
	remoteHashBig := new(big.Int).SetBytes(remoteBlock.Hash.Bytes())
	logger.Debugf("[ComparePV]PV is the same,compare hash big:%v,%v", chainNextHashBig, remoteHashBig)
	hashBigCompareValue := chainNextHashBig.Cmp(remoteHashBig)
	if hashBigCompareValue > 0 {
		return true
	}
	return false
}
