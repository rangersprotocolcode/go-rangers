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
)

const (
	forkDBPrefix            = "fork"
	commonAncestorHeightKey = "commonAncestor"
	lastestHeightKey        = "lastest"
	chainPieceLength        = 9
)

func (chain *blockChain) getChainPiece(sourceChainHeight uint64) []*types.BlockHeader {
	chain.lock.Lock("GetChainPieceInfo")
	defer chain.lock.Unlock("GetChainPieceInfo")
	localHeight := chain.latestBlock.Height

	var endHeight uint64 = 0
	if localHeight < sourceChainHeight {
		endHeight = localHeight
	} else {
		endHeight = sourceChainHeight
	}

	var height uint64 = 0
	if sourceChainHeight > chainPieceLength {
		height = sourceChainHeight - chainPieceLength
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
		if count >= 10 {
			break
		}

		header := chain.QueryBlockHeaderByHeight(i, true)
		if header == nil {
			blockSyncLogger.Errorf("Block chain get nil block!Height:%d", i)
			break
		}
		block := chain.queryBlockByHash(header.Hash)
		if block == nil {
			blockSyncLogger.Errorf("Block chain get nil block!Height:%d", i)
			break
		}
		result = append(result, block)
		count++
	}
	return result
}
func (chain *blockChain) tryMergeFork(fork *fork) bool {
	chain.lock.Lock("tryMergeFork")
	defer chain.lock.Unlock("tryMergeFork")
	defer blockSyncLogger.Debugf("tryMergeFork return")

	localTopHeader := chain.latestBlock
	blockSyncLogger.Debugf("Try merge fork.Local chain:%d-%d,fork:%d-%d", localTopHeader.Height, localTopHeader.TotalQN, fork.latestBlock.Height, fork.latestBlock.TotalQN)
	if fork.latestBlock.TotalQN < localTopHeader.TotalQN {
		return false
	}

	//重新确定共同祖先
	var commonAncestor *types.BlockHeader
	for height := fork.header; height <= fork.latestBlock.Height; height++ {
		blockSyncLogger.Debugf("height:%d", height)
		forkBlock := fork.getBlock(height)
		if forkBlock == nil {
			break
		}
		if chain.GetBlockHash(height) != forkBlock.Header.Hash {
			break
		}
		commonAncestor = forkBlock.Header
	}

	if commonAncestor == nil {
		blockSyncLogger.Debugf("Try merge fork. common ancestor is nil.")
		return false
	}

	blockSyncLogger.Debugf("Try merge fork. common ancestor:%d", commonAncestor.Height)
	if fork.latestBlock.TotalQN == localTopHeader.TotalQN && chain.nextPvGreatThanFork(commonAncestor, *fork) {
		return false
	}

	chain.removeFromCommonAncestor(commonAncestor)
	for height := fork.header + 1; height <= fork.latestBlock.Height; height++ {
		forkBlock := fork.getBlock(height)
		if forkBlock == nil {
			return false
		}
		var result types.AddBlockResult
		blockSyncLogger.Debugf("add block on chain.%d,%s", forkBlock.Header.Height, forkBlock.Header.Hash.String())
		result = blockChainImpl.addBlockOnChain(forkBlock)
		if result != types.AddBlockSucc {
			return false
		}
	}
	return true
}

func (chain *blockChain) nextPvGreatThanFork(commonAncestor *types.BlockHeader, fork fork) bool {
	commonAncestorHeight := commonAncestor.Height
	if commonAncestorHeight < fork.latestBlock.Height && commonAncestorHeight < chain.latestBlock.Height {
		forkBlock := fork.getBlock(commonAncestorHeight + 1)
		chainBlock := chain.QueryBlock(commonAncestorHeight + 1)
		if forkBlock != nil && chainBlock != nil && chainBlock.Header.ProveValue.Cmp(forkBlock.Header.ProveValue) < 0 {
			return false
		}
	}
	return true
}
