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
	"math"
)

const (
	MAX_GROUP_BLOCK_TIME   int = 2
	MAX_WAIT_BLOCK_TIME    int = 0
	CONSENSUS_VERSION          = 1
	MAX_UNKNOWN_BLOCKS         = 5
	GROUP_INIT_MAX_SECONDS     = 60 * 60 * 24

	SSSS_THRESHOLD       int = 51
	GROUP_MAX_MEMBERS    int = 10
	GROUP_MIN_MEMBERS    int = 3
	CANDIDATES_MIN_RATIO     = 1

	Group_Wait_Pong_Gap   = common.Group_Create_Gap + common.EPOCH*2
	GROUP_Ready_GAP       = common.Group_Create_Gap + common.EPOCH*6
	Group_Create_Interval = common.EPOCH * 10
)

type ConsensusParam struct {
	GroupMemberMax      int
	GroupMemberMin      int
	MaxQN               int
	SSSSThreshold       int
	MaxGroupCastTime    int
	MaxWaitBlockTime    int
	MaxFutureBlock      int
	GroupInitMaxSeconds int
	Epoch               uint64
	CreateGroupInterval uint64
	MinerMaxJoinGroup   int
	CandidatesMinRatio  int
	GroupReadyGap       uint64
	GroupWorkGap        uint64
	GroupworkDuration   uint64
	GroupCreateGap      uint64
	GroupWaitPongGap    uint64

	PotentialProposal      uint64
	PotentialProposalMax   uint64
	PotentialProposalIndex int

	ProposalBonus uint64
	PackBonus     uint64
	VerifyBonus   uint64

	MaxSlotSize int
}

var Param ConsensusParam

func InitParam(cc common.SectionConfManager) {
	Param = ConsensusParam{
		GroupMemberMax:      cc.GetInt("group_member_max", GROUP_MAX_MEMBERS),
		GroupMemberMin:      cc.GetInt("group_member_min", GROUP_MIN_MEMBERS),
		SSSSThreshold:       SSSS_THRESHOLD,
		MaxWaitBlockTime:    cc.GetInt("max_wait_block_time", MAX_WAIT_BLOCK_TIME),
		MaxGroupCastTime:    cc.GetInt("max_group_cast_time", MAX_GROUP_BLOCK_TIME),
		MaxQN:               5,
		MaxFutureBlock:      MAX_UNKNOWN_BLOCKS,
		GroupInitMaxSeconds: GROUP_INIT_MAX_SECONDS,
		Epoch:               uint64(cc.GetInt("epoch", common.EPOCH)),
		CandidatesMinRatio:  cc.GetInt("candidates_min_ratio", CANDIDATES_MIN_RATIO),
		GroupReadyGap:       uint64(cc.GetInt("group_ready_gap", GROUP_Ready_GAP)),
		//EffectGapAfterApply: EPOCH,
		PotentialProposal:      3,
		PotentialProposalMax:   5,
		PotentialProposalIndex: 20,

		CreateGroupInterval: uint64(Group_Create_Interval),
		GroupCreateGap:      uint64(common.Group_Create_Gap),
		GroupWaitPongGap:    uint64(Group_Wait_Pong_Gap),
	}
}

func (p *ConsensusParam) GetGroupK(max int) int {
	return int(math.Ceil(float64(max*p.SSSSThreshold) / 100))
}

func (p *ConsensusParam) IsGroupMemberCountLegal(cnt int) bool {
	return p.GroupMemberMin <= cnt && cnt <= p.GroupMemberMax
}
func (p *ConsensusParam) CreateGroupMinCandidates() int {
	return p.GroupMemberMin * p.CandidatesMinRatio
}

func (p *ConsensusParam) CreateGroupMemberCount(availCandidates int) int {
	cnt := int(math.Ceil(float64(availCandidates / p.CandidatesMinRatio)))
	if cnt > p.GroupMemberMax {
		cnt = p.GroupMemberMax
	} else if cnt < p.GroupMemberMin {
		cnt = 0
	}
	return cnt
}
