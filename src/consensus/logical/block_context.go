package logical

import (
	"bytes"
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/model"
	"com.tuntun.rocket/node/src/middleware/types"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type castedBlock struct {
	hash    common.Hash
	preHash common.Hash
}

///////////////////////////////////////////////////////////////////////////////
//组铸块共识上下文结构（一个高度有一个上下文，一个组的不同铸块高度不重用）
type BlockContext struct {
	Version      uint
	GroupMembers int                 //组成员数量
	Proc         *Processor          //处理器
	MinerID      *model.GroupMinerID //矿工ID和所属组ID

	signedMaxQN uint64

	//变化
	vctxs map[common.Hash]*VerifyContext //height -> *VerifyContext

	// 缓存最近cast过的信息
	recentCasted [100]*castedBlock
	curr         int

	lock sync.RWMutex
}

func NewBlockContext(p *Processor, sgi *model.GroupInfo) *BlockContext {
	bc := &BlockContext{
		Proc:         p,
		MinerID:      model.NewGroupMinerID(sgi.GroupID, p.GetMinerID()),
		GroupMembers: sgi.GetMemberCount(),
		vctxs:        make(map[common.Hash]*VerifyContext),
		Version:      model.CONSENSUS_VERSION,
		curr:         0,
	}

	return bc
}

func (bc *BlockContext) threshold() int {
	return model.Param.GetGroupK(bc.GroupMembers)
}

func (bc *BlockContext) GetVerifyContextByHash(hash common.Hash) *VerifyContext {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	return bc.getVctxByHash(hash)
}

func (bc *BlockContext) getVctxByHash(hash common.Hash) *VerifyContext {
	if v, ok := bc.vctxs[hash]; ok {
		return v
	}
	return nil
}

func (bc *BlockContext) replaceVerifyCtx(hash common.Hash, expireTime time.Time, preBH *types.BlockHeader) *VerifyContext {
	vctx := newVerifyContext(bc, expireTime, preBH, hash)
	bc.vctxs[hash] = vctx
	return vctx
}

func (bc *BlockContext) getOrNewVctx(expireTime time.Time, bh, preBH *types.BlockHeader) *VerifyContext {
	var vctx *VerifyContext
	blog := newBizLog("getOrNewVctx")
	hash := bh.Hash

	//若该高度还没有verifyContext， 则创建一个
	if vctx = bc.getVctxByHash(hash); vctx == nil {
		vctx = newVerifyContext(bc, expireTime, preBH, hash)
		bc.vctxs[hash] = vctx
		blog.log("add vctx expire %v", expireTime)
	} else {
		// hash不一致的情况下，
		if vctx.prevBH.Hash != preBH.Hash {
			blog.log("vctx pre hash diff, hash=%v, existHash=%v, commingHash=%v", hash, vctx.prevBH.Hash.ShortS(), preBH.Hash.ShortS())
			preOld := bc.Proc.getBlockHeaderByHash(vctx.prevBH.Hash)
			//原来的preBH可能被分叉调整干掉了，则此vctx已无效， 重新用新的preBH
			if preOld == nil {
				vctx = bc.replaceVerifyCtx(hash, expireTime, preBH)
				return vctx
			}
			preNew := bc.Proc.getBlockHeaderByHash(preBH.Hash)
			//新的preBH不存在了，也可能被分叉干掉了，此处直接返回nil
			if preNew == nil {
				return nil
			}
			//新旧preBH都非空， 取高度高的preBH？
			if preOld.Height < preNew.Height {
				vctx = bc.replaceVerifyCtx(hash, expireTime, preNew)
			}
		} else {
			if bh.Height == 1 && expireTime.After(vctx.expireTime) {
				vctx.expireTime = expireTime
			}
			blog.log("get exist vctx hash %v, expire %v", hash, vctx.expireTime)
		}
	}
	return vctx
}

func (bc *BlockContext) SafeGetVerifyContexts() []*VerifyContext {
	bc.lock.RLock()
	defer bc.lock.RUnlock()
	vctx := make([]*VerifyContext, len(bc.vctxs))
	i := 0
	for _, vc := range bc.vctxs {
		vctx[i] = vc
		i++
	}
	return vctx
}

func (bc *BlockContext) GetOrNewVerifyContext(bh *types.BlockHeader, preBH *types.BlockHeader) *VerifyContext {
	deltaHeightByTime := DeltaHeightByTime(bh, preBH)

	expireTime := GetCastExpireTime(preBH.CurTime, deltaHeightByTime, bh.Height)

	bc.lock.Lock()
	defer bc.lock.Unlock()

	vctx := bc.getOrNewVctx(expireTime, bh, preBH)
	return vctx
}

func (bc *BlockContext) CleanVerifyContext(height uint64) {
	newCtxs := make(map[common.Hash]*VerifyContext, 0)
	for _, ctx := range bc.SafeGetVerifyContexts() {
		bRemove := ctx.shouldRemove(height)
		if !bRemove {
			newCtxs[ctx.blockHash] = ctx
		} else {
			ctx.Clear()
			bc.Proc.blockContexts.removeReservedVctx(ctx.blockHash)
			stdLogger.Debugf("CleanVerifyContext: ctx.castHash=%v, ctx.prevHash=%v, signedNum=%v\n", ctx.blockHash, ctx.prevBH.Hash.ShortS(), ctx.signedNum)
		}
	}

	bc.lock.Lock()
	defer bc.lock.Unlock()
	bc.vctxs = newCtxs
}

//
func (bc *BlockContext) IsHashCasted(hash, pre common.Hash) (cb *castedBlock, casted bool) {
	bc.lock.RLock()
	defer bc.lock.RUnlock()
	for i, h := range bc.recentCasted {
		if h != nil && bytes.Compare(h.hash.Bytes(), hash.Bytes()) == 0 {
			cb = bc.recentCasted[i]
			casted = h.preHash == pre
			return
		}
	}
	return
}

func (bc *BlockContext) AddCastedHeight(hash, pre common.Hash) {
	if cb, same := bc.IsHashCasted(hash, pre); same {
		return
	} else {
		bc.lock.Lock()
		defer bc.lock.Unlock()

		if cb != nil {
			cb.preHash = pre
		} else {
			bc.recentCasted[bc.curr] = &castedBlock{hash: hash, preHash: pre}
			bc.curr = (bc.curr + 1) % len(bc.recentCasted)
		}
	}
}

func (bc *BlockContext) CheckQN(bh *types.BlockHeader) error {
	//只签qn不小于已签出的最高块的块
	if bc.hasSignedBiggerQN(bh.TotalQN) {
		return fmt.Errorf("已签过更高qn块%v,本块qn%v", bc.getSignedMaxQN(), bh.TotalQN)
	}

	return nil
}

func (bc *BlockContext) getSignedMaxQN() uint64 {
	return atomic.LoadUint64(&bc.signedMaxQN)
}

func (bc *BlockContext) hasSignedBiggerQN(totalQN uint64) bool {
	return bc.getSignedMaxQN() > totalQN
}

func (bc *BlockContext) updateSignedMaxQN(totalQN uint64) bool {
	if bc.getSignedMaxQN() < totalQN {
		atomic.StoreUint64(&bc.signedMaxQN, totalQN)
		return true
	}
	return false
}
