package logical

import (
	"fmt"
	"x/src/common"
	"x/src/consensus/base"
	"x/src/consensus/groupsig"
	"x/src/consensus/model"
	"x/src/consensus/vrf"
	"x/src/middleware/types"
)

//后续如有全局定时器，从这个函数启动
func (p *Processor) Start() bool {
	// 检查是否要出块
	p.Ticker.RegisterRoutine(p.getCastCheckRoutineName(), p.checkSelfCastRoutine, common.CastingCheckInterval)

	// 出块后的广播
	p.Ticker.RegisterRoutine(p.getBroadcastRoutineName(), p.broadcastRoutine, 300)
	p.Ticker.StartTickerRoutine(p.getBroadcastRoutineName(), false)

	// 组解散
	p.Ticker.RegisterRoutine(p.getReleaseRoutineName(), p.releaseRoutine, 600)
	p.Ticker.StartTickerRoutine(p.getReleaseRoutineName(), false)

	p.Ticker.RegisterRoutine(p.getUpdateGlobalGroupsRoutineName(), p.updateGlobalGroups, 60*1000)
	p.Ticker.StartTickerRoutine(p.getUpdateGlobalGroupsRoutineName(), false)

	p.triggerCastCheck()
	p.prepareMiner()
	p.ready = true
	return true
}

//预留接口
func (p *Processor) Stop() {
	return
}

func (p *Processor) prepareMiner() {

	topHeight := p.MainChain.TopBlock().Height

	stdLogger.Infof("prepareMiner get groups from groupchain")
	iterator := p.GroupChain.Iterator()
	groups := make([]*model.GroupInfo, 0)
	for coreGroup := iterator.Current(); coreGroup != nil; coreGroup = iterator.MovePre() {
		stdLogger.Infof("get group from core, id=%+v", coreGroup.Header)
		if coreGroup.Id == nil || len(coreGroup.Id) == 0 {
			continue
		}
		needBreak := false
		sgi := model.ConvertToGroupInfo(coreGroup)
		//if sgi.Dismissed(topHeight) {
		//	needBreak = true
		//	genesis := p.GroupChain.GetGroupByHeight(0)
		//	if coreGroup == nil {
		//		panic("get genesis group nil")
		//	}
		//	sgi = NewSGIFromCoreGroup(genesis)
		//}
		groups = append(groups, sgi)
		stdLogger.Infof("load group=%v, beginHeight=%v, topHeight=%v\n", sgi.GroupID.ShortS(), sgi.GetGroupHeader().WorkHeight, topHeight)
		if sgi.MemExist(p.GetMinerID()) {
			jg := p.belongGroups.GetJoinedGroupInfo(sgi.GroupID)
			if jg == nil {
				stdLogger.Infof("prepareMiner get join group fail, gid=%v\n", sgi.GroupID.ShortS())
			} else {
				p.belongGroups.JoinGroup(jg, p.mi.ID)
			}
		}
		if needBreak {
			break
		}
	}
	for i := len(groups) - 1; i >= 0; i-- {
		p.acceptGroup(groups[i])
	}
	stdLogger.Infof("prepare finished")
}

func (p *Processor) Ready() bool {
	return p.ready
}

func (p *Processor) GetCastQualifiedGroups(height uint64) []*model.GroupInfo {
	return p.globalGroups.GetEffectiveGroups(height)
}

func (p *Processor) Finalize() {
	if p.belongGroups != nil {
		p.belongGroups.Close()
	}
}

func (p *Processor) GetVrfWorker() *vrfWorker {
	if v := p.vrf.Load(); v != nil {
		return v.(*vrfWorker)
	}
	return nil
}

func (p *Processor) setVrfWorker(vrf *vrfWorker) {
	p.vrf.Store(vrf)
}

func (p *Processor) GetSelfMinerDO() *model.SelfMinerInfo {
	md := p.minerReader.GetProposeMiner(p.GetMinerID())
	if md != nil {
		p.mi.MinerInfo = *md
	}
	return p.mi
}

func (p *Processor) canProposalAt(h uint64) bool {
	miner := p.minerReader.GetProposeMiner(p.GetMinerID())
	if miner == nil {
		//		stdLogger.Errorf("get nil proposeMiner:%s", p.GetMinerID().String())
		return false
	}
	return miner.CanCastAt(h)
}

func (p *Processor) GetJoinedWorkGroupNums() (work, avail int) {
	h := p.MainChain.TopBlock().Height
	groups := p.globalGroups.GetAvailableGroups(h)
	for _, g := range groups {
		if !g.MemExist(p.GetMinerID()) {
			continue
		}
		if g.IsEffective(h) {
			work++
		}
		avail++
	}
	return
}

func (p *Processor) CalcBlockHeaderQN(bh *types.BlockHeader) uint64 {
	pi := vrf.VRFProve(bh.ProveValue.Bytes())
	castor := groupsig.DeserializeID(bh.Castor)
	miner := p.minerReader.GetProposeMiner(castor)
	if miner == nil {
		stdLogger.Infof("CalcBHQN getMiner nil id=%v, bh=%v", castor.ShortS(), bh.Hash.ShortS())
		return 0
	}
	pre := p.MainChain.QueryBlockByHash(bh.PreHash)
	if pre == nil {
		return 0
	}
	totalStake := p.minerReader.GetTotalStake(pre.Header.Height, false)
	_, qn := validateProve(pi, miner.Stake, totalStake)
	return qn
}

//func marshalBlock(b types.Block) ([]byte, error) {
//	if b.Transactions != nil && len(b.Transactions) == 0 {
//		b.Transactions = nil
//	}
//	if b.Header.Transactions != nil && len(b.Header.Transactions) == 0 {
//		b.Header.Transactions = nil
//	}
//	return msgpack.Marshal(&b)
//}

func (p *Processor) GenVerifyHash(b *types.Block, id groupsig.ID) common.Hash {
	buf, err := types.MarshalBlock(b)
	if err != nil {
		panic(fmt.Sprintf("marshal block error, hash=%v, err=%v", b.Header.Hash.ShortS(), err))
	}
	//header := &b.Header
	//log.Printf("GenVerifyHash aaa bufHash=%v, buf %v", base.Data2CommonHash(buf).ShortS(), buf)
	//log.Printf("GenVerifyHash aaa headerHash=%v, genHash=%v", b.Header.Hash.ShortS(), b.Header.GenHash().ShortS())

	//headBuf, _ := msgpack.Marshal(header)
	//log.Printf("GenVerifyHash aaa headerBufHash=%v, headerBuf=%v", base.Data2CommonHash(headBuf).ShortS(), headBuf)

	//log.Printf("GenVerifyHash height:%v,id:%v,%v, bbbbbuf %v", b.Header.Height,id.ShortS(), b.Transactions == nil, buf)
	//log.Printf("GenVerifyHash height:%v,id:%v,bbbbbuf ids %v", b.Header.Height,id.ShortS(),id.Serialize())
	buf = append(buf, id.Serialize()...)
	//log.Printf("GenVerifyHash height:%v,id:%v,bbbbbuf after %v", b.Header.Height,id.ShortS(),buf)
	h := base.Data2CommonHash(buf)
	//log.Printf("GenVerifyHash height:%v,id:%v,bh:%v,vh:%v", b.Header.Height,id.ShortS(),b.Header.Hash.ShortS(), h.ShortS())
	return h
}

func (p *Processor) GetJoinGroupInfo(gid string) *model.JoinedGroupInfo {
	var id groupsig.ID
	id.SetHexString(gid)
	jg := p.belongGroups.GetJoinedGroupInfo(id)
	return jg
}

//func (p *Processor) GetAllMinerDOs() ([]*model.MinerDO) {
//	h := p.MainChain.Height()
//	dos := make([]*model.MinerDO, 0)
//	miners := p.minerReader.getAllMinerDOByType(common.MinerTypeProposer, h)
//	dos = append(dos, miners...)
//
//	miners = p.minerReader.getAllMinerDOByType(common.MinerTypeValidator, h)
//	dos = append(dos, miners...)
//	return dos
//}

func (p *Processor) GetCastQualifiedGroupsFromChain(height uint64) []*types.Group {
	return p.globalGroups.GetCastQualifiedGroupFromChains(height)
}
