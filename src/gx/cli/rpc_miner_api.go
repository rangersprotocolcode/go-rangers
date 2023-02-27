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

package cli

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus"
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"com.tuntun.rocket/node/src/core"
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"fmt"
)

func (api *GtasAPI) GetAllGroups() (*Result, error) {
	count := core.GetGroupChain().Count()
	return api.GetGroupsAfter(count)
}

func (api *GtasAPI) GetGroupsAfter(height uint64) (*Result, error) {
	count := core.GetGroupChain().Count()
	if count <= height {
		return failResult("exceed local height")
	}
	groups := make([]*types.Group, count-height)
	for i := height; i < count; i++ {
		group := core.GetGroupChain().GetGroupByHeight(i)
		if nil != group {
			groups[i-height] = group
		}

	}

	ret := make([]map[string]interface{}, 0)
	h := height
	for _, g := range groups {
		gmap := convertGroup(g)
		gmap["height"] = h
		h++
		ret = append(ret, gmap)
	}
	return successResult(ret)
}

func (api *GtasAPI) PageGetGroups(page, limit int) (*Result, error) {
	chain := core.GetGroupChain()
	total := chain.Count()
	pageObject := PageObjects{
		Total: total,
		Data:  make([]interface{}, 0),
	}

	i := 0
	b := int64(0)
	if page < 1 {
		page = 1
	}
	num := uint64((page - 1) * limit)
	if total < num {
		return successResult(pageObject)
	}
	b = int64(total - num)

	for i < limit && b >= 0 {
		g := chain.GetGroupByHeight(uint64(b))
		b--
		if g == nil {
			continue
		}

		mems := make([]string, 0)
		for _, mem := range g.Members {
			mems = append(mems, groupsig.DeserializeID(mem).ShortS())
		}

		group := &Group{
			Height:        uint64(b + 1),
			Id:            groupsig.DeserializeID(g.Id),
			PreId:         groupsig.DeserializeID(g.Header.PreGroup),
			ParentId:      groupsig.DeserializeID(g.Header.Parent),
			BeginHeight:   g.Header.WorkHeight,
			DismissHeight: g.Header.DismissHeight,
			Members:       mems,
		}
		pageObject.Data = append(pageObject.Data, group)
		i++
	}
	return successResult(pageObject)
}

func (api *GtasAPI) GetWorkGroup(height uint64) (*Result, error) {
	groups := consensus.Proc.GetCastQualifiedGroupsFromChain(height)
	ret := make([]map[string]interface{}, 0)
	h := height
	for _, g := range groups {
		gmap := convertGroup(g)
		gmap["height"] = h
		h++
		ret = append(ret, gmap)
	}
	return successResult(ret)
}

func (api *GtasAPI) WorkGroupNum(height uint64) (*Result, error) {
	groups := consensus.Proc.GetCastQualifiedGroups(height)
	return successResult(groups)
}

func (api *GtasAPI) GetCurrentWorkGroup() (*Result, error) {
	height := core.GetBlockChain().Height()
	return api.GetWorkGroup(height)
}

func convertGroup(g *types.Group) map[string]interface{} {
	gmap := make(map[string]interface{})
	if g.Id != nil && len(g.Id) != 0 {
		gmap["group_id"] = groupsig.DeserializeID(g.Id).GetHexString()
		gmap["g_hash"] = g.Header.Hash.String()
	}
	gmap["parent"] = groupsig.DeserializeID(g.Header.Parent).GetHexString()
	gmap["pre"] = groupsig.DeserializeID(g.Header.PreGroup).GetHexString()
	gmap["begin_height"] = g.Header.WorkHeight
	gmap["dismiss_height"] = g.Header.DismissHeight
	gmap["create_height"] = g.Header.CreateHeight
	gmap["create_time"] = g.Header.BeginTime
	gmap["mem_size"] = len(g.Members)
	mems := make([]string, 0)
	for _, mem := range g.Members {
		memberStr := groupsig.DeserializeID(mem).GetHexString()
		mems = append(mems, memberStr[0:6]+"-"+memberStr[len(memberStr)-6:])
	}
	gmap["members"] = mems
	gmap["extends"] = g.Header.Extends
	return gmap
}

func (api *GtasAPI) GetMiner(minerId string) (*Result, error) {
	accountDB := middleware.AccountDBManagerInstance.GetLatestStateDB()
	miner := service.MinerManagerImpl.GetMiner(common.FromHex(minerId), accountDB)

	if nil == miner {
		return failResult(fmt.Sprintf("miner: %s does not exist", minerId))
	}

	return successResult(miner)
}
