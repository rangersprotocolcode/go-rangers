package core

import "com.tuntun.rocket/node/src/middleware/types"

func (chain *groupChain) getGroupChainPiece(sourceChainHeight uint64) []*types.Group {
	chain.lock.RLock()
	defer chain.lock.RUnlock()

	var endHeight uint64 = 0
	if chain.count < sourceChainHeight {
		endHeight = chain.count
	} else {
		endHeight = sourceChainHeight
	}

	var height uint64 = 0
	if sourceChainHeight > groupChainPieceLength {
		height = sourceChainHeight - groupChainPieceLength
	}

	chainPiece := make([]*types.Group, 0)
	for ; height <= endHeight; height++ {
		group := chain.getGroupByHeight(height)
		if group == nil {
			syncLogger.Errorf("Group chain get nil group!Height:%d", height)
			break
		}
		group.GroupHeight = height
		chainPiece = append(chainPiece, group)
	}
	return chainPiece
}

func (chain *groupChain) getSyncedGroup(reqHeight uint64) []*types.Group {
	chain.lock.RLock()
	defer chain.lock.RUnlock()

	result := make([]*types.Group, 0)
	count := 0
	for i := reqHeight; i <= chain.count; i++ {
		if count >= syncedGroupCount {
			break
		}

		group := chain.getGroupByHeight(i)
		if group == nil {
			syncLogger.Errorf("Group chain get nil group!Height:%d", i)
			break
		}
		group.GroupHeight = i
		result = append(result, group)
		count++
	}
	return result
}

func (chain *groupChain) removeFromCommonAncestor(commonAncestor *types.Group) {
	chain.lock.Lock()
	defer chain.lock.Unlock()
	logger.Debugf("[GroupChain]remove from common ancestor.hash:%s,height:%d,local height:%d", commonAncestor.Header.Hash.String(), commonAncestor.GroupHeight, chain.count)
	for height := chain.count; height > commonAncestor.GroupHeight; height-- {
		group := chain.getGroupByHeight(height)
		if group == nil {
			logger.Debugf("Group chain get nil height:%d", height)
			continue
		}
		chain.remove(group)
		logger.Debugf("Remove local group hash:%s, height %d", group.Header.Hash.String(), group.GroupHeight)
	}
}
