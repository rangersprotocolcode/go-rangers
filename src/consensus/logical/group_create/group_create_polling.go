package logical

import (
	"bytes"
	"math"
	"fmt"
	"x/src/consensus/model"
	"x/src/middleware/types"
	"x/src/consensus/base"
	"x/src/consensus/groupsig"
)

//检查建组
// CreateNextGroupRoutine start the group-create routine
func (p *groupCreateProcessor) StartCreateGroupPolling() {
	//todo 这里要确定是否是由创始组创建其他组
	//if !gm.processor.genesisMember {
	//	return
	//}
	top := p.blockChain.TopBlock()
	topHeight := top.Height

	gap := model.Param.GroupCreateGap
	if topHeight > gap {
		//todo 这里为啥要来一次？
		p.tryStartParentConsensus(topHeight)

		pre := p.blockChain.QueryBlockByHash(top.PreHash)
		if pre != nil {
			for h := top.Height; h > pre.Header.Height && h > gap; h-- {
				baseHeight := h - gap
				if validateHeight(baseHeight) {
					p.tryCreateGroup(baseHeight)
					break
				}
			}
		}
	}

}

//检查建组条件
// checkCreateGroupRoutine check if the height meets the conditions for creating a group
// if so then start the group-create process
func (p *groupCreateProcessor) tryCreateGroup(baseHeight uint64) {
	create := false
	var err error

	defer func() {
		ret := ""
		if err != nil {
			ret = err.Error()
		}
		groupCreateLogger.Debugf("baseBH height=%v, create=%v, err=%v", baseHeight, create, ret)
	}()

	// The specified height has appeared on the group chain
	if p.hasCreatedGroup(baseHeight) {
		err = fmt.Errorf("topHeight already created")
		return
	}

	// generate the basic context
	baseCtx, err2 := p.genCreateGroupBaseInfo(baseHeight)
	if err2 != nil {
		err = err2
		return
	}

	//todo 既然改成创始组创建了 之类还有必要检查吗？
	// if current node doesn't belong to the selected parent group, it won't start the routine
	if !p.belongGroup(baseCtx.parentGroupInfo.GroupID) {
		err = fmt.Errorf("next select group id %v, not belong to the group", baseCtx.parentGroupInfo.GroupID.GetHexString())
		return
	}

	kings, isKing := p.selectKing(baseCtx.baseBlockHeader, baseCtx.parentGroupInfo)

	p.setCreatingGroupContext(baseCtx, kings, isKing)
	groupCreateLogger.Infof("createGroupContext info=%v", p.context.String())

	p.pingNodes()
	create = true
}

//随机选一半的组成员作为KING
// selectKing just choose half of the people. Each person's weight is decremented in order
func (p *groupCreateProcessor) selectKing(theBH *types.BlockHeader, group *model.GroupInfo) (kings []groupsig.ID, isKing bool) {
	num := int(math.Ceil(float64(group.GetMemberCount() / 2)))
	if num < 1 {
		num = 1
	}

	rand := base.RandFromBytes(theBH.Random)

	isKing = false

	selectIndexs := rand.RandomPerm(group.GetMemberCount(), num)
	kings = make([]groupsig.ID, len(selectIndexs))
	for i, idx := range selectIndexs {
		kings[i] = group.GetMemberID(idx)
		if p.minerInfo.GetMinerID().IsEqual(kings[i]) {
			isKing = true
		}
	}
	groupCreateLogger.Infof("SelectKings:king index=%v, ids=%v, isKing %v",selectIndexs, kings, isKing)
	return
}

// selectParentGroup determine the parent group randomly and the result is deterministic because of the base BlockHeader
//获取父亲组
//todo 都是由创世组创建的？
func (p *groupCreateProcessor) selectParentGroup(baseBH *types.BlockHeader, preGroupID []byte) (*model.GroupInfo, error) {
	//todo 这里如何选择？
	return p.groupAccessor.GetGenesisGroup(), nil
}

//生成 CreateGroupContext
func (p *groupCreateProcessor) genCreateGroupBaseInfo(baseHeight uint64) (*createGroupBaseInfo, error) {
	lastGroup := p.groupChain.LastGroup()
	baseBH := p.blockChain.QueryBlock(baseHeight).Header
	if !validateHeight(baseHeight) {
		return nil, fmt.Errorf("cannot create group at the height")
	}
	if baseBH == nil {
		return nil, fmt.Errorf("base block is nil, height=%v", baseHeight)
	}
	sgi, err := p.selectParentGroup(baseBH, lastGroup.Id)
	if sgi == nil || err != nil {
		return nil, fmt.Errorf("select parent group err %v", err)
	}
	enough, candidates := p.selectCandidates(baseBH)
	if !enough {
		return nil, fmt.Errorf("not enough candidates")
	}
	return newCreateGroupBaseInfo(sgi, baseBH, lastGroup, candidates), nil
}


//选取候选人
// selectCandidates randomly select a sufficient number of miners from the miners' pool as new group candidates
func (p *groupCreateProcessor) selectCandidates(theBH *types.BlockHeader) (enough bool, cands []groupsig.ID) {
	min := model.Param.CreateGroupMinCandidates()
	height := theBH.Height
	allCandidates := p.minerReader.GetCandidateMiners(height)

	ids := make([]string, len(allCandidates))
	for idx, can := range allCandidates {
		ids[idx] = can.ID.ShortS()
	}
	groupCreateLogger.Debugf("GetAllCandidates:height %v, %v size %v", height, ids, len(allCandidates))
	if len(allCandidates) < min {
		return
	}
	groups := p.availableGroupsAt(theBH.Height)
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
		if joinedNum < model.Param.MinerMaxJoinGroup {
			candidates = append(candidates, cand)
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
	groupCreateLogger.Infof("Got Candidates: %v", str)
	return true, result
}

func (p *groupCreateProcessor) availableGroupsAt(h uint64) []*types.Group {
	iter := p.groupChain.Iterator()
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

//heightCreated
func (p *groupCreateProcessor) hasCreatedGroup(h uint64) bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	for _, height := range p.createdHeights {
		if h == height {
			return true
		}
	}
	return false
}

func (p *groupCreateProcessor) setCreatingGroupContext(baseCtx *createGroupBaseInfo, kings []groupsig.ID, isKing bool) {
	ctx := newCreateGroupContext(baseCtx, kings, isKing, p.blockChain.Height())
	p.context = ctx
}

//checkCreate
func validateHeight(h uint64) bool {
	return h > 0 && h%model.Param.CreateGroupInterval == 0
}

//// GroupCreateChecker is responsible for legality verification
//type GroupCreateChecker struct {
//	processor      *Processor
//	access         *MinerPoolReader
//	createdHeights [50]uint64 // Identifies whether the group height has already been created
//	curr           int
//	lock           sync.RWMutex // CreateHeightGroups mutex to prevent repeated writes
//}
//
//func newGroupCreateChecker(proc *Processor) *GroupCreateChecker {
//	return &GroupCreateChecker{
//		processor: proc,
//		access:    proc.minerReader,
//		curr:      0,
//	}
//}
