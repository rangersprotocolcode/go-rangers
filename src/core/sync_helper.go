package core

import (
	"bytes"
	"com.tuntun.rocket/node/src/middleware/types"
)

func (iterator *GroupForkIterator) Current() *types.Group {
	return iterator.current
}

func (iterator *GroupForkIterator) MovePre() *types.Group {
	if SyncProcessor == nil || iterator.current == nil {
		return nil
	}
	preGroupId := iterator.current.Header.PreGroup
	if iterator.current.GroupHeight > SyncProcessor.groupFork.header && SyncProcessor.groupFork != nil {
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

func (p *syncProcessor) GetGroupById(id []byte) *types.Group {
	var group *types.Group
	group = p.groupChain.getGroupById(id)
	if group == nil && p.groupFork != nil {
		group = p.groupFork.getGroupById(id)
	}
	return group
}

func (p *syncProcessor) GetAvailableGroupsByMinerId(height uint64, minerId []byte) []*types.Group {
	allGroups := p.groupChain.availableGroupsAtFromFork(height)
	group := make([]*types.Group, 0)

	for _, g := range allGroups {
		for _, mem := range g.Members {
			if bytes.Equal(mem, minerId) {
				group = append(group, g)
				break
			}
		}
	}

	return group
	return nil
}

func (chain *groupChain) availableGroupsAtFromFork(h uint64) []*types.Group {
	iter := chain.ForkIterator()
	gs := make([]*types.Group, 0)
	for g := iter.Current(); g != nil; g = iter.MovePre() {
		if g.Header.DismissHeight > h {
			gs = append(gs, g)
		} else {
			genesis := chain.GetGroupByHeight(0)
			gs = append(gs, genesis)
			break
		}
	}
	return gs
}
