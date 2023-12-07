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
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package model

import (
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/middleware/types"
)

// GroupInfo is static group structure (joined to GlobalGroups after
// the group is created and add-on-chain successfully)
type GroupInfo struct {
	GroupID       groupsig.ID     // Group ID (can be generated by the group public key)
	GroupPK       groupsig.Pubkey // Group public key
	GroupInitInfo *GroupInitInfo  // Fixed group info after consensus

	MemberIndexMap map[string]int // Find member information by ID (member ID -> index in members)
	ParentGroupID  groupsig.ID    // Parent Group ID
	PrevGroupID    groupsig.ID    // Previous group id
}

func NewGroupInfo(groupId groupsig.ID, groupPubkey groupsig.Pubkey, groupInitInfo *GroupInitInfo) *GroupInfo {
	groupInfo := &GroupInfo{
		GroupID:       groupId,
		GroupPK:       groupPubkey,
		GroupInitInfo: groupInitInfo,
		ParentGroupID: groupInitInfo.ParentGroupID(),
		PrevGroupID:   groupInitInfo.PreGroupID(),
	}
	groupInfo.BuildMemberIndex()
	return groupInfo
}

// newSGIFromCoreGroup convert the group info from chain to the StaticGroupInfo
func ConvertToGroupInfo(coreGroup *types.Group) *GroupInfo {
	groupHeader := coreGroup.Header
	mems := make([]groupsig.ID, len(coreGroup.Members))
	for i, mem := range coreGroup.Members {
		mems[i] = groupsig.DeserializeID(mem)
	}
	groupInitInfo := &GroupInitInfo{
		GroupHeader:     groupHeader,
		ParentGroupSign: *groupsig.DeserializeSign(coreGroup.Signature),
		GroupMembers:    mems,
	}
	groupInfo := &GroupInfo{
		GroupID:       groupsig.DeserializeID(coreGroup.Id),
		GroupPK:       groupsig.ByteToPublicKey(coreGroup.PubKey),
		ParentGroupID: groupsig.DeserializeID(groupHeader.Parent),
		PrevGroupID:   groupsig.DeserializeID(groupHeader.PreGroup),
		GroupInitInfo: groupInitInfo,
	}
	groupInfo.BuildMemberIndex()
	return groupInfo
}

// returns the public key of the group
func (groupInfo *GroupInfo) GetGroupPubKey() groupsig.Pubkey {
	return groupInfo.GroupPK
}

// GetMemberCount returns the member count
func (groupInfo *GroupInfo) GetMemberCount() int {
	return groupInfo.GroupInitInfo.MemberSize()
}

func (groupInfo *GroupInfo) GetGroupHeader() *types.GroupHeader {
	return groupInfo.GroupInitInfo.GroupHeader
}

// GetMemberID gets the member id at the specified position
func (groupInfo *GroupInfo) GetMemberID(i int) groupsig.ID {
	var m groupsig.ID
	if i >= 0 && i < len(groupInfo.MemberIndexMap) {
		m = groupInfo.GroupInitInfo.GroupMembers[i]
	}
	return m
}

// GetMembers
// GetMembers returns the member ids of the group
func (groupInfo *GroupInfo) GetGroupMembers() []groupsig.ID {
	return groupInfo.GroupInitInfo.GroupMembers
}

// GetMinerPos
// GetMinerPos get a miner's position in the group
func (groupInfo *GroupInfo) GetMemberPosition(id groupsig.ID) int {
	pos := -1
	if v, ok := groupInfo.MemberIndexMap[id.GetHexString()]; ok {
		pos = v
		// Double verification
		if !groupInfo.GroupInitInfo.GroupMembers[pos].IsEqual(id) {
			panic("double check fail!id=" + id.GetHexString())
		}
	}
	return pos
}

// MemExist check if the specified miner is belong to the group
func (groupInfo *GroupInfo) MemExist(minerId groupsig.ID) bool {
	_, ok := groupInfo.MemberIndexMap[minerId.GetHexString()]
	return ok
}

// CastQualified
// CastQualified check if the group is cast qualified at the specified height
func (groupInfo *GroupInfo) IsEffective(height uint64) bool {
	gh := groupInfo.GetGroupHeader()
	//return IsGroupWorkQualifiedAt(gh, height)
	return gh.WorkHeight <= height && height < gh.DismissHeight
}

// Dismissed
// Dismissed means whether the group has been dismissed
func (groupInfo *GroupInfo) NeedDismiss(height uint64) bool {
	//return isGroupDissmisedAt(sgi.getGroupHeader(), height)
	return groupInfo.GetGroupHeader().DismissHeight <= height
}

func (groupInfo *GroupInfo) GetReadyTimeout(height uint64) bool {
	return groupInfo.GetGroupHeader().CreateHeight+Param.GroupReadyGap <= height
}

func (groupInfo *GroupInfo) BuildMemberIndex() {
	if groupInfo.MemberIndexMap == nil {
		groupInfo.MemberIndexMap = make(map[string]int)
	}
	for index, mem := range groupInfo.GroupInitInfo.GroupMembers {
		groupInfo.MemberIndexMap[mem.GetHexString()] = index
	}
}
