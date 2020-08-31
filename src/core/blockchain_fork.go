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
	"math"
)

func (chain *blockChain) getChainPieceInfo(reqHeight uint64) []*types.BlockHeader {
	chain.lock.Lock("GetChainPieceInfo")
	defer chain.lock.Unlock("GetChainPieceInfo")
	localHeight := chain.latestBlock.Height
	logger.Debugf("Req chain piece info height:%d,local height:%d", reqHeight, localHeight)

	var height uint64
	if reqHeight > localHeight {
		height = localHeight
	} else {
		height = reqHeight
	}

	chainPiece := make([]*types.BlockHeader, 0)

	var lastChainPieceBlock *types.BlockHeader
	for i := height; i <= chain.Height(); i++ {
		bh := chain.QueryBlockHeaderByHeight(i, true)
		if nil == bh {
			continue
		}
		lastChainPieceBlock = bh
		break
	}
	if lastChainPieceBlock == nil {
		logger.Errorf("Last chain piece block should not be nil!")
		return chainPiece
	}

	chainPiece = append(chainPiece, lastChainPieceBlock)

	hash := lastChainPieceBlock.PreHash
	for i := 0; i < chainPieceLength; i++ {
		header := chain.queryBlockHeaderByHash(hash)
		if header == nil {
			//创世块 pre hash 不存在
			break
		}
		chainPiece = append(chainPiece, header)
		hash = header.PreHash
	}
	return chainPiece
}

func (chain *blockChain) getChainPieceBlocks(reqHeight uint64) []*types.Block {
	chain.lock.Lock("GetChainPieceBlocks")
	defer chain.lock.Unlock("GetChainPieceBlocks")
	localHeight := chain.latestBlock.Height
	logger.Debugf("Req chain piece block height:%d,local height:%d", reqHeight, localHeight)

	var height uint64
	if reqHeight > localHeight {
		height = localHeight
	} else {
		height = reqHeight
	}

	var firstChainPieceBlock *types.BlockHeader
	for i := height; i <= chain.Height(); i++ {
		bh := chain.QueryBlockHeaderByHeight(i, true)
		if nil == bh {
			continue
		}
		firstChainPieceBlock = bh
		break
	}
	if firstChainPieceBlock == nil {
		panic("last chain piece block should not be nil!")
	}

	chainPieceBlocks := make([]*types.Block, 0)
	for i := firstChainPieceBlock.Height; i <= chain.Height(); i++ {
		bh := chain.QueryBlockHeaderByHeight(i, true)
		if nil == bh {
			continue
		}
		b := chain.queryBlockByHash(bh.Hash)
		if nil == b {
			continue
		}
		chainPieceBlocks = append(chainPieceBlocks, b)
		if len(chainPieceBlocks) > chainPieceBlockLength {
			break
		}
	}
	return chainPieceBlocks
}

//status 0 忽略该消息  不需要同步
//status 1 需要同步ChainPieceBlock
//status 2 需要继续同步ChainPieceInfo
func (chain *blockChain) processChainPieceInfo(chainPiece []*types.BlockHeader, topHeader *types.BlockHeader) (status int, reqHeight uint64) {
	chain.lock.Lock("ProcessChainPieceInfo")
	defer chain.lock.Unlock("ProcessChainPieceInfo")

	localTopHeader := chain.latestBlock
	if topHeader.TotalQN < localTopHeader.TotalQN {
		return 0, math.MaxUint64
	}
	logger.Debugf("ProcessChainPiece %d-%d,topHeader height:%d,totalQn:%d,hash:%v", chainPiece[len(chainPiece)-1].Height, chainPiece[0].Height, topHeader.Height, topHeader.TotalQN, topHeader.Hash.Hex())
	commonAncestor, hasCommonAncestor, index := chain.findCommonAncestor(chainPiece, 0, len(chainPiece)-1)
	if hasCommonAncestor {
		logger.Debugf("Got common ancestor! Height:%d,localHeight:%d", commonAncestor.Height, localTopHeader.Height)
		if topHeader.TotalQN > localTopHeader.TotalQN {
			return 1, commonAncestor.Height + 1
		}

		if topHeader.TotalQN == chain.latestBlock.TotalQN {
			var remoteNext *types.BlockHeader
			for i := index - 1; i >= 0; i-- {
				if chainPiece[i].ProveValue != nil {
					remoteNext = chainPiece[i]
					break
				}
			}
			if remoteNext == nil {
				return 0, math.MaxUint64
			}
			if chain.compareValue(commonAncestor, remoteNext) {
				logger.Debugf("Local value is great than coming value!")
				return 0, math.MaxUint64
			}
			logger.Debugf("Coming value is great than local value!")
			return 1, commonAncestor.Height + 1
		}
		return 0, math.MaxUint64
	}
	//Has no common ancestor
	if index == 0 {
		logger.Debugf("Local chain is same with coming chain piece.")
		return 1, chainPiece[0].Height + 1
	} else {
		var preHeight uint64
		preBlock := chain.queryBlockByHash(chain.latestBlock.PreHash)
		if preBlock != nil {
			preHeight = preBlock.Header.Height
		} else {
			preHeight = 0
		}
		lastPieceHeight := chainPiece[len(chainPiece)-1].Height

		var minHeight uint64
		if preHeight < lastPieceHeight {
			minHeight = preHeight
		} else {
			minHeight = lastPieceHeight
		}
		var baseHeight uint64
		if minHeight != 0 {
			baseHeight = minHeight - 1
		} else {
			baseHeight = 0
		}
		logger.Debugf("Do not find common ancestor in chain piece info:%d-%d!Continue to request chain piece info,base height:%d", chainPiece[len(chainPiece)-1].Height, chainPiece[0].Height, baseHeight, )
		return 2, baseHeight
	}

}

func (chain *blockChain) mergeFork(blockChainPiece []*types.Block, topHeader *types.BlockHeader) {
	if topHeader == nil || len(blockChainPiece) == 0 {
		return
	}
	chain.lock.Lock("MergeFork")
	defer chain.lock.Unlock("MergeFork")

	localTopHeader := chain.latestBlock
	if blockChainPiece[len(blockChainPiece)-1].Header.TotalQN < localTopHeader.TotalQN {
		return
	}

	if blockChainPiece[len(blockChainPiece)-1].Header.TotalQN == localTopHeader.TotalQN {
		if !chain.compareNextBlockPv(blockChainPiece[0].Header) {
			return
		}
	}

	originCommonAncestorHash := (*blockChainPiece[0]).Header.PreHash
	originCommonAncestor := chain.queryBlockByHash(originCommonAncestorHash)
	if originCommonAncestor == nil {
		return
	}

	var index = -100
	for i := 0; i < len(blockChainPiece); i++ {
		block := blockChainPiece[i]
		if chain.queryBlockByHash(block.Header.Hash) == nil {
			index = i - 1
			break
		}
	}

	if index == -100 {
		return
	}

	var realCommonAncestor *types.BlockHeader
	if index == -1 {
		realCommonAncestor = originCommonAncestor.Header
	} else {
		realCommonAncestor = blockChainPiece[index].Header
	}
	chain.removeFromCommonAncestor(realCommonAncestor)

	for i := index + 1; i < len(blockChainPiece); i++ {
		block := blockChainPiece[i]
		var result types.AddBlockResult
		result = blockChainImpl.addBlockOnChain("", block, types.MergeFork)
		if result != types.AddBlockSucc {
			return
		}
	}
}

func (chain *blockChain) compareNextBlockPv(remoteNextHeader *types.BlockHeader) bool {
	if remoteNextHeader == nil {
		return false
	}
	remoteNextBlockPv := remoteNextHeader.ProveValue
	if remoteNextBlockPv == nil {
		return false
	}
	commonAncestor := chain.queryBlockByHash(remoteNextHeader.PreHash)
	if commonAncestor == nil {
		logger.Debugf("MergeFork common ancestor should not be nil!")
		return false
	}

	var localNextBlock *types.BlockHeader
	for i := commonAncestor.Header.Height + 1; i <= chain.Height(); i++ {
		bh := chain.QueryBlockHeaderByHeight(i, true)
		if nil == bh {
			continue
		}
		localNextBlock = bh
		break
	}
	if localNextBlock == nil {
		return true
	}
	if remoteNextBlockPv.Cmp(localNextBlock.ProveValue) > 0 {
		return true
	}
	return false
}

func (chain *blockChain) findCommonAncestor(chainPiece []*types.BlockHeader, l int, r int) (*types.BlockHeader, bool, int) {
	if l > r {
		return nil, false, -1
	}

	m := (l + r) / 2
	result := chain.isCommonAncestor(chainPiece, m)
	if result == 0 {
		return chainPiece[m], true, m
	}

	if result == 1 {
		return chain.findCommonAncestor(chainPiece, l, m-1)
	}

	if result == -1 {
		return chain.findCommonAncestor(chainPiece, m+1, r)
	}
	if result == 100 {
		return nil, false, 0
	}
	return nil, false, -1
}

//bhs 中没有空值
//返回值
// 0  当前HASH相等，后面一块HASH不相等 是共同祖先
//1   当前HASH相等，后面一块HASH相等
//100  当前HASH相等，但是到达数组边界，找不到后面一块 无法判断同祖先
//-1  当前HASH不相等
//-100 参数不合法
func (chain *blockChain) isCommonAncestor(chainPiece []*types.BlockHeader, index int) int {
	if index < 0 || index >= len(chainPiece) {
		return -100
	}
	he := chainPiece[index]

	bh := chain.QueryBlockHeaderByHeight(he.Height, true)
	if bh == nil {
		logger.Debugf("isCommonAncestor:Height:%d,local hash:%x,coming hash:%x\n", he.Height, nil, he.Hash)
		return -1
	}
	logger.Debugf("isCommonAncestor:Height:%d,local hash:%x,coming hash:%x\n", he.Height, bh.Hash, he.Hash)
	if index == 0 && bh.Hash == he.Hash {
		return 100
	}
	if index == 0 {
		return -1
	}
	//判断链更后面的一块
	afterHe := chainPiece[index-1]
	afterBh := chain.QueryBlockHeaderByHeight(afterHe.Height, true)
	if afterBh == nil {
		logger.Debugf("isCommonAncestor:after block height:%d,local hash:%s,coming hash:%x\n", afterHe.Height, "null", afterHe.Hash)
		if afterHe != nil && bh.Hash == he.Hash {
			return 0
		}
		return -1
	}
	logger.Debugf("isCommonAncestor:after block height:%d,local hash:%x,coming hash:%x\n", afterHe.Height, afterBh.Hash, afterHe.Hash)
	if afterHe.Hash != afterBh.Hash && bh.Hash == he.Hash {
		return 0
	}
	if afterHe.Hash == afterBh.Hash && bh.Hash == he.Hash {
		return 1
	}
	return -1
}