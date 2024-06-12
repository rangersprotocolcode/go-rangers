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

package cli

import (
	"com.tuntun.rangers/node/src/consensus"
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/middleware/types"
	"sort"
)

type SysWorkSummary struct {
	BeginHeight         uint64                `json:"begin_height"`
	ToHeight            uint64                `json:"to_height"`
	GroupSummary        []*GroupVerifySummary `json:"group_summary"`
	AverCastTime        float64               `json:"aver_cast_time"`
	MaxCastTime         float64               `json:"max_cast_time"`
	HeightOfMaxCastTime uint64                `json:"height_of_max_cast_time"`
	JumpRate            float64               `json:"jump_rate"`
	summaryMap          map[string]*GroupVerifySummary
	allGroup            map[string]*types.Group
}

func (s *SysWorkSummary) Len() int {
	return len(s.GroupSummary)
}

func (s *SysWorkSummary) Less(i, j int) bool {
	return s.GroupSummary[i].DissmissHeight < s.GroupSummary[j].DissmissHeight
}

func (s *SysWorkSummary) Swap(i, j int) {
	s.GroupSummary[i], s.GroupSummary[j] = s.GroupSummary[j], s.GroupSummary[i]
}

func (s *SysWorkSummary) sort() {
	if len(s.summaryMap) == 0 {
		return
	}
	tmp := make([]*GroupVerifySummary, 0)
	for _, s := range s.summaryMap {
		tmp = append(tmp, s)
	}
	s.GroupSummary = tmp
	sort.Sort(s)
}

func (s *SysWorkSummary) getGroupSummary(gid groupsig.ID, top uint64, nextSelected bool) *GroupVerifySummary {
	gidStr := gid.GetHexString()
	if v, ok := s.summaryMap[gidStr]; ok {
		return v
	}
	gvs := &GroupVerifySummary{
		Gid:            gidStr,
		LastJumpHeight: make([]uint64, 0),
	}
	g := s.allGroup[gidStr]
	gvs.fillGroupInfo(g, top)
	gvs.NextSelected = nextSelected
	s.summaryMap[gidStr] = gvs

	return gvs
}

type GroupVerifySummary struct {
	Gid            string   `json:"gid"`
	DissmissHeight uint64   `json:"dissmiss_height"`
	Dissmissed     bool     `json:"dissmissed"`
	NumVerify      int      `json:"num_verify"`
	NumJump        int      `json:"num_jump"`
	LastJumpHeight []uint64 `json:"last_jump_height"`
	NextSelected   bool     `json:"next_selected"`
	JumpRate       float64  `json:"jump_rate"`
}

func (s *GroupVerifySummary) addJumpHeight(h uint64) {
	if len(s.LastJumpHeight) < 50 {
		s.LastJumpHeight = append(s.LastJumpHeight, h)
	} else {
		find := false
		for i := 1; i < len(s.LastJumpHeight); i++ {
			if s.LastJumpHeight[i-1] > s.LastJumpHeight[i] {
				s.LastJumpHeight[i] = h
				find = true
				break
			}
		}
		if !find {
			s.LastJumpHeight[0] = h
		}
	}
	s.NumJump += 1
}

func (s *GroupVerifySummary) calJumpRate() {
	s.JumpRate = float64(s.NumJump) / float64(s.NumVerify+s.NumJump)
}

func (s *GroupVerifySummary) fillGroupInfo(g *types.Group, top uint64) {
	if g == nil {
		return
	}
	s.DissmissHeight = g.Header.DismissHeight
	s.Dissmissed = s.DissmissHeight <= top
}

func (api *GtasAPI) DebugJoinGroupInfo(gid string) (*Result, error) {
	jg := consensus.Proc.GetJoinGroupInfo(gid)
	return successResult(jg)
}
