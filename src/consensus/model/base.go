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

// ConsensusGroupInitInfo
// 组初始化信息
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

// ParentID
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

// 组内秘密分享消息结构
type SharePiece struct {
	Share groupsig.Seckey //秘密共享
	Pub   groupsig.Pubkey //矿工（组私密）公钥
}

func (piece SharePiece) IsValid() bool {
	return piece.Share.IsValid() && piece.Pub.IsValid()
}

func (piece SharePiece) IsEqual(rhs SharePiece) bool {
	return piece.Share.IsEqual(rhs.Share) && piece.Pub.IsEqual(rhs.Pub)
}

// 矿工ID信息
type GroupMinerID struct {
	Gid groupsig.ID //组ID
	Uid groupsig.ID //成员ID
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

// 成为当前铸块组共识摘要
type CastGroupSummary struct {
	PreHash     common.Hash //上一块哈希
	PreTime     time.Time   //上一块完成时间
	BlockHeight uint64      //当前铸块高度
	GroupID     groupsig.ID //当前组ID
	Castor      groupsig.ID
	CastorPos   int32
}

type BlockProposalDetail struct {
	BH     *types.BlockHeader
	Proves []common.Hash
}

////数据签名结构
//type SignData struct {
//	Version    int32
//	DataHash   common.Hash        //哈希值
//	DataSign   groupsig.Signature //签名
//	SignMember groupsig.ID        //用户ID或组ID，看消息类型
//}
//
//func (sd SignData) IsEqual(rhs SignData) bool {
//	return sd.DataHash.Str() == rhs.DataHash.Str() && sd.SignMember.IsEqual(rhs.SignMember) && sd.DataSign.IsEqual(rhs.DataSign)
//}
//
//func GenSignData(h common.Hash, id groupsig.ID, sk groupsig.Seckey) SignData {
//	return SignData{
//		DataHash: h,
//		DataSign: groupsig.Sign(sk, h.Bytes()),
//		SignMember: id,
//		Version: common.ConsensusVersion,
//	}
//}
//
//
//func (sd SignData) GetID() groupsig.ID {
//	return sd.SignMember
//}
//
//
////用pk验证签名，验证通过返回true，否则false。
//func (sd SignData) VerifySign(pk groupsig.Pubkey) bool {
//	return groupsig.VerifySig(pk, sd.DataHash.Bytes(), sd.DataSign)
//}

//
////用sk生成签名
//func (sd *SignData) GenSign(sk groupsig.Seckey) bool {
//	b := sk.IsValid()
//	if b {
//		sd.DataSign = groupsig.Sign(sk, sd.DataHash.Bytes())
//	}
//	return b
//}

////是否已有签名数据
//func (sd SignData) HasSign() bool {
//	return sd.DataSign.IsValid() && sd.SignMember.IsValid()
//}

////id->公钥对
//type PubKeyInfo struct {
//	ID groupsig.ID
//	PK groupsig.Pubkey
//}
//
//func NewPubKeyInfo(id groupsig.ID, pk groupsig.Pubkey) PubKeyInfo {
//	return PubKeyInfo{
//		ID:id,
//		PK:pk,
//	}
//}
//
//func (p PubKeyInfo) IsValid() bool {
//	return p.ID.IsValid() && p.PK.IsValid()
//}
//
//func (p PubKeyInfo) GetID() groupsig.ID {
//	return p.ID
//}
//
////id->私钥对
//type SecKeyInfo struct {
//	ID groupsig.ID
//	SK groupsig.Seckey
//}
//
//func NewSecKeyInfo(id groupsig.ID, sk groupsig.Seckey) SecKeyInfo {
//	return SecKeyInfo{
//		ID:id,
//		SK:sk,
//	}
//}
//
//func (s SecKeyInfo) IsValid() bool {
//	return s.ID.IsValid() && s.SK.IsValid()
//}
//
//func (s SecKeyInfo) GetID() groupsig.ID {
//	return s.ID
//}

////组初始化共识摘要
//type ConsensusGroupInitSummary struct {
//	Signature groupsig.Signature //父亲组签名
//	GHeader   *types.GroupHeader
//}
//
//func (gis *ConsensusGroupInitSummary) GetHash() common.Hash {
//	return gis.GHeader.Hash
//}
//
//func (gis *ConsensusGroupInitSummary) ParentID() groupsig.ID {
//	return groupsig.DeserializeId(gis.GHeader.Parent)
//}
//
//func (gis *ConsensusGroupInitSummary) PreGroupID() groupsig.ID {
//	return groupsig.DeserializeId(gis.GHeader.PreGroup)
//}
//
//func (gis *ConsensusGroupInitSummary) CreateHeight() uint64 {
//	return gis.GHeader.CreateHeight
//}
//
//func GenMemberRootByIds(ids []groupsig.ID) common.Hash {
//	data := bytes.Buffer{}
//	for _, m := range ids {
//		data.Write(m.Serialize())
//	}
//	return base.Data2CommonHash(data.Bytes())
//}
//
//func (gis *ConsensusGroupInitSummary) CheckMemberHash(mems []groupsig.ID) bool {
//	return gis.GHeader.MemberRoot == GenMemberRootByIds(mems)
//}
//
//func (gis *ConsensusGroupInitSummary) ReadyTimeout(height uint64) bool {
//	return gis.GHeader.ReadyHeight <= height
//}

//func (gis *GroupInitInfo) GetHash() common.Hash {
//	return gis.GroupHeader.Hash
//}

//map(id->秘密分享)
//type SharePieceMap map[string]SharePiece
