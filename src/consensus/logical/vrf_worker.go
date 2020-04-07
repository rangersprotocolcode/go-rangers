package logical

import (
	"sync/atomic"
	"time"
	"errors"

	"x/src/consensus/model"
	"x/src/middleware/types"
	"x/src/consensus/vrf"
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
	delta := int(castTime.Sub(vrfWorker.baseBH.CurTime).Seconds()) / model.MAX_GROUP_BLOCK_TIME
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
	//return time.Now().After(vrf.expire)
	return false
}
