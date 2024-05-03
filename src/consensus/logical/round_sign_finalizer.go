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
)

func (r *round2) Close() {

}

func (r *round2) Update(msg model.ConsensusMessage) *Error {
	return nil
}

func (r *round2) CanAccept(msg model.ConsensusMessage) int {
	return -1
}

func (r *round2) NextRound() Round {
	return nil
}

func (r *round2) Start() *Error {
	if r.finished {
		return NewError(fmt.Errorf("fail to start, already ended"), "finalizer", 2, "", nil)
	}

	r.finished = true
	bh := r.bh
	r.logger.Debugf("round2 start, hash: %s, height: %d", bh.Hash.String(), bh.Height)

	if err := r.checkBlockExisted(); err != nil {
		return err
	}

	if err := r.checkSignature(r.group); err != nil {
		return err
	}
	block := r.blockchain.GenerateBlock(*bh)
	if block == nil {
		return NewError(fmt.Errorf("fail to generate block, height: %d, hash: %s", bh.Height, bh.Hash.String()), "finalizer", r.RoundNumber(), "", nil)
	}

	go func() {
		result := r.blockchain.AddBlockOnChain(block)
		if types.AddBlockSucc != result {
			r.logger.Warnf("round2 not add and broadcast block, height: %d, hash: %s, result: %d, isSend: %v", bh.Height, bh.Hash.String(), result, r.isSend)
			return
		}

		r.logger.Infof("round2 add block, height: %d, hash: %s", bh.Height, bh.Hash.String())
		if r.isSend {
			r.broadcastNewBlock(*block)
		} else {
			r.logger.Infof("round2 not broadcast block, height: %d, hash: %s", bh.Height, bh.Hash.String())
		}
	}()

	r.done <- 1
	return nil
}

func (r *round2) checkSignature(group *model.GroupInfo) *Error {
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
