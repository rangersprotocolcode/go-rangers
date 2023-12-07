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
	"bytes"
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus/base"
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/middleware/types"
	"time"
)

type GroupInitInfo struct {
	GroupHeader     *types.GroupHeader
	ParentGroupSign groupsig.Signature //父亲组签名
	GroupMembers    []groupsig.ID
}

func (gi *GroupInitInfo) MemberSize() int {
	return len(gi.GroupMembers)
}

func (gi *GroupInitInfo) GroupHash() common.Hash {
	return gi.GroupHeader.Hash
}

func (gi *GroupInitInfo) ParentGroupID() groupsig.ID {
	return groupsig.DeserializeID(gi.GroupHeader.Parent)
}

func (gi *GroupInitInfo) PreGroupID() groupsig.ID {
	return groupsig.DeserializeID(gi.GroupHeader.PreGroup)
}

func (gi *GroupInitInfo) CreateHeight() uint64 {
	return gi.GroupHeader.CreateHeight
}

// GenMemberRootByIds
func GenGroupMemberRoot(ids []groupsig.ID) common.Hash {
	data := bytes.Buffer{}
	for _, m := range ids {
		data.Write(m.Serialize())
	}
	return base.Data2CommonHash(data.Bytes())
}

func (gi *GroupInitInfo) ReadyTimeout(height uint64) bool {
	return gi.GroupHeader.CreateHeight+Param.GroupReadyGap <= height
}
func (gi *GroupInitInfo) MemberExists(id groupsig.ID) bool {
	for _, mem := range gi.GroupMembers {
		if mem.IsEqual(id) {
			return true
		}
	}
	return false
}

type SharePiece struct {
	Share groupsig.Seckey
	Pub   groupsig.Pubkey
}

func (piece SharePiece) IsValid() bool {
	return piece.Share.IsValid() && piece.Pub.IsValid()
}

func (piece SharePiece) IsEqual(rhs SharePiece) bool {
	return piece.Share.IsEqual(rhs.Share) && piece.Pub.IsEqual(rhs.Pub)
}

type GroupMinerID struct {
	Gid groupsig.ID
	Uid groupsig.ID
}

func NewGroupMinerID(gid groupsig.ID, uid groupsig.ID) *GroupMinerID {
	return &GroupMinerID{
		Gid: gid,
		Uid: uid,
	}
}

func (id GroupMinerID) IsValid() bool {
	return id.Gid.IsValid() && id.Uid.IsValid()
}

type CastGroupSummary struct {
	PreHash     common.Hash
	PreTime     time.Time
	BlockHeight uint64
	GroupID     groupsig.ID
	Castor      groupsig.ID
	CastorPos   int32
}

type BlockProposalDetail struct {
	BH     *types.BlockHeader
	Proves []common.Hash
}
