package access

import (
	"fmt"
	"math/big"
	"sync"

	"x/src/consensus/groupsig"
	"x/src/consensus/model"
	"x/src/middleware/types"
	"x/src/core"
	"x/src/common"
)

var groupAccessorInstance *GroupAccessor

//GlobalGroups
// GlobalGroup caches all work-group slices, and as the new group joins, the old group is dismissed,
// and the cache is constantly updated.
//
// Although we can get all the work-groups from the chain, the cache is to speed up the calculations.
type GroupAccessor struct {
	groups        []*model.GroupInfo // The work-group slices
	groupIndexMap map[string]int     // Group index supports quick retrieval of group information through group id(in hex string)

	chain core.GroupChain
	lock  sync.RWMutex
}

//newGlobalGroups
func NewGroupAccessor(chain core.GroupChain) *GroupAccessor {
	if groupAccessorInstance == nil {
		groupAccessorInstance = &GroupAccessor{
			groups:        make([]*model.GroupInfo, 1),
			groupIndexMap: make(map[string]int),
			chain:         chain,
			lock:          sync.RWMutex{},
		}
	}
	return groupAccessorInstance
}

//AddStaticGroup
// AddStaticGroup adda a legal effective group to the cache, which may be a work group currently or in the near future
//
// Consider the group synchronization process, the method may be called concurrently,
// resulting in the order of the groups being out of order
// It has to be processed carefully
func (groupAccessor *GroupAccessor) AddGroupInfo(g *model.GroupInfo) bool {
	groupAccessor.lock.Lock()
	defer groupAccessor.lock.Unlock()

	result := ""
	defer func() {
		logger.Debugf("add group info! group id=%v, hash=%v, beginHeight=%v, result=%v\n", g.GroupID.ShortS(), g.GetGroupHeader().Hash.ShortS(), g.GetGroupHeader().WorkHeight, result)
	}()

	if _, ok := groupAccessor.groupIndexMap[g.GroupID.GetHexString()]; !ok {
		if g.GetGroupHeader().WorkHeight == 0 { // the genesis group
			groupAccessor.groups[0] = g
			groupAccessor.groupIndexMap[g.GroupID.GetHexString()] = 0
			result = "success"
			return true
		}
		if idx, right := groupAccessor.findPos(g); idx >= 0 {
			cnt := len(groupAccessor.groups)
			if idx == cnt {
				groupAccessor.append(g)
				result = "append"
			} else {
				groupAccessor.groups = append(groupAccessor.groups, g)
				for i := cnt; i > idx; i-- {
					groupAccessor.groups[i] = groupAccessor.groups[i-1]
					groupAccessor.groupIndexMap[groupAccessor.groups[i].GroupID.GetHexString()] = i
				}
				groupAccessor.groups[idx] = g
				groupAccessor.groupIndexMap[g.GroupID.GetHexString()] = idx
				result = "insert"
			}
			if right {
				result += "and linked"
			} else {
				result += "but not linked"
			}
			return true
		}
		result = "can't find insert pos"
	} else {
		result = "already exist this group, ignored"
	}
	return false
}

// GetGroupByID returns the group info of the specified id.
func (groupAccessor *GroupAccessor) GetGroupByID(id groupsig.ID) (g *model.GroupInfo, err error) {
	if g, err = groupAccessor.GetGroupFromCache(id); err != nil {
		return
	}
	if g == nil {
		chainGroup := groupAccessor.chain.GetGroupById(id.Serialize())
		if chainGroup != nil {
			g = model.ConvertToGroupInfo(chainGroup)
		}
	}
	if g == nil {
		logger.Debugf("GetGroupByID nil, groupId=%v\n", id.ShortS())
		for _, g := range groupAccessor.groups {
			logger.Debugf("cached groupId %v\n", g.GroupID.ShortS())
		}
		g = &model.GroupInfo{}
	}
	return
}

// IsGroupMember check if a user is a member of a group
func (groupAccessor *GroupAccessor) IsGroupMember(groupId groupsig.ID, minerId groupsig.ID) bool {
	g, err := groupAccessor.GetGroupByID(groupId)
	if err == nil {
		return g.MemExist(minerId)
	}
	return false
}

//GetGroupSize
func (groupAccessor *GroupAccessor) GroupSize() int {
	groupAccessor.lock.RLock()
	defer groupAccessor.lock.RUnlock()
	return len(groupAccessor.groups)
}

//DismissGroups
func (groupAccessor *GroupAccessor) GetDismissGroups(height uint64) []*model.GroupInfo {
	groupAccessor.lock.RLock()
	defer groupAccessor.lock.RUnlock()

	ids := make([]*model.GroupInfo, 0)
	for _, g := range groupAccessor.groups {
		if g == nil {
			continue
		}
		if g.NeedDismiss(height) {
			ids = append(ids, g)
		}
	}
	return ids
}

//GetEffective
// SelectNextGroupFromCache determines the next verification group through the cached work-group slices according to the previous random number.
// The result is random and certain
func (groupAccessor *GroupAccessor) SelectVerifyGroupFromCache(hash common.Hash, height uint64) (groupsig.ID, error) {
	qualifiedGS := groupAccessor.GetEffectiveGroups(height)

	var ga groupsig.ID

	gids := make([]string, 0)
	for _, g := range qualifiedGS {
		gids = append(gids, g.GroupID.ShortS())
	}

	if hash.Big().BitLen() > 0 && len(qualifiedGS) > 0 {
		index := groupAccessor.selectIndex(len(qualifiedGS), hash)
		ga = qualifiedGS[index].GroupID
		logger.Debugf("SelectVerifyGroupFromCache! Height:%v,Qualified groups:%v, index:%v\n", height, gids, index)
		return ga, nil
	}
	return ga, fmt.Errorf("SelectVerifyGroupFromCache failed, hash %v, qualified groups %v", hash.ShortS(), gids)
}

//SelectNextGroupFromChain
// SelectNextGroupFromChain determines the next verification group through the chained work-groups according to the previous random number.
// The result is random and certain, and mostly should be the same as method SelectNextGroupFromCache
//
// This method can be used to compensate when the result of the calculation through the cache(method SelectNextGroupFromCache)
// does not match the expectation
func (groupAccessor *GroupAccessor) SelectVerifyGroupFromChain(hash common.Hash, height uint64) (groupsig.ID, error) {
	quaulifiedGS := groupAccessor.GetCastQualifiedGroupFromChains(height)
	idshort := make([]string, len(quaulifiedGS))
	for idx, g := range quaulifiedGS {
		idshort[idx] = groupsig.DeserializeID(g.Id).ShortS()
	}

	var ga groupsig.ID
	if hash.Big().BitLen() > 0 && len(quaulifiedGS) > 0 {
		index := groupAccessor.selectIndex(len(quaulifiedGS), hash)
		ga = groupsig.DeserializeID(quaulifiedGS[index].Id)
		logger.Debugf("SelectVerifyGroupFromChain! Height:%v,qualified groups %v, index %v\n", height, idshort, index)
		return ga, nil
	}
	return ga, fmt.Errorf("SelectVerifyGroupFromChain failed, arg error")
}

//GetCastQualifiedGroups
// GetCastQualifiedGroups gets all work-groups at the specified height
func (groupAccessor *GroupAccessor) GetEffectiveGroups(height uint64) []*model.GroupInfo {
	groupAccessor.lock.RLock()
	defer groupAccessor.lock.RUnlock()

	gs := make([]*model.GroupInfo, 0)
	for _, g := range groupAccessor.groups {
		if g == nil {
			continue
		}
		if g.IsEffective(height) {
			gs = append(gs, g)
		}
	}
	return gs
}

func (groupAccessor *GroupAccessor) GetGenesisGroup() *model.GroupInfo {
	if groupAccessor.GroupSize() == 0 {
		return nil
	}
	g := groupAccessor.groups[0]
	if g.GroupInitInfo.GroupHeader.WorkHeight != 0 {
		panic("genesis group error")
	}
	return g
}

func (groupAccessor *GroupAccessor) findPos(g *model.GroupInfo) (idx int, right bool) {
	cnt := len(groupAccessor.groups)
	if cnt == 1 {
		return 1, true
	}
	last := groupAccessor.lastGroup()

	// Just connected to the last one, most of the time this is the case
	if g.PrevGroupID.IsEqual(last.GroupID) {
		return cnt, true
	}

	// Belong to the group that follows, append to the end
	if g.GetGroupHeader().WorkHeight > last.GetGroupHeader().WorkHeight {
		return cnt, false
	}
	for i := 1; i < cnt; i++ {
		if groupAccessor.groups[i].GetGroupHeader().WorkHeight > g.GetGroupHeader().WorkHeight {
			return i, g.GroupID.IsEqual(groupAccessor.groups[i].PrevGroupID) && (i == 1 || g.PrevGroupID.IsEqual(groupAccessor.groups[i-1].GroupID))
		}
	}
	return -1, false
}

func (groupAccessor *GroupAccessor) lastGroup() *model.GroupInfo {
	return groupAccessor.groups[len(groupAccessor.groups)-1]
}

func (groupAccessor *GroupAccessor) append(g *model.GroupInfo) bool {
	groupAccessor.groups = append(groupAccessor.groups, g)
	groupAccessor.groupIndexMap[g.GroupID.GetHexString()] = len(groupAccessor.groups) - 1
	return true
}

func (groupAccessor *GroupAccessor) getGroupByIndex(i int) (g *model.GroupInfo, err error) {
	if i >= 0 && i < len(groupAccessor.groups) {
		g = groupAccessor.groups[i]
	} else {
		err = fmt.Errorf("out of range")
	}
	return
}

func (groupAccessor *GroupAccessor) GetGroupFromCache(id groupsig.ID) (g *model.GroupInfo, err error) {
	groupAccessor.lock.RLock()
	defer groupAccessor.lock.RUnlock()

	index, ok := groupAccessor.groupIndexMap[id.GetHexString()]
	if ok {
		g, err = groupAccessor.getGroupByIndex(index)
		if !g.GroupID.IsEqual(id) {
			panic("ggIndex error")
		}
	}
	return
}

func (groupAccessor *GroupAccessor) selectIndex(num int, hash common.Hash) int64 {
	value := hash.Big()
	index := value.Mod(value, big.NewInt(int64(num)))
	return index.Int64()
}

func (groupAccessor *GroupAccessor) GetCastQualifiedGroupFromChains(height uint64) []*types.Group {
	iter := groupAccessor.chain.Iterator()
	groups := make([]*types.Group, 0)
	for g := iter.Current(); g != nil; g = iter.MovePre() {
		if isGroupWorkQualifiedAt(g.Header, height) {
			groups = append(groups, g)
		} else if isGroupDissmisedAt(g.Header, height) {
			g = groupAccessor.chain.GetGroupByHeight(0)
			groups = append(groups, g)
			break
		}
	}
	n := len(groups)
	reverseGroups := make([]*types.Group, n)
	for i := 0; i < n; i++ {
		reverseGroups[n-i-1] = groups[i]
	}
	return reverseGroups
}

func (groupAccessor *GroupAccessor) RemoveGroupsFromCache(gids []groupsig.ID) {
	if len(gids) == 0 {
		return
	}
	removeIDMap := make(map[string]bool)
	for _, gid := range gids {
		removeIDMap[gid.GetHexString()] = true
	}
	newGS := make([]*model.GroupInfo, 0)
	for _, g := range groupAccessor.groups {
		if g == nil {
			continue
		}
		if _, ok := removeIDMap[g.GroupID.GetHexString()]; !ok {
			newGS = append(newGS, g)
		}
	}
	indexMap := make(map[string]int)
	for idx, g := range newGS {
		indexMap[g.GroupID.GetHexString()] = idx
	}

	groupAccessor.lock.Lock()
	defer groupAccessor.lock.Unlock()

	groupAccessor.groups = newGS
	groupAccessor.groupIndexMap = indexMap
}

func isGroupDissmisedAt(gh *types.GroupHeader, h uint64) bool {
	return gh.DismissHeight <= h
}

// IsGroupWorkQualifiedAt check if the specified group is work qualified at the given height
func isGroupWorkQualifiedAt(gh *types.GroupHeader, h uint64) bool {
	return !isGroupDissmisedAt(gh, h) && gh.WorkHeight <= h
}

// GetAvailableGroups gets all valid groups at a given height, including those can work currently or in the near future
func (groupAccessor *GroupAccessor) GetAvailableGroups(height uint64) []*model.GroupInfo {
	groupAccessor.lock.RLock()
	defer groupAccessor.lock.RUnlock()

	gs := make([]*model.GroupInfo, 0)
	for _, g := range groupAccessor.groups {
		if g == nil {
			continue
		}
		if !g.NeedDismiss(height) {
			gs = append(gs, g)
		}
	}
	return gs
}
