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
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/middleware/types"
	"fmt"
	"math/big"
)

func (r *round3) Close() {

}

func (r *round3) Update(msg model.ConsensusMessage) *Error {
	return nil
}

func (r *round3) CanAccept(msg model.ConsensusMessage) int {
	return -1
}

func (r *round3) NextRound() Round {
	return nil
}

func (r *round3) Start() *Error {
	bh := r.bh
	r.logger.Debugf("round3 start, hash: %s, height: %d", bh.Hash.String(), bh.Height)

	if r.blockchain.HasBlockByHash(bh.Hash) {
		return NewError(fmt.Errorf("blockheader already existed, height: %d, hash: %s", bh.Height, bh.Hash.String()), "finalizer", r.RoundNumber(), "", nil)
	}

	group, err := r.globalGroups.GetGroupByID(groupsig.DeserializeID(bh.GroupId))
	if nil != err {
		return NewError(fmt.Errorf("fail to get group, height: %d, hash: %s, group: %s", bh.Height, bh.Hash.String(), common.ToHex(bh.GroupId)), "finalizer", r.RoundNumber(), "", nil)
	}
	if err := r.checkSignature(group); err != nil {
		return err
	}

	block := r.blockchain.GenerateBlock(*bh)
	if block == nil {
		return NewError(fmt.Errorf("fail to generate block, height: %d, hash: %s", bh.Height, bh.Hash.String()), "finalizer", r.RoundNumber(), "", nil)
	}
	result := r.blockchain.AddBlockOnChain(block)
	if types.AddBlockSucc != result {
		return NewError(fmt.Errorf("fail to add block, height: %d, hash: %s, group: %s, result: %d", bh.Height, bh.Hash.String(), common.ToHex(bh.GroupId), result), "finalizer", r.RoundNumber(), "", nil)
	}
	r.logger.Infof("round3 add block, height: %d, hash: %s", bh.Height, bh.Hash.String())

	// send block if nesscessary
	r.broadcastNewBlock(group, *block)
	return nil
}

func (r *round3) broadcastNewBlock(group *model.GroupInfo, block types.Block) {
	bh := block.Header

	seed := big.NewInt(0).SetBytes(bh.Hash.Bytes()).Uint64()
	index := seed % uint64(group.GetMemberCount())
	id := group.GetMemberID(int(index)).GetBigInt()
	if id.Cmp(r.mi.GetBigInt()) == 0 {
		cbm := &model.ConsensusBlockMessage{
			Block: block,
		}
		r.netServer.BroadcastNewBlock(cbm)
		r.logger.Infof("round3 broadcasted block, height: %d, hash: %s", bh.Height, bh.Hash.String())
	} else {
		r.logger.Infof("round3 not broadcasted block, height: %d, hash: %s", bh.Height, bh.Hash.String())
	}
}

func (r *round3) checkSignature(group *model.GroupInfo) *Error {
	bh := r.bh
	gpk := group.GetGroupPubKey()
	if !groupsig.VerifySig(gpk, bh.Hash.Bytes(), *groupsig.DeserializeSign(bh.Signature)) {
		return NewError(fmt.Errorf("fail to verify group sign, height: %d, hash: %s, group: %s", bh.Height, bh.Hash.String(), common.ToHex(bh.GroupId)), "finalizer", r.RoundNumber(), "", nil)
	}
	if !groupsig.VerifySig(gpk, r.preBH.Random, *groupsig.DeserializeSign(bh.Random)) {
		return NewError(fmt.Errorf("fail to verify random sign, height: %d, hash: %s, group: %s", bh.Height, bh.Hash.String(), common.ToHex(bh.GroupId)), "finalizer", r.RoundNumber(), "", nil)
	}

	return nil
}
