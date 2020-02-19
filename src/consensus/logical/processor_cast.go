package logical

import (
	"bytes"
	"x/src/common"
	"x/src/consensus/base"
	"x/src/consensus/groupsig"
	"x/src/consensus/model"
	"x/src/consensus/net"

	"x/src/middleware/types"
	"sync"
	"time"
	"runtime/debug"

	"x/src/middleware"
)

type CastBlockContexts struct {
	blockCtxs    sync.Map //string -> *BlockContext
	reservedVctx sync.Map //uint64 -> *VerifyContext 存储已经有签出块的verifyContext，待广播
}

func NewCastBlockContexts() *CastBlockContexts {
	return &CastBlockContexts{
		blockCtxs: sync.Map{},
	}
}

func (bctx *CastBlockContexts) addBlockContext(bc *BlockContext) (add bool) {
	_, load := bctx.blockCtxs.LoadOrStore(bc.MinerID.Gid.GetHexString(), bc)
	return !load
}

func (bctx *CastBlockContexts) getBlockContext(gid groupsig.ID) *BlockContext {
	if v, ok := bctx.blockCtxs.Load(gid.GetHexString()); ok {
		return v.(*BlockContext)
	}
	return nil
}

func (bctx *CastBlockContexts) blockContextSize() int32 {
	size := int32(0)
	bctx.blockCtxs.Range(func(key, value interface{}) bool {
		size++
		return true
	})
	return size
}

func (bctx *CastBlockContexts) removeBlockContexts(gids []groupsig.ID) {
	for _, id := range gids {
		stdLogger.Infof("removeBlockContexts %v", id.ShortS())
		bc := bctx.getBlockContext(id)
		if bc != nil {
			//bc.removeTicker()
			for _, vctx := range bc.SafeGetVerifyContexts() {
				bctx.removeReservedVctx(vctx.castHeight)
			}
			bctx.blockCtxs.Delete(id.GetHexString())
		}
	}
}

func (bctx *CastBlockContexts) forEachBlockContext(f func(bc *BlockContext) bool) {
	bctx.blockCtxs.Range(func(key, value interface{}) bool {
		v := value.(*BlockContext)
		return f(v)
	})
}

func (bctx *CastBlockContexts) removeReservedVctx(height uint64) {
	bctx.reservedVctx.Delete(height)
}

func (bctx *CastBlockContexts) addReservedVctx(vctx *VerifyContext) bool {
	_, load := bctx.reservedVctx.LoadOrStore(vctx.castHeight, vctx)
	return !load
}

func (bctx *CastBlockContexts) forEachReservedVctx(f func(vctx *VerifyContext) bool) {
	bctx.reservedVctx.Range(func(key, value interface{}) bool {
		v := value.(*VerifyContext)
		return f(v)
	})
}

//增加一个铸块上下文（一个组有一个铸块上下文）
func (p *Processor) AddBlockContext(bc *BlockContext) bool {
	var add = p.blockContexts.addBlockContext(bc)
	newBizLog("AddBlockContext").log("gid=%v, result=%v\n.", bc.MinerID.Gid.ShortS(), add)
	return add
}

//取得一个铸块上下文
//gid:组ID hex 字符串
func (p *Processor) GetBlockContext(gid groupsig.ID) *BlockContext {
	return p.blockContexts.getBlockContext(gid)
}

//立即触发一次检查自己是否下个铸块组
func (p *Processor) triggerCastCheck() {
	//p.Ticker.StartTickerRoutine(p.getCastCheckRoutineName(), true)
	p.Ticker.StartAndTriggerRoutine(p.getCastCheckRoutineName())
}

func (p *Processor) CalcVerifyGroupFromCache(preBH *types.BlockHeader, height uint64) (*groupsig.ID) {
	var hash = CalcRandomHash(preBH, height)

	selectGroup, err := p.globalGroups.SelectVerifyGroupFromCache(hash, height)
	if err != nil {
		stdLogger.Errorf("SelectNextGroupFromCache height=%v, err: %v", height, err)
		return nil
	}
	return &selectGroup
}

func (p *Processor) CalcVerifyGroupFromChain(preBH *types.BlockHeader, height uint64) (*groupsig.ID) {
	var hash = CalcRandomHash(preBH, height)

	selectGroup, err := p.globalGroups.SelectNextGroupFromChain(hash, height)
	if err != nil {
		stdLogger.Errorf("SelectNextGroupFromChain height=%v, err:%v", height, err)
		return nil
	}
	return &selectGroup
}

func (p *Processor) spreadGroupBrief(bh *types.BlockHeader, height uint64) *net.GroupBrief {
	nextId := p.CalcVerifyGroupFromCache(bh, height)
	if nextId == nil {
		return nil
	}
	group := p.GetGroup(*nextId)
	g := &net.GroupBrief{
		Gid:    *nextId,
		MemIds: group.GetMembers(),
	}
	return g
}

func (p *Processor) reserveBlock(vctx *VerifyContext, slot *SlotContext) {
	bh := slot.BH
	blog := newBizLog("reserveBLock")
	blog.log("height=%v, totalQN=%v, hash=%v, slotStatus=%v", bh.Height, bh.TotalQN, bh.Hash.ShortS(), slot.GetSlotStatus())
	if slot.IsRecovered() {
		vctx.markCastSuccess() //onBlockAddSuccess方法中也mark了，该处调用是异步的
		p.blockContexts.addReservedVctx(vctx)
		if !p.tryBroadcastBlock(vctx) {
			blog.log("reserved, height=%v", vctx.castHeight)
		}

	}

	return
}

func (p *Processor) tryBroadcastBlock(vctx *VerifyContext) bool {
	if sc := vctx.checkBroadcast(); sc != nil {
		bh := sc.BH
		tlog := newHashTraceLog("tryBroadcastBlock", bh.Hash, p.GetMinerID())
		tlog.log("try broadcast, height=%v, totalQN=%v, 耗时%v秒", bh.Height, bh.TotalQN, time.Since(bh.CurTime).Seconds())

		//异步进行，使得请求快速返回，防止消息积压
		go p.successNewBlock(vctx, sc) //上链和组外广播

		p.blockContexts.removeReservedVctx(vctx.castHeight)
		return true
	}
	return false
}

//在某个区块高度的QN值成功出块，保存上链，向组外广播
//同一个高度，可能会因QN不同而多次调用该函数
//但一旦低的QN出过，就不该出高的QN。即该函数可能被多次调用，但是调用的QN值越来越小
func (p *Processor) successNewBlock(vctx *VerifyContext, slot *SlotContext) {
	defer func() {
		if r := recover(); r != nil {
			common.DefaultLogger.Errorf("error：%v\n", r)
			s := debug.Stack()
			common.DefaultLogger.Errorf(string(s))
		}
	}()

	bh := slot.BH

	blog := newBizLog("successNewBlock")

	if slot.IsFailed() {
		blog.log("slot is failed")
		return
	}
	if vctx.broadCasted() {
		blog.log("block broadCasted!")
		return
	}

	if p.blockOnChain(bh.Hash) { //已经上链
		blog.log("block alreayd onchain!")
		return
	}

	block := p.MainChain.GenerateBlock(*bh)

	if block == nil {
		blog.log("core.GenerateBlock is nil! won't broadcast block!")
		return
	}
	gb := p.spreadGroupBrief(bh, bh.Height+1)
	if gb == nil {
		blog.log("spreadGroupBrief nil, bh=%v, height=%v", bh.Hash.ShortS(), bh.Height)
		return
	}

	gpk := p.getGroupPubKey(groupsig.DeserializeID(bh.GroupId))
	if !slot.VerifyGroupSigns(gpk, vctx.prevBH.Random) { //组签名验证通过
		blog.log("group pub key local check failed, gpk=%v, hash in slot=%v, hash in bh=%v status=%v.",
			gpk.ShortS(), slot.BH.Hash.ShortS(), bh.Hash.ShortS(), slot.GetSlotStatus())
		return
	}

	r := p.doAddOnChain(block)

	if r != int8(types.AddBlockSucc) { //分叉调整或 上链失败都不走下面的逻辑
		if r != int8(types.Forking) {
			slot.setSlotStatus(SS_FAILED)
		}
		return
	}

	tlog := newHashTraceLog("successNewBlock", bh.Hash, p.GetMinerID())

	tlog.log("height=%v, status=%v", bh.Height, vctx.consensusStatus)
	cbm := &model.ConsensusBlockMessage{
		Block: *block,
	}

	p.NetServer.BroadcastNewBlock(cbm, gb)
	tlog.log("broadcasted height=%v, 耗时%v秒", bh.Height, time.Since(bh.CurTime).Seconds())

	//发送日志
	//le := &monitor.LogEntry{
	//	LogType:  monitor.LogTypeBlockBroadcast,
	//	Height:   bh.Height,
	//	Hash:     bh.Hash.Hex(),
	//	PreHash:  bh.PreHash.Hex(),
	//	Proposer: slot.castor.GetHexString(),
	//	Verifier: gb.Gid.GetHexString(),
	//}
	//monitor.Instance.AddLog(le)

	vctx.broadcastSlot = slot
	vctx.markBroadcast()
	slot.setSlotStatus(SS_SUCCESS)

	blog.log("After BroadcastNewBlock hash=%v:%v", bh.Hash.ShortS(), time.Now().Format(TIMESTAMP_LAYOUT))
	return
}

//对该id进行区块抽样
func (p *Processor) sampleBlockHeight(heightLimit uint64, rand []byte, id groupsig.ID) uint64 {
	//随机抽取10块前的块，确保不抽取到分叉上的块
	//
	if heightLimit > 2*model.Param.Epoch {
		heightLimit -= 2 * model.Param.Epoch
	}
	return base.RandFromBytes(rand).DerivedRand(id.Serialize()).ModuloUint64(heightLimit)
}

func (p *Processor) GenProveHashs(heightLimit uint64, rand []byte, ids []groupsig.ID) (proves []common.Hash, root common.Hash) {
	hashs := make([]common.Hash, len(ids))

	blog := newBizLog("GenProveHashs")
	for idx, id := range ids {
		h := p.sampleBlockHeight(heightLimit, rand, id)
		b := p.getNearestBlockByHeight(h)
		hashs[idx] = p.GenVerifyHash(b, id)
		blog.log("sampleHeight for %v is %v, real height is %v, proveHash is %v", id.String(), h, b.Header.Height, hashs[idx].String())
	}
	proves = hashs

	buf := bytes.Buffer{}
	for _, hash := range hashs {
		buf.Write(hash.Bytes())
	}
	root = base.Data2CommonHash(buf.Bytes())
	buf.Reset()
	return
}

func (p *Processor) blockProposal() {
	start := time.Now()
	blog := newBizLog("blockProposal")
	top := p.MainChain.TopBlock()
	worker := p.GetVrfWorker()
	if worker.getBaseBH().Hash != top.Hash {
		blog.log("vrf baseBH differ from top!")
		return
	}

	if worker.isProposed() || worker.isSuccess() {
		blog.log("vrf worker proposed/success, status %v", worker.getStatus())
		return
	}
	height := worker.castHeight

	totalStake := p.minerReader.getTotalStake(worker.baseBH.Height, false)
	blog.log("totalStake height=%v, stake=%v", height, totalStake)
	pi, qn, err := worker.genProve(totalStake)
	if err != nil {
		blog.log("vrf prove not ok! %v", err)
		return
	}

	if worker.timeout() {
		blog.log("vrf worker timeout")
		return
	}
	middleware.PerfLogger.Debugf("after genProve, last: %v, height: %v", time.Since(start), height)

	gb := p.spreadGroupBrief(top, height)
	if gb == nil {
		blog.log("spreadGroupBrief nil, bh=%v, height=%v", top.Hash.ShortS(), height)
		return
	}
	gid := gb.Gid
	middleware.PerfLogger.Debugf("after spreadGroupBrief, last: %v, height: %v", time.Since(start), height)

	//随机抽取n个块，生成proveHash
	//proveHash, root := p.GenProveHashs(height, worker.getBaseBH().Random, gb.MemIds)

	middleware.PerfLogger.Infof("start cast block, last: %v, height: %v", time.Since(start), height)
	block := p.MainChain.CastBlock(start, uint64(height), pi.Big(), common.Hash{}, qn, p.GetMinerID().Serialize(), gid.Serialize())
	if block == nil {
		blog.log("MainChain::CastingBlock failed, height=%v", height)
		return
	}
	bh := block.Header
	middleware.PerfLogger.Infof("fin cast block, last: %v, hash: %v, height: %v", time.Since(start), bh.Hash.String(), bh.Height)

	tlog := newHashTraceLog("CASTBLOCK", bh.Hash, p.GetMinerID())
	blog.log("begin proposal, hash=%v, height=%v, qn=%v,, verifyGroup=%v, pi=%v...", bh.Hash.ShortS(), height, qn, gid.ShortS(), pi.ShortS())
	tlog.logStart("height=%v,qn=%v, preHash=%v, verifyGroup=%v", bh.Height, qn, bh.PreHash.ShortS(), gid.ShortS())

	if bh.Height > 0 && bh.Height == height && bh.PreHash == worker.baseBH.Hash {
		skey := p.mi.SK //此处需要用普通私钥，非组相关私钥
		//发送该出块消息
		var ccm model.ConsensusCastMessage
		ccm.BH = *bh
		ccm.ProveHash = []common.Hash{}
		//ccm.GroupID = gid
		if !ccm.GenSign(model.NewSecKeyInfo(p.GetMinerID(), skey), &ccm) {
			blog.log("sign fail, id=%v, sk=%v", p.GetMinerID().ShortS(), skey.ShortS())
			return
		}
		//blog.log("hash=%v, proveRoot=%v, pi=%v, piHash=%v", bh.Hash.ShortS(), root.ShortS(), pi.ShortS(), common.Bytes2Hex(vrf.VRFProof2Hash(pi)))
		//ccm.GenRandomSign(skey, worker.baseBH.Random)//castor不能对随机数签名
		tlog.log("铸块成功, SendVerifiedCast, 时间间隔 %v, castor=%v, hash=%v, genHash=%v", bh.CurTime.Sub(bh.PreTime).Seconds(), ccm.SI.GetID().ShortS(), bh.Hash.ShortS(), ccm.SI.DataHash.ShortS())
		p.NetServer.SendCastVerify(&ccm, gb, block.Transactions)

		//发送日志
		//le := &monitor.LogEntry{
		//	LogType:  monitor.LogTypeProposal,
		//	Height:   bh.Height,
		//	Hash:     bh.Hash.Hex(),
		//	PreHash:  bh.PreHash.Hex(),
		//	Proposer: p.GetMinerID().GetHexString(),
		//	Verifier: gb.Gid.GetHexString(),
		//	Ext:      fmt.Sprintf("qn:%v,totalQN:%v", qn, bh.TotalQN),
		//}
		//monitor.Instance.AddLog(le)

		worker.markProposed()

		middleware.PerfLogger.Infof("fin block, last: %v, hash: %v, height: %v", time.Since(start), bh.Hash.String(), bh.Height)
		//statistics.AddBlockLog(common.BootId, statistics.SendCast, ccm.BH.Height, ccm.BH.ProveValue.Uint64(), -1, -1,
		//	time.Now().UnixNano(), p.GetMinerID().ShortS(), gid.ShortS(), common.InstanceIndex, ccm.BH.CurTime.UnixNano())
	} else {
		blog.log("bh/prehash Error or sign Error, bh=%v, real height=%v. bc.prehash=%v, bh.prehash=%v", height, bh.Height, worker.baseBH.Hash, bh.PreHash)
	}

}
