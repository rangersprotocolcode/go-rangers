package consensus

import (
	"x/src/common"
	"x/src/consensus/vrf"
	"x/src/consensus/groupsig"
	"x/src/consensus/logical"
	"x/src/consensus/model"
	"errors"
	"fmt"
	"math/big"
	"x/src/middleware/types"
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
	return logical.GenerateGenesis()
}

func (helper *ConsensusHelperImpl) VRFProve2Value(prove *big.Int) *big.Int {
	return vrf.VRFProof2Hash(vrf.VRFProve(prove.Bytes())).Big()
}

func (helper *ConsensusHelperImpl) CalculateQN(bh *types.BlockHeader) uint64 {
	return Proc.CalcBlockHeaderQN(bh)
}

func (helper *ConsensusHelperImpl) VerifyHash(b *types.Block) common.Hash {
	return Proc.GenVerifyHash(b, helper.ID)
}

func (helper *ConsensusHelperImpl) CheckProveRoot(bh *types.BlockHeader) (bool, error) {
	preBlock := Proc.MainChain.QueryBlockByHash(bh.PreHash)
	if preBlock == nil {
		return false, errors.New(fmt.Sprintf("preBlock is nil,hash %v", bh.PreHash.ShortS()))
	}
	gid := groupsig.DeserializeId(bh.GroupId)
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

func (helper *ConsensusHelperImpl) CheckGroup(g *types.Group) (ok bool, err error) {
	return Proc.VerifyGroup(g)
}
