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

package group_create

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus/access"
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/consensus/net"
	"com.tuntun.rangers/node/src/core"
	"com.tuntun.rangers/node/src/middleware/log"
	"com.tuntun.rangers/node/src/utility"
	lru "github.com/hashicorp/golang-lru"
	"strconv"
	"sync"
)

var (
	groupCreateLogger      log.Logger
	groupCreateDebugLogger log.Logger

	GroupCreateProcessor groupCreateProcessor
)

type groupCreateProcessor struct {
	minerInfo model.SelfMinerInfo
	context   *createGroupContext

	//key:create group hash,value:create group base height
	createGroupCache *lru.Cache

	// Identifies whether the group height has already been created
	createdHeights      [50]uint64
	createdHeightsIndex int

	groupInitContextCache groupInitContextCache
	joinedGroupStorage    *access.JoinedGroupStorage
	groupSignCollectorMap sync.Map // id==>group hash,value==>GroupPubkeyCollector

	minerReader   *access.MinerPoolReader
	groupAccessor *access.GroupAccessor

	blockChain core.BlockChain
	groupChain core.GroupChain

	NetServer net.NetworkServer
	lock      sync.RWMutex
}

func (p *groupCreateProcessor) Init(minerInfo model.SelfMinerInfo, joinedGroupStorage *access.JoinedGroupStorage) {
	groupCreateLogger = log.GetLoggerByIndex(log.GroupCreateLogConfig, strconv.Itoa(common.InstanceIndex))
	groupCreateDebugLogger = log.GetLoggerByIndex(log.GroupCreateDebugLogConfig, strconv.Itoa(common.InstanceIndex))

	p.minerInfo = minerInfo
	p.createdHeightsIndex = 0

	p.createGroupCache = common.CreateLRUCache(100)

	p.joinedGroupStorage = joinedGroupStorage
	p.groupSignCollectorMap = sync.Map{}
	p.groupInitContextCache = newGroupInitContextCache()

	p.minerReader = access.NewMinerPoolReader()
	access.InitPubkeyPool(p.minerReader)
	p.groupAccessor = access.NewGroupAccessor(core.GetGroupChain())

	p.blockChain = core.GetBlockChain()
	p.groupChain = core.GetGroupChain()
	p.NetServer = net.NewNetworkServer()

	p.lock = sync.RWMutex{}
}

// getMemberSignPubKey get the signature public key of the member in the group
func (p *groupCreateProcessor) GetMemberSignPubKey(groupId groupsig.ID, minerId groupsig.ID) (pk groupsig.Pubkey, ok bool) {
	if jg := p.joinedGroupStorage.GetJoinedGroupInfo(groupId); jg != nil {
		pk, ok = jg.GetMemberSignPK(minerId)
		if !ok && !p.minerInfo.ID.IsEqual(minerId) {
			p.askSignPK(minerId, groupId)
		}
	}
	return
}

func (p *groupCreateProcessor) getInGroupSignSecKey(groupId groupsig.ID) groupsig.Seckey {
	if joinedGroup := p.joinedGroupStorage.GetJoinedGroupInfo(groupId); joinedGroup != nil {
		return joinedGroup.SignSecKey
	}
	return groupsig.Seckey{}
}

func (p *groupCreateProcessor) OnGroupAddSuccess(g *model.GroupInfo) {
	ctx := p.context
	if ctx != nil && ctx.groupInitInfo != nil && ctx.groupInitInfo.GroupHash() == g.GroupInitInfo.GroupHash() {
		top := p.blockChain.Height()
		groupCreateLogger.Infof("onGroupAddSuccess info=%v, gHash=%v, gid=%v, costHeight=%v", ctx.String(), g.GroupInitInfo.GroupHash().ShortS(), g.GroupID.ShortS(), top-ctx.createTopHeight)
		p.removeContext()
		groupCreateDebugLogger.Infof("Group create success. Group hash:%s, group id:%s\n", ctx.groupInitInfo.GroupHash().String(), g.GroupID.GetHexString())
	}

	p.groupSignCollectorMap.Delete(g.GroupInitInfo.GroupHash().Hex())
	if p.joinedGroupStorage.BelongGroup(g.GroupID) {
		p.groupInitContextCache.RemoveContext(g.GroupInitInfo.GroupHash())

		p.NetServer.ReleaseGroupNet(g.GroupInitInfo.GroupHash().String())
	}
	p.createGroupCache.Remove(g.GroupInitInfo.GroupHash())

}

func (p *groupCreateProcessor) removeContext() {
	p.context = nil
}

func (p *groupCreateProcessor) ReleaseGroups(topHeight uint64) (needDimissGroups []groupsig.ID) {
	groups := p.groupAccessor.GetDismissGroups(topHeight - model.Param.CreateGroupInterval)
	ids := make([]groupsig.ID, 0)
	for _, g := range groups {
		ids = append(ids, g.GroupID)
	}

	if len(ids) > 0 {
		groupCreateLogger.Debugf("clean group %v\n", len(ids))
		needDimissGroups = ids
		p.groupAccessor.RemoveGroupsFromCache(ids)
		p.joinedGroupStorage.LeaveGroups(ids)
		for _, g := range groups {
			gid := g.GroupID
			p.NetServer.ReleaseGroupNet(gid.GetHexString())
			p.groupInitContextCache.RemoveContext(g.GroupInitInfo.GroupHash())
		}
	}

	invalidDummyGroups := make([]common.Hash, 0)
	p.groupInitContextCache.forEach(func(gc *groupInitContext) bool {
		if gc.groupInitInfo == nil || gc.status == GisGroupInitDone {
			return true
		}
		groupInitInfo := gc.groupInitInfo
		gHash := groupInitInfo.GroupHash()

		if groupInitInfo.ReadyTimeout(topHeight) {
			if topHeight < groupInitInfo.GroupHeader.CreateHeight+model.Param.GroupReadyGap+model.Param.CreateGroupInterval {
				p.tryReqSharePiece(gc)
			} else {
				invalidDummyGroups = append(invalidDummyGroups, gHash)
			}
		}
		return true
	})
	for _, groupHash := range invalidDummyGroups {
		groupCreateLogger.Debugf("DissolveGroupNet dummyGroup from joiningGroups gHash %v", groupHash.ShortS())
		//quit group net.group hash
		p.NetServer.ReleaseGroupNet(groupHash.Hex())
		p.groupInitContextCache.RemoveContext(groupHash)
	}

	gctx := p.context
	if gctx != nil && gctx.timeout(topHeight) {
		groupCreateLogger.Infof("releaseRoutine:info=%v, elapsed %v. ready timeout.", gctx.String(), utility.GetTime().Sub(gctx.createTime))
		p.removeContext()
	}

	p.forEach(func(ig *groupPubkeyCollector) bool {
		hash := ig.groupInitInfo.GroupHash()
		if ig.groupInitInfo.ReadyTimeout(topHeight) {
			groupCreateLogger.Debugf("remove groupPubkeyCollector, gHash %v", hash.ShortS())
			p.NetServer.ReleaseGroupNet(hash.Hex())
			p.groupSignCollectorMap.Delete(hash.Hex())
		}
		return true
	})

	cleanSignPkReqRecord()
	return
}

func (p *groupCreateProcessor) tryReqSharePiece(gc *groupInitContext) {
	waitPieceIds := make([]string, 0)
	waitIds := make([]groupsig.ID, 0)
	for _, mem := range gc.groupInitInfo.GroupMembers {
		if !gc.nodeInfo.hasSharePiece(mem) {
			waitPieceIds = append(waitPieceIds, mem.ShortS())
			waitIds = append(waitIds, mem)
		}
	}

	msg := &model.ReqSharePieceMessage{
		GroupHash: gc.groupInitInfo.GroupHash(),
	}
	groupCreateLogger.Infof("reqSharePieceRoutine:req size %v, ghash=%v", len(waitIds), gc.groupInitInfo.GroupHash().ShortS())
	if signInfo, ok := model.NewSignInfo(p.minerInfo.SecKey, p.minerInfo.ID, msg); ok {
		msg.SignInfo = signInfo
		for _, receiver := range waitIds {
			groupCreateLogger.Infof("reqSharePieceRoutine:req share piece msg from %v, ghash=%v", receiver, gc.groupInitInfo.GroupHash().ShortS())
			p.NetServer.ReqSharePiece(msg, receiver)
		}
	} else {
		groupCreateLogger.Infof("gen req sharepiece sign fail, ski=%v %v", p.minerInfo.ID.ShortS(), p.minerInfo.SecKey.ShortS())
	}

}
