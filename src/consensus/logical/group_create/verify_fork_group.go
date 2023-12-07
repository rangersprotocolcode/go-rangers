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
	"bytes"
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus/base"
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/middleware/types"
	"fmt"
)

func (p *groupCreateProcessor) VerifyGroupForFork(g *types.Group, preGroup *types.Group, parentGroup *types.Group, baseBlock *types.Block) (ok bool, err error) {
	if len(g.Signature) == 0 {
		return false, fmt.Errorf("sign is empty")
	}

	mems := make([]groupsig.ID, len(g.Members))
	for idx, mem := range g.Members {
		mems[idx] = groupsig.DeserializeID(mem)
	}
	gInfo := &model.GroupInitInfo{
		ParentGroupSign: *groupsig.DeserializeSign(g.Signature),
		GroupHeader:     g.Header,
		GroupMembers:    mems,
	}

	groupHeader := gInfo.GroupHeader
	if groupHeader.Hash != groupHeader.GenHash() {
		return false, fmt.Errorf("gh hash error, hash=%v, genHash=%v", groupHeader.Hash.ShortS(), groupHeader.GenHash().ShortS())
	}

	// check if the member count is legal
	if !model.Param.IsGroupMemberCountLegal(len(gInfo.GroupMembers)) {
		return false, fmt.Errorf("group member size error %v(%v-%v)", len(gInfo.GroupMembers), model.Param.GroupMemberMin, model.Param.GroupMemberMax)
	}
	// check if the create height is legal
	if !validateHeight(groupHeader.CreateHeight) {
		return false, fmt.Errorf("cannot create at the height %v", groupHeader.CreateHeight)
	}
	if baseBlock == nil {
		return false, common.ErrCreateBlockNil
	}
	if baseBlock.Header.Height != groupHeader.CreateHeight {
		return false, fmt.Errorf("group base block height diff %v-%v", baseBlock.Header.Height, groupHeader.CreateHeight)
	}
	// The previous group, whether the parent group exists
	if preGroup == nil {
		return false, fmt.Errorf("preGroup is nil, gid=%v", groupsig.DeserializeID(groupHeader.PreGroup).ShortS())
	}
	if parentGroup == nil {
		return false, fmt.Errorf("parentGroup is nil, gid=%v", groupsig.DeserializeID(groupHeader.Parent).ShortS())
	}

	// check if it is the specified parent group
	caledParentGroup, err := p.selectParentGroupForFork(baseBlock.Header, groupHeader.PreGroup)
	if err != nil {
		return false, fmt.Errorf("select parent group err %v", err)
	}
	if !bytes.Equal(caledParentGroup.Id, parentGroup.Id) {
		return false, fmt.Errorf("select parent group not equal, expect %v, recieve %v", common.ToHex(caledParentGroup.Id), common.ToHex(parentGroup.Id))
	}
	parentGroupPubkey := groupsig.ByteToPublicKey(parentGroup.PubKey)

	// check the signature of the parent group
	if !groupsig.VerifySig(parentGroupPubkey, groupHeader.Hash.Bytes(), gInfo.ParentGroupSign) {
		return false, fmt.Errorf("verify parent sign fail")
	}

	// check if the candidates are legal
	enough, candidates := p.selectCandidatesForForkGroup(baseBlock.Header)
	if !enough {
		return false, fmt.Errorf("not enough candidates")
	}
	// Whether the selected member is in the designated candidate
	for _, mem := range gInfo.GroupMembers {
		find := false
		for _, cand := range candidates {
			if mem.IsEqual(cand) {
				find = true
				break
			}
		}
		if !find {
			return false, fmt.Errorf("mem error: %v is not a legal candidate", mem.ShortS())
		}
	}

	gpk := groupsig.ByteToPublicKey(g.PubKey)
	gid := groupsig.NewIDFromPubkey(gpk).Serialize()
	if !bytes.Equal(gid, g.Id) {
		return false, fmt.Errorf("gid error, expect %v, receive %v", gid, g.Id)
	}
	return true, nil
}

// selectParentGroup determine the parent group randomly and the result is deterministic because of the base BlockHeader
// 获取父亲组
func (p *groupCreateProcessor) selectParentGroupForFork(baseBH *types.BlockHeader, preGroupID []byte) (*types.Group, error) {
	//return p.groupAccessor.GetGenesisGroup(), nil
	rand := baseBH.Random
	rand = append(rand, preGroupID...)
	group, err := p.groupAccessor.SelectVerifyGroupFromFork(base.Data2CommonHash(rand), baseBH.Height)
	if err != nil {
		return nil, err
	}
	groupCreateLogger.Debugf("Get Parent group:%s", common.ToHex(group.Id))
	return group, nil
}

// 选取候选人
// selectCandidates randomly select a sufficient number of miners from the miners' pool as new group candidates
func (p *groupCreateProcessor) selectCandidatesForForkGroup(theBH *types.BlockHeader) (enough bool, cands []groupsig.ID) {
	min := model.Param.CreateGroupMinCandidates()
	height := theBH.Height
	allCandidates := p.minerReader.GetCandidateMiners(height, theBH.StateTree)

	ids := make([]string, len(allCandidates))
	for idx, can := range allCandidates {
		ids[idx] = can.ID.ShortS()
	}
	groupCreateLogger.Debugf("GetAllCandidates:height %v, %v size %v", height, ids, len(allCandidates))
	if len(allCandidates) < min {
		return
	}
	groups := p.availableForkGroupsAt(theBH.Height)
	groupCreateLogger.Debugf("available group size %v", len(groups))

	candidates := make([]model.MinerInfo, 0)
	for _, cand := range allCandidates {
		joinedNum := 0
		for _, g := range groups {
			for _, mem := range g.Members {
				if bytes.Equal(mem, cand.ID.Serialize()) {
					joinedNum++
					break
				}
			}
		}

		if common.IsProposal009() {
			if joinedNum < int(cand.Stake/common.ValidatorStake) && joinedNum < common.MAXGROUP {
				candidates = append(candidates, cand)
			}
		} else {
			if joinedNum < int(cand.Stake/common.ValidatorStake) {
				candidates = append(candidates, cand)
			}
		}
	}
	num := len(candidates)

	selectNum := model.Param.CreateGroupMemberCount(num)
	if selectNum <= 0 {
		groupCreateLogger.Warnf("not enough candidates, got %v", len(candidates))
		return
	}

	rand := base.RandFromBytes(theBH.Random)
	seqs := rand.RandomPerm(num, selectNum)

	result := make([]groupsig.ID, len(seqs))
	for i, seq := range seqs {
		result[i] = candidates[seq].ID
	}

	str := ""
	for _, id := range result {
		str += id.ShortS() + ","
	}
	groupCreateLogger.Infof("Got Candidates: %v,size:%d", str, len(result))
	return true, result
}

func (p *groupCreateProcessor) availableForkGroupsAt(h uint64) []*types.Group {
	iter := p.groupChain.ForkIterator()
	gs := make([]*types.Group, 0)
	for g := iter.Current(); g != nil; g = iter.MovePre() {
		if g.Header.DismissHeight > h {
			gs = append(gs, g)
		} else {
			genesis := p.groupChain.GetGroupByHeight(0)
			gs = append(gs, genesis)
			break
		}
	}
	return gs
}
