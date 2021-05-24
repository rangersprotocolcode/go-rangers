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
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package consensus

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"com.tuntun.rocket/node/src/consensus/logical/group_create"
	"com.tuntun.rocket/node/src/consensus/model"
	"com.tuntun.rocket/node/src/consensus/vrf"
	"com.tuntun.rocket/node/src/middleware/types"
	"errors"
	"fmt"
	"math/big"
)

type ConsensusHelperImpl struct {
	ID groupsig.ID
}

func NewConsensusHelper(id groupsig.ID) types.ConsensusHelper {
	return &ConsensusHelperImpl{ID: id}
}

func (helper *ConsensusHelperImpl) ProposalBonus() *big.Int {
	return new(big.Int).SetUint64(model.Param.ProposalBonus)
}

func (helper *ConsensusHelperImpl) PackBonus() *big.Int {
	return new(big.Int).SetUint64(model.Param.PackBonus)
}

func (helper *ConsensusHelperImpl) GenerateGenesisInfo() []*types.GenesisInfo {
	return group_create.GetGenesisInfo()
}

func (helper *ConsensusHelperImpl) VRFProve2Value(prove *big.Int) *big.Int {
	return vrf.VRFProof2Hash(vrf.VRFProve(prove.Bytes())).Big()
}

func (helper *ConsensusHelperImpl) VerifyHash(b *types.Block) common.Hash {
	return Proc.GenVerifyHash(b, helper.ID)
}

func (helper *ConsensusHelperImpl) CheckProveRoot(bh *types.BlockHeader) (bool, error) {
	//preBlock := Proc.MainChain.QueryBlockByHash(bh.PreHash)
	//if preBlock == nil {
	//	return false, errors.New(fmt.Sprintf("preBlock is nil,hash %v", bh.PreHash.ShortS()))
	//}
	gid := groupsig.DeserializeID(bh.GroupId)
	group := Proc.GetGroup(gid)
	if !group.GroupID.IsValid() {
		return false, errors.New(fmt.Sprintf("group is invalid, gid %v", gid))
	}

	//todo 暂时去掉全量账本验证(当取样块高不存在时可能导致计算量很大)
	return true, nil
	//preBh := preBlock.Header
	//if _, root := Proc.GenProveHashs(bh.Height, preBH.Random, group.GetMembers()); root == bh.ProveRoot {
	//	return true, nil
	//} else {
	//	return false, errors.New(fmt.Sprintf("proveRoot expect %v, receive %v", bh.ProveRoot.String(), root.String()))
	//}

}

func (helper *ConsensusHelperImpl) VerifyNewBlock(bh *types.BlockHeader, preBH *types.BlockHeader) (bool, error) {
	return Proc.VerifyBlock(bh, preBH)
}

func (helper *ConsensusHelperImpl) VerifyBlockHeader(bh *types.BlockHeader) (bool, error) {
	return Proc.VerifyBlockHeader(bh)
}

func (helper *ConsensusHelperImpl) VerifyGroupSign(groupPubkey []byte, blockHash common.Hash, sign []byte) (bool, error) {
	return Proc.VerifyGroupSign(groupPubkey, blockHash, sign)
}

func (helper *ConsensusHelperImpl) CheckGroup(g *types.Group) (ok bool, err error) {
	return Proc.VerifyGroup(g)
}

func (helper *ConsensusHelperImpl) VerifyGroupForFork(g *types.Group, preGroup *types.Group, parentGroup *types.Group, baseBlock *types.Block) (ok bool, err error) {
	return group_create.GroupCreateProcessor.VerifyGroupForFork(g, preGroup, parentGroup, baseBlock)
}

func (helper *ConsensusHelperImpl) VerifyMemberInfo(bh *types.BlockHeader, preBH *types.BlockHeader) (bool, error) {
	return Proc.IsCastLegalForFork(bh, preBH)
}
