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
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/utility"
	lru "github.com/hashicorp/golang-lru"
	"sync"
	"sync/atomic"
	"time"
)

const (
	InitFail = -1
	Initing  = 0
	// InitSuccess initialization successful, group public key generation
	InitSuccess = 1
)

const (
	// GisInit means the group is in its original state (knowing who is a group,
	// but the group public key and group ID have not yet been generated)
	GisInit int32 = iota

	// GisSendSharePiece Sent sharepiece
	GisSendSharePiece

	// GisSendSignPk sent my own signature public key
	GisSendSignPk

	// GisSendInited means group public key and ID have been generated, will casting
	GisSendInited

	// GisGroupInitDone means the group has been initialized and has been add on chain
	GisGroupInitDone
)

// GroupContext is the group consensus context, and the verification determines
// whether a message comes from within the group.
//
// Determine if a message is legal and verify in the outer layer
type groupInitContext struct {
	groupInitInfo *model.GroupInitInfo // Group initialization information (specified by the parent group)
	nodeInfo      *groupNodeInfo       // Group node information (for initializing groups of public and signed private keys)

	status        int32 // Group initialization state
	candidates    []groupsig.ID
	sharePieceMap map[string]model.SharePiece
	createTime    time.Time
}

// CreateGroupContextWithRawMessage creates a GroupContext structure from
// a group initialization message
func newGroupInitContext(groupInitInfo *model.GroupInitInfo, candidates []groupsig.ID, minerInfo *model.SelfMinerInfo) *groupInitContext {
	for k, v := range groupInitInfo.GroupMembers {
		if !v.IsValid() {
			groupCreateLogger.Debug("NewGroupInitContext ID failed! index=%v, id=%v.\n", k, v.GetHexString())
			return nil
		}
	}
	context := new(groupInitContext)
	context.createTime = utility.GetTime()
	context.status = GisInit
	context.groupInitInfo = groupInitInfo
	context.candidates = candidates

	context.nodeInfo = NewGroupNodeInfo(minerInfo, groupInitInfo.GroupHash(), len(groupInitInfo.GroupMembers))
	return context
}

// GenSharePieces generate secret sharing sent to members of the group: si = F(IDi)
func (context *groupInitContext) GenSharePieces() map[string]model.SharePiece {
	shares := make(map[string]model.SharePiece, 0)
	secs := context.nodeInfo.genSharePiece(context.groupInitInfo.GroupMembers)
	var piece model.SharePiece
	piece.Pub = context.nodeInfo.getSeedPubKey()
	for k, v := range secs {
		piece.Share = v
		shares[k] = piece
	}
	context.sharePieceMap = shares
	return shares
}

// Return -1 is abnormal, return 0 is normal, return 1 is the private key
// of the aggregated group member (used for signing)
func (context *groupInitContext) HandleSharePiece(id groupsig.ID, share *model.SharePiece) int {
	result := context.nodeInfo.handleSharePiece(id, share)
	return result
}

func (context *groupInitContext) GetGroupStatus() int32 {
	return atomic.LoadInt32(&context.status)
}

// getMembers
func (context *groupInitContext) getGroupMembers() []groupsig.ID {
	return context.groupInitInfo.GroupMembers
}

func (context *groupInitContext) MemExist(id groupsig.ID) bool {
	return context.groupInitInfo.MemberExists(id)
}

// StatusTransfrom
func (context *groupInitContext) TransformStatus(from, to int32) bool {
	return atomic.CompareAndSwapInt32(&context.status, from, to)
}

func (context *groupInitContext) generateMemberMask() (mask []byte) {
	mask = make([]byte, (len(context.candidates)+7)/8)

	for i, id := range context.candidates {
		b := mask[i/8]
		if context.MemExist(id) {
			b |= 1 << byte(i%8)
			mask[i/8] = b
		}
	}
	return
}

// JoiningGroups
type groupInitContextCache struct {
	cache *lru.Cache //key= groupHash,value = groupInitContext
}

func newGroupInitContextCache() groupInitContextCache {
	return groupInitContextCache{
		cache: common.CreateLRUCache(100),
	}
}

// ConfirmGroupFromRaw
func (groupInitContextCache *groupInitContextCache) GetOrNewContext(groupInitInfo *model.GroupInitInfo, candidates []groupsig.ID, mi *model.SelfMinerInfo) *groupInitContext {
	groupHash := groupInitInfo.GroupHash()
	v := groupInitContextCache.GetContext(groupHash)
	if v != nil {
		status := v.GetGroupStatus()
		groupCreateLogger.Debugf("found Initing group context, status=%v...\n", status)
		return v
	}

	groupCreateLogger.Debug("create new Initing group context\n")
	v = newGroupInitContext(groupInitInfo, candidates, mi)
	if v != nil {
		groupInitContextCache.cache.Add(groupHash.Hex(), v)
	}
	return v
}

// GetGroup
func (groupInitContextCache *groupInitContextCache) GetContext(groupHash common.Hash) *groupInitContext {
	if v, ok := groupInitContextCache.cache.Get(groupHash.Hex()); ok {
		return v.(*groupInitContext)
	}
	return nil
}

// Clean
// todo  rename
func (groupInitContextCache *groupInitContextCache) Clean(groupHash common.Hash) {
	gc := groupInitContextCache.GetContext(groupHash)
	if gc != nil && gc.TransformStatus(GisSendInited, GisGroupInitDone) {
	}
}

// RemoveGroup
func (groupInitContextCache *groupInitContextCache) RemoveContext(groupHash common.Hash) {
	groupInitContextCache.cache.Remove(groupHash.Hex())
}

func (groupInitContextCache *groupInitContextCache) forEach(f func(context *groupInitContext) bool) {
	for _, key := range groupInitContextCache.cache.Keys() {
		v, ok := groupInitContextCache.cache.Get(key)
		if !ok {
			continue
		}
		gc := v.(*groupInitContext)
		if !f(gc) {
			break
		}
	}
}

type groupPubkeyCollector struct {
	groupInitInfo *model.GroupInitInfo

	groupPK            groupsig.Pubkey            // output generated group public key
	receivedGroupPKMap map[string]groupsig.Pubkey //key=>id,value=>group pubkey 从中选取正确的组公钥

	threshold int
	// -1, Group initialization failed (timeout or unable to reach consensus, irreversible)
	// 0,Group is initializing
	// 1,Group initialization succeeded
	status int32

	lock sync.RWMutex
}

// createInitedGroup create a group in initialization
func NewGroupPubkeyCollector(groupInitInfo *model.GroupInitInfo) *groupPubkeyCollector {
	threshold := model.Param.GetGroupK(len(groupInitInfo.GroupMembers))
	return &groupPubkeyCollector{
		receivedGroupPKMap: make(map[string]groupsig.Pubkey),
		status:             Initing,
		threshold:          threshold,
		groupInitInfo:      groupInitInfo,
	}
}

// receive
func (collector *groupPubkeyCollector) handleGroupSign(memberId groupsig.ID, groupPubkey groupsig.Pubkey) int32 {
	status := atomic.LoadInt32(&collector.status)
	if status != Initing {
		return status
	}

	collector.lock.Lock()
	defer collector.lock.Unlock()

	collector.receivedGroupPKMap[memberId.GetHexString()] = groupPubkey
	collector.tryGenGroupPubkey()
	return collector.status
}

// receiveSize
func (collector *groupPubkeyCollector) receiveGroupPKCount() int {
	collector.lock.RLock()
	defer collector.lock.RUnlock()

	return len(collector.receivedGroupPKMap)
}

func (collector *groupPubkeyCollector) hasReceived(id groupsig.ID) bool {
	collector.lock.RLock()
	defer collector.lock.RUnlock()

	_, ok := collector.receivedGroupPKMap[id.GetHexString()]
	return ok
}

// convergence
// convergence find out the most received values
func (collector *groupPubkeyCollector) tryGenGroupPubkey() {
	groupCreateLogger.Debugf("GroupPubkeyCollector try gen grouo pubkey, threshold=%v\n", collector.threshold)

	type countData struct {
		count int
		pk    groupsig.Pubkey
	}
	countMap := make(map[string]*countData, 0) //key=> pubkeyStr value=>countData

	// Statistical occurrences
	for _, groupPubkey := range collector.receivedGroupPKMap {
		ps := groupPubkey.GetHexString()
		if k, ok := countMap[ps]; ok {
			k.count++
			countMap[ps] = k
		} else {
			item := &countData{
				count: 1,
				pk:    groupPubkey,
			}
			countMap[ps] = item
		}
	}

	// Find the most elements
	var groupPubkey groupsig.Pubkey
	var maxCnt = utility.MinInt64
	for _, v := range countMap {
		if v.count > maxCnt {
			maxCnt = v.count
			groupPubkey = v.pk
		}
	}

	if maxCnt >= collector.threshold && atomic.CompareAndSwapInt32(&collector.status, Initing, InitSuccess) {
		groupCreateLogger.Debugf("Gen group pubkey! gproupPK=%v, count=%v.\n", groupPubkey.ShortS(), maxCnt)
		collector.groupPK = groupPubkey
	}
}

// getInitedGroup
func (p *groupCreateProcessor) getGroupPubkeyCollector(groupHash common.Hash) *groupPubkeyCollector {
	if v, ok := p.groupSignCollectorMap.Load(groupHash.Hex()); ok {
		return v.(*groupPubkeyCollector)
	}
	return nil
}

// addInitedGroup
func (p *groupCreateProcessor) addGroupPubkeyCollector(collector *groupPubkeyCollector) *groupPubkeyCollector {
	v, _ := p.groupSignCollectorMap.LoadOrStore(collector.groupInitInfo.GroupHash().Hex(), collector)
	return v.(*groupPubkeyCollector)
}

// removeInitedGroup
func (p *groupCreateProcessor) removeGroupPubkeyCollector(groupHash common.Hash) {
	p.groupSignCollectorMap.Delete(groupHash.Hex())
}

func (p *groupCreateProcessor) forEach(f func(ig *groupPubkeyCollector) bool) {
	p.groupSignCollectorMap.Range(func(key, value interface{}) bool {
		g := value.(*groupPubkeyCollector)
		return f(g)
	})
}
