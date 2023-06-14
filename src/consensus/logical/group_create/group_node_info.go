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
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package group_create

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/base"
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"com.tuntun.rocket/node/src/consensus/model"
	"sync"
)

type groupNodeInfo struct {
	secretSeed         base.Rand
	groupMemberNum     int       // Number of group members
	receivedSharePiece map[string]model.SharePiece

	minerSignSeckey groupsig.Seckey // Output: miner signature private key (aggregated by secret shared receiving pool)
	groupPubKey     groupsig.Pubkey // Output: group public key (aggregated by the miner's signature public key receiving pool)

	lock sync.RWMutex
}

// InitForMiner+InitForGroup
func NewGroupNodeInfo(mi *model.SelfMinerInfo, groupHash common.Hash, groupMemberNum int) *groupNodeInfo {
	var nodeInfo = groupNodeInfo{}
	nodeInfo.secretSeed = mi.GenSecretForGroup(groupHash) // Generate a private seed for the group
	nodeInfo.groupMemberNum = groupMemberNum
	nodeInfo.receivedSharePiece = make(map[string]model.SharePiece)

	nodeInfo.minerSignSeckey = groupsig.Seckey{} // initialization
	nodeInfo.groupPubKey = groupsig.Pubkey{}
	return &nodeInfo
}

// GenSharePiece generate secret sharing for all members of the group
func (nodeInfo *groupNodeInfo) genSharePiece(mems []groupsig.ID) map[string]groupsig.Seckey {
	shares := make(map[string]groupsig.Seckey)
	// How many thresholds are there, how many keys are generated
	secs := nodeInfo.genSecKeyList(nodeInfo.threshold())
	// How many members, how many shared secrets are generated
	for _, id := range mems {
		shares[id.GetHexString()] = *groupsig.ShareSeckey(secs, id)
	}
	return shares
}

// SetInitPiece
// Receiving secret sharing,
// Returns:
//
//	0: normal reception
//	-1: exception
//	1: complete signature private key aggregation and group public key aggregation
func (nodeInfo *groupNodeInfo) handleSharePiece(id groupsig.ID, share *model.SharePiece) int {
	nodeInfo.lock.Lock()
	defer nodeInfo.lock.Unlock()
	groupCreateLogger.Debugf("HandleSharePiece: sender=%v, share=%v, pub=%v...\n", id.ShortS(), share.Share.ShortS(), share.Pub.ShortS())

	if _, ok := nodeInfo.receivedSharePiece[id.GetHexString()]; !ok {
		// Did not receive this member message
		nodeInfo.receivedSharePiece[id.GetHexString()] = *share
	} else {
		// Received this member message
		return -1
	}
	// Has received secret sharing from all members of the group
	if nodeInfo.gotAllSharePiece() {
		if nodeInfo.aggregateKeys() {
			return 1
		}
		return -1
	}
	return 0
}

// hasPiece
func (nodeInfo *groupNodeInfo) hasSharePiece(id groupsig.ID) bool {
	nodeInfo.lock.RLock()
	defer nodeInfo.lock.RUnlock()
	_, ok := nodeInfo.receivedSharePiece[id.GetHexString()]
	return ok
}

// GetGroupPubKey get group public key (valid after secret exchange)
func (nodeInfo *groupNodeInfo) getGroupPubKey() groupsig.Pubkey {
	return nodeInfo.groupPubKey
}

// getSignSecKey get the signature private key (this function
// is not available in the official version)
func (nodeInfo *groupNodeInfo) getSignSecKey() groupsig.Seckey {
	return nodeInfo.minerSignSeckey
}

// GetSeedPubKey obtain a private key (related to the group)
func (nodeInfo *groupNodeInfo) getSeedPubKey() groupsig.Pubkey {
	return *groupsig.GeneratePubkey(nodeInfo.genSeedSecKey())
}

func (nodeInfo *groupNodeInfo) threshold() int {
	return model.Param.GetGroupK(nodeInfo.groupMemberNum)
}

// GenSecKey generate a private key list for a group
// threshold : threshold number
func (nodeInfo *groupNodeInfo) genSecKeyList(threshold int) []groupsig.Seckey {
	secs := make([]groupsig.Seckey, threshold)
	for i := 0; i < threshold; i++ {
		secs[i] = *groupsig.NewSeckeyFromRand(nodeInfo.secretSeed.Deri(i))
	}
	return secs
}

func (nodeInfo *groupNodeInfo) gotAllSharePiece() bool {
	return nodeInfo.receivedSharePieceCount() == nodeInfo.groupMemberNum
}

// GetSize
func (nodeInfo *groupNodeInfo) receivedSharePieceCount() int {
	return len(nodeInfo.receivedSharePiece)
}

// beingValidMiner become an effective miner
func (nodeInfo *groupNodeInfo) aggregateKeys() bool {
	if !nodeInfo.groupPubKey.IsValid() || !nodeInfo.minerSignSeckey.IsValid() {
		// Generate group public key
		nodeInfo.groupPubKey = *nodeInfo.genGroupPubKey()
		// Generate miner signature private key
		nodeInfo.minerSignSeckey = *nodeInfo.genMinerSignSecKey()
	}
	return nodeInfo.groupPubKey.IsValid() && nodeInfo.minerSignSeckey.IsValid()
}

// GenMinerSignSecKey generate miner signature private key
func (nodeInfo *groupNodeInfo) genMinerSignSecKey() *groupsig.Seckey {
	shares := make([]groupsig.Seckey, 0)
	for _, v := range nodeInfo.receivedSharePiece {
		shares = append(shares, v.Share)
	}
	sk := groupsig.AggregateSeckeys(shares)
	return sk
}

// GenGroupPubKey generate group public key
func (nodeInfo *groupNodeInfo) genGroupPubKey() *groupsig.Pubkey {
	pubs := make([]groupsig.Pubkey, 0)
	for _, v := range nodeInfo.receivedSharePiece {
		pubs = append(pubs, v.Pub)
	}
	gpk := groupsig.AggregatePubkeys(pubs)
	return gpk
}

// GenSecKey generate a  private key for a group
func (nodeInfo *groupNodeInfo) genSeedSecKey() groupsig.Seckey {
	return *groupsig.NewSeckeyFromRand(nodeInfo.secretSeed.Deri(0))
}
