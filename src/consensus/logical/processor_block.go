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
	"fmt"
)

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
