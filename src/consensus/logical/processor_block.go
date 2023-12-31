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

package logical

import (
	"bytes"
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/consensus/logical/group_create"
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/utility"
	"fmt"
	"sync"
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

func (p *Processor) doAddOnChain(block *types.Block) (result int8) {
	bh := block.Header

	rlog := newRtLog("doAddOnChain")
	rlog.log("start, height=%v, hash=%v", bh.Height, bh.Hash.ShortS())
	result = int8(p.MainChain.AddBlockOnChain(block))
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
	begin := utility.GetTime()
	defer func() {
		if utility.GetTime().Sub(begin).Seconds() > 0.5 {
			slowLogger.Warnf("slowQueryBlockHeaderByHash: cost %v, hash=%v", utility.GetTime().Sub(begin), hash.ShortS())
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
	p.NetServer.JoinGroupNet(sgi.GroupID.GetHexString())

	bc := NewBlockContext(p, sgi)

	stdLogger.Debugf("prepareForCast current ID %v.\n", p.GetMinerID().ShortS())

	b := p.AddBlockContext(bc)
	stdLogger.Infof("(proc:%v) prepareForCast Add BlockContext result = %v, bc_size=%v.\n", p.getPrefix(), b, p.blockContexts.blockContextSize())
}

func (p *Processor) getNearestBlockByHeight(h uint64) *types.Block {
	for {
		b := p.MainChain.QueryBlock(h)
		if b != nil {
			b := p.MainChain.QueryBlockByHash(b.Header.Hash)
			if b != nil {
				return b
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

func (p *Processor) VerifyGroupSign(groupPubkey []byte, blockHash common.Hash, sign []byte) (ok bool, err error) {
	gpk := groupsig.ByteToPublicKey(groupPubkey)
	sig := groupsig.DeserializeSign(sign)
	b := groupsig.VerifySig(gpk, blockHash.Bytes(), *sig)
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

	if _, ok, err := group_create.GroupCreateProcessor.ValidateGroupInfo(gInfo); ok {
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
