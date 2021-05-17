package core

import "com.tuntun.rocket/node/src/middleware/types"

func (iterator *GroupForkIterator) Current() *types.Group {
	return iterator.current
}

func (iterator *GroupForkIterator) MovePre() *types.Group {
	if SyncProcessor == nil || SyncProcessor.groupFork == nil || iterator.current == nil {
		return nil
	}
	preGroupId := iterator.current.Header.PreGroup
	if iterator.current.GroupHeight > SyncProcessor.groupFork.header {
		iterator.current = SyncProcessor.groupFork.getGroupById(preGroupId)
	} else {
		iterator.current = groupChainImpl.GetGroupById(iterator.current.Header.PreGroup)
	}
	return iterator.current
}

func (p *syncProcessor) GetBlockHeader(height uint64) *types.BlockHeader {
	var bh *types.BlockHeader
	bh = p.blockChain.QueryBlockHeaderByHeight(height, true)
	if bh == nil && p.blockFork != nil {
		forkBlock := p.blockFork.getBlock(height)
		if forkBlock != nil {
			bh = forkBlock.Header
		}
	}
	return bh
}
func (p *syncProcessor) GetAvailableGroupsByMinerId(height uint64, minerId []byte) []*types.Group {
	return nil
}

func (p *syncProcessor) GetGroupById(id []byte) *types.Group {
	var group *types.Group
	group = p.groupChain.getGroupById(id)
	if group == nil && p.groupFork != nil {
		group = p.groupFork.getGroupById(id)
	}
	return group
}
