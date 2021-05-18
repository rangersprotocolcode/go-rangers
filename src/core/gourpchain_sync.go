package core

import "com.tuntun.rocket/node/src/middleware/types"

func (chain *groupChain) height() uint64 {
	count := chain.count
	if count > 1 {
		return count - 1
	} else {
		return 0
	}
}

func (chain *groupChain) removeFromCommonAncestor(commonAncestor *types.Group) {
	chain.lock.Lock()
	defer chain.lock.Unlock()
	syncLogger.Debugf("[GroupChain]remove from common ancestor.hash:%s,height:%d,local height:%d", commonAncestor.Header.Hash.String(), commonAncestor.GroupHeight, chain.height())
	for height := chain.height(); height > commonAncestor.GroupHeight; height-- {
		group := chain.getGroupByHeight(height)
		if group == nil {
			syncLogger.Debugf("Group chain get nil height:%d", height)
			continue
		}
		chain.remove(group)
		syncLogger.Debugf("Remove local group hash:%s, height %d", group.Header.Hash.String(), group.GroupHeight)
	}
}

func (chain *groupChain) getFirstGroupBelowHeight(createBlockHeight uint64) *types.Group {
	iterator := chain.Iterator()
	for g := iterator.Current(); g != nil; g = iterator.MovePre() {
		if g.Header.CreateHeight <= createBlockHeight {
			return g
		}
	}
	return nil
}
