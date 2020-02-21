package logical

import (
	"x/src/common"
	"x/src/consensus/groupsig"
	"x/src/consensus/model"
	"fmt"
	"x/src/middleware/types"
	"sync"
	"bytes"
	"time"
)

type FutureMessageHolder struct {
	messages sync.Map
}

func NewFutureMessageHolder() *FutureMessageHolder {
	return &FutureMessageHolder{
		messages: sync.Map{},
	}
}
func (holder *FutureMessageHolder) addMessage(hash common.Hash, msg interface{}) {
	if vs, ok := holder.messages.Load(hash); ok {
		vsSlice := vs.([]interface{})
		vsSlice = append(vsSlice, msg)
		holder.messages.Store(hash, vsSlice)
	} else {
		slice := make([]interface{}, 0)
		slice = append(slice, msg)
		holder.messages.Store(hash, slice)
	}
}

func (holder *FutureMessageHolder) getMessages(hash common.Hash) []interface{} {
	if vs, ok := holder.messages.Load(hash); ok {
		return vs.([]interface{})
	}
	return nil
}

func (holder *FutureMessageHolder) remove(hash common.Hash) {
	holder.messages.Delete(hash)
}

func (holder *FutureMessageHolder) forEach(f func(key common.Hash, arr []interface{}) bool) {
	holder.messages.Range(func(key, value interface{}) bool {
		arr := value.([]interface{})
		return f(key.(common.Hash), arr)
	})
}

func (holder *FutureMessageHolder) size() int {
	cnt := 0
	holder.forEach(func(key common.Hash, value []interface{}) bool {
		cnt += len(value)
		return true
	})
	return cnt
}

//func (p *Processor) addFutureBlockMsg(msg *model.ConsensusBlockMessage) {
//	b := msg.Block
//	log.Printf("future block receive cached! h=%v, hash=%v\n", b.Header.Height, b.Header.Hash.ShortS())
//
//	p.futureBlockMsgs.addMessage(b.Header.PreHash, msg)
//}
//
//func (p *Processor) getFutureBlockMsgs(hash common.Hash) []*model.ConsensusBlockMessage {
//	if vs := p.futureBlockMsgs.getMessages(hash); vs != nil {
//		ret := make([]*model.ConsensusBlockMessage, len(vs))
//		for idx, m := range vs {
//			ret[idx] = m.(*model.ConsensusBlockMessage)
//		}
//		return ret
//	}
//	return nil
//}
//
//func (p *Processor) removeFutureBlockMsgs(hash common.Hash) {
//	p.futureBlockMsgs.remove(hash)
//}

func (p *Processor) doAddOnChain(block *types.Block) (result int8) {
	//begin := time.Now()
	//defer func() {
	//	log.Printf("doAddOnChain begin at %v, cost %v\n", begin.String(), time.Since(begin).String())
	//}()
	bh := block.Header

	rlog := newRtLog("doAddOnChain")
	//blog.log("start, height=%v, hash=%v", bh.Height, bh.Hash.ShortS())
	result = int8(p.MainChain.AddBlockOnChain("", block, types.LocalGenerateNewBlock))

	//log.Printf("AddBlockOnChain header %v \n", p.blockPreview(bh))
	//log.Printf("QueryTopBlock header %v \n", p.blockPreview(p.MainChain.QueryTopBlock()))
	rlog.log("height=%v, hash=%v, result=%v.", bh.Height, bh.Hash.ShortS(), result)
	castor := groupsig.DeserializeID(bh.Castor)
	tlog := newHashTraceLog("doAddOnChain", bh.Hash, castor)
	tlog.log("result=%v,castor=%v", result, castor.ShortS())

	if result == -1 {
		p.removeFutureVerifyMsgs(block.Header.Hash)

	}

	return result

}

func (p *Processor) blockOnChain(h common.Hash) bool {
	return p.MainChain.HasBlockByHash(h)
}

func (p *Processor) getBlockHeaderByHash(hash common.Hash) *types.BlockHeader {
	begin := time.Now()
	defer func() {
		if time.Since(begin).Seconds() > 0.5 {
			slowLogger.Warnf("slowQueryBlockHeaderByHash: cost %v, hash=%v", time.Since(begin).String(), hash.ShortS())
		}
	}()
	b := p.MainChain.QueryBlockByHash(hash)
	if b != nil {
		return b.Header
	}
	return nil
}

func (p *Processor) addFutureVerifyMsg(msg *model.ConsensusCastMessage) {
	b := msg.BH
	stdLogger.Debugf("future verifyMsg receive cached! h=%v, hash=%v, preHash=%v\n", b.Height, b.Hash.ShortS(), b.PreHash.ShortS())

	p.futureVerifyMsgs.addMessage(b.PreHash, msg)
}

func (p *Processor) getFutureVerifyMsgs(hash common.Hash) []*model.ConsensusCastMessage {
	if vs := p.futureVerifyMsgs.getMessages(hash); vs != nil {
		ret := make([]*model.ConsensusCastMessage, len(vs))
		for idx, m := range vs {
			ret[idx] = m.(*model.ConsensusCastMessage)
		}
		return ret
	}
	return nil
}

func (p *Processor) removeFutureVerifyMsgs(hash common.Hash) {
	p.futureVerifyMsgs.remove(hash)
}

func (p *Processor) blockPreview(bh *types.BlockHeader) string {
	return fmt.Sprintf("hash=%v, height=%v, curTime=%v, preHash=%v, preTime=%v", bh.Hash.ShortS(), bh.Height, bh.CurTime, bh.PreHash.ShortS(), bh.PreTime)
}

func (p *Processor) prepareForCast(sgi *model.GroupInfo) {
	//组建组网络
	p.NetServer.BuildGroupNet(sgi.GroupID.GetHexString(), sgi.GetGroupMembers())

	bc := NewBlockContext(p, sgi)

	bc.pos = sgi.GetMemberPosition(p.GetMinerID())
	stdLogger.Debugf("prepareForCast current ID %v in group pos=%v.\n", p.GetMinerID().ShortS(), bc.pos)
	//to do:只有自己属于这个组的节点才需要调用AddBlockConext
	b := p.AddBlockContext(bc)
	stdLogger.Infof("(proc:%v) prepareForCast Add BlockContext result = %v, bc_size=%v.\n", p.getPrefix(), b, p.blockContexts.blockContextSize())

	//bc.registerTicker()
	//p.triggerCastCheck()
}

func (p *Processor) getNearestBlockByHeight(h uint64) *types.Block {
	for {
		b := p.MainChain.QueryBlock(h)
		if b != nil {
			b := p.MainChain.QueryBlockByHash(b.Header.Hash)
			if b != nil {
				return b
			} else {
				//bh2 := p.MainChain.QueryBlockByHeight(h)
				//stdLogger.Debugf("get bh not nil, but block is nil! hash1=%v, hash2=%v, height=%v", bh.Hash.ShortS(), bh2.Hash.ShortS(), bh.Height)
				//if bh2.Hash == bh.Hash {
				//	panic("chain queryBlockByHash nil!")
				//} else {
				//	continue
				//}
			}
		}
		if h == 0 {
			panic("cannot find block of height 0")
		}
		h--
	}
}

func (p *Processor) getNearestVerifyHashByHeight(h uint64) (realHeight uint64, vhash common.Hash) {
	slog := newSlowLog("getNearestVerifyHashByHeight", 0.3)
	defer func() {
		slog.log("height %v", h)
	}()
	for {
		hash, err := p.MainChain.GetVerifyHash(h)

		if err == nil {
			return h, hash
		}
		if h == 0 {
			panic("cannot find verifyHash of height 0")
		}
		//todo 暂不检查取样块高不存在的,可能计算量很大
		break
		h--
	}
	return
}

func (p *Processor) VerifyBlock(bh *types.BlockHeader, preBH *types.BlockHeader) (ok bool, err error) {
	tlog := newMsgTraceLog("VerifyBlock", bh.Hash.ShortS(), "")
	defer func() {
		tlog.log("preHash=%v, height=%v, result=%v %v", bh.PreHash.ShortS(), bh.Height, ok, err)
		newBizLog("VerifyBlock").log("hash=%v, preHash=%v, height=%v, result=%v %v", bh.Hash.ShortS(), bh.PreHash.ShortS(), bh.Height, ok, err)
	}()
	if bh.Hash != bh.GenHash() {
		err = fmt.Errorf("block hash error")
		return
	}
	if preBH.Hash != bh.PreHash {
		err = fmt.Errorf("preHash error")
		return
	}

	if ok2, group, err2 := p.isCastLegal(bh, preBH); !ok2 {
		err = err2
		return
	} else {
		gpk := group.GroupPK
		sig := groupsig.DeserializeSign(bh.Signature)
		b := groupsig.VerifySig(gpk, bh.Hash.Bytes(), *sig)
		if !b {
			err = fmt.Errorf("signature verify fail")
			return
		}
		rsig := groupsig.DeserializeSign(bh.Random)
		b = groupsig.VerifySig(gpk, preBH.Random, *rsig)
		if !b {
			err = fmt.Errorf("random verify fail")
			return
		}
	}
	ok = true
	return
}

func (p *Processor) VerifyBlockHeader(bh *types.BlockHeader) (ok bool, err error) {
	if bh.Hash != bh.GenHash() {
		err = fmt.Errorf("block hash error")
		return
	}

	gid := groupsig.DeserializeID(bh.GroupId)
	gpk := p.getGroupPubKey(gid)
	sig := groupsig.DeserializeSign(bh.Signature)
	b := groupsig.VerifySig(gpk, bh.Hash.Bytes(), *sig)
	if !b {
		err = fmt.Errorf("signature verify fail")
		return
	}
	ok = true
	return
}

func (p *Processor) VerifyGroup(g *types.Group) (ok bool, err error) {
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
	//检验头和签名
	if _, ok, err := p.groupManager.checkGroupInfo(gInfo); ok {
		gpk := groupsig.ByteToPublicKey(g.PubKey)
		gid := groupsig.NewIDFromPubkey(gpk).Serialize()
		if !bytes.Equal(gid, g.Id) {
			return false, fmt.Errorf("gid error, expect %v, receive %v", gid, g.Id)
		}
	} else {
		return false, err
	}
	ok = true
	return
}
