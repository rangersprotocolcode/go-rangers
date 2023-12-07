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
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"sync"
)

// JoinedGroup
// JoinedGroup stores group-related infos the current node joins in.
// Note that, nodes outside the group don't care the infos
// 加入的组的信息和自身在组内的信息
type JoinedGroupInfo struct {
	GroupHash common.Hash
	GroupID   groupsig.ID     // Group ID
	GroupPK   groupsig.Pubkey // Group public key (backup, which can be got from the global group)

	SignSecKey          groupsig.Seckey            // Miner signature private key related to the group
	MemberSignPubkeyMap map[string]groupsig.Pubkey // Group related public keys of all members

	lock sync.RWMutex
}

func NewJoindGroupInfo(signSeckey groupsig.Seckey, groupPubkey groupsig.Pubkey, groupHash common.Hash) *JoinedGroupInfo {
	joinedGroup := &JoinedGroupInfo{
		GroupHash:           groupHash,
		GroupPK:             groupPubkey,
		GroupID:             *groupsig.NewIDFromPubkey(groupPubkey),
		SignSecKey:          signSeckey,
		MemberSignPubkeyMap: make(map[string]groupsig.Pubkey, 0),
	}
	return joinedGroup
}

func (joinedGroupInfo *JoinedGroupInfo) MemberSignPKNum() int {
	joinedGroupInfo.lock.RLock()
	defer joinedGroupInfo.lock.RUnlock()

	return len(joinedGroupInfo.MemberSignPubkeyMap)
}

// getMemberMap
func (joinedGroupInfo *JoinedGroupInfo) GetMemberPKs() map[string]groupsig.Pubkey {
	joinedGroupInfo.lock.RLock()
	defer joinedGroupInfo.lock.RUnlock()

	m := make(map[string]groupsig.Pubkey, 0)
	for key, pk := range joinedGroupInfo.MemberSignPubkeyMap {
		m[key] = pk
	}
	return m
}

func (joinedGroupInfo *JoinedGroupInfo) AddMemberSignPK(memberId groupsig.ID, signPK groupsig.Pubkey) {
	joinedGroupInfo.lock.Lock()
	defer joinedGroupInfo.lock.Unlock()

	joinedGroupInfo.MemberSignPubkeyMap[memberId.GetHexString()] = signPK
}

// getMemSignPK get the signature public key of a member of the group
func (joinedGroupInfo *JoinedGroupInfo) GetMemberSignPK(memberId groupsig.ID) (pk groupsig.Pubkey, ok bool) {
	joinedGroupInfo.lock.RLock()
	defer joinedGroupInfo.lock.RUnlock()

	pk, ok = joinedGroupInfo.MemberSignPubkeyMap[memberId.GetHexString()]
	return
}
