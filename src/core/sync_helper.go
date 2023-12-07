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

import (
	"bytes"
	"com.tuntun.rangers/node/src/middleware/types"
)

func (iterator *GroupForkIterator) Current() *types.Group {
	return iterator.current
}

func (iterator *GroupForkIterator) MovePre() *types.Group {
	if SyncProcessor == nil || iterator.current == nil {
		return nil
	}
	preGroupId := iterator.current.Header.PreGroup
	if SyncProcessor.groupFork != nil && iterator.current.GroupHeight > SyncProcessor.groupFork.header {
		iterator.current = SyncProcessor.groupFork.getGroupById(preGroupId)
	} else {
		iterator.current = groupChainImpl.GetGroupById(iterator.current.Header.PreGroup)
	}
	return iterator.current
}

func (p *syncProcessor) GetBlockHeader(height uint64) *types.BlockHeader {
	var bh *types.BlockHeader
	if p.blockFork != nil {
		forkBlock := p.blockFork.getBlock(height)
		if forkBlock != nil {
			bh = forkBlock.Header
		}
	}
	if bh == nil {
		bh = p.blockChain.QueryBlockHeaderByHeight(height, true)
	}
	return bh
}

func (p *syncProcessor) GetGroupById(id []byte) *types.Group {
	var group *types.Group
	if p.groupFork != nil {
		group = p.groupFork.getGroupById(id)
	}
	if group == nil {
		group = p.groupChain.getGroupById(id)
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
