// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package logical

import (
	"errors"
	"sync/atomic"
	"time"

	"com.tuntun.rocket/node/src/consensus/model"
	"com.tuntun.rocket/node/src/consensus/vrf"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/utility"
)

const (
	prove    int32 = 0
	proposed       = 1
	success        = 2
)

type vrfWorker struct {
	miner      *model.SelfMinerInfo
	baseBH     *types.BlockHeader
	castHeight uint64
	expire     time.Time
	status     int32
}

func newVRFWorker(miner *model.SelfMinerInfo, bh *types.BlockHeader, castHeight uint64, expire time.Time) *vrfWorker {
	return &vrfWorker{
		miner:      miner,
		baseBH:     bh,
		castHeight: castHeight,
		expire:     expire,
		status:     prove,
	}
}

func (vrfWorker *vrfWorker) genProve(castTime time.Time, totalStake uint64) (vrf.VRFProve, uint64, error) {
	delta := CalDeltaByTime(castTime, vrfWorker.baseBH.CurTime)
	vrfMsg := genVrfMsg(vrfWorker.baseBH.Random, delta)
	prove, err := vrf.VRFGenProve(vrfWorker.miner.VrfPK, vrfWorker.miner.VrfSK, vrfMsg)
	if err != nil {
		return nil, 0, err
	}

	if ok, qn := validateProve(prove, vrfWorker.miner.Stake, totalStake); ok {
		return prove, qn, nil
	}
	return nil, 0, errors.New("proof fail")
}

func (vrf *vrfWorker) markProposed() {
	atomic.CompareAndSwapInt32(&vrf.status, prove, proposed)
}

func (vrf *vrfWorker) markSuccess() {
	atomic.CompareAndSwapInt32(&vrf.status, proposed, success)
}

func (vrf *vrfWorker) getBaseBH() *types.BlockHeader {
	return vrf.baseBH
}

func (vrf *vrfWorker) isSuccess() bool {
	return vrf.getStatus() == success
}

func (vrf *vrfWorker) isProposed() bool {
	return vrf.getStatus() == proposed
}

func (vrf *vrfWorker) getStatus() int32 {
	return atomic.LoadInt32(&vrf.status)
}

func (vrf *vrfWorker) workingOn(bh *types.BlockHeader, castHeight uint64) bool {
	return bh.Hash == vrf.baseBH.Hash && castHeight == vrf.castHeight && !vrf.timeout()
}

func (vrf *vrfWorker) timeout() bool {
	return utility.GetTime().After(vrf.expire)
}
