package logical

import (
	"sync/atomic"
	"time"
	"errors"

	"x/src/consensus/model"
	"x/src/middleware/types"
	"x/src/consensus/vrf"
	"fmt"
	"x/src/consensus/base"
	"math/big"
	"math"
)

const (
	prove    int32 = 0
	proposed       = 1
	success        = 2
)

type vrfWorker struct {
	miner      *model.SelfMinerDO
	baseBH     *types.BlockHeader
	castHeight uint64
	expire     time.Time
	status     int32
}

func newVRFWorker(miner *model.SelfMinerDO, bh *types.BlockHeader, castHeight uint64, expire time.Time) *vrfWorker {
	return &vrfWorker{
		miner:      miner,
		castHeight: castHeight,
		expire:     expire,
		status:     prove,
	}
}

func (vrfWorker *vrfWorker) genProve(totalStake uint64) (vrf.VRFProve, uint64, error) {
	vrfMsg := genVrfMsg(vrfWorker.baseBH.Random, vrfWorker.castHeight-vrfWorker.baseBH.Height)
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
	return bh.Hash == vrf.baseBH.Hash && castHeight == vrf.castHeight && !time.Now().After(vrf.expire)
}

func (vrf *vrfWorker) timeout() bool {
	return time.Now().After(vrf.expire)
}

func vrfVerifyBlock(bh *types.BlockHeader, preBH *types.BlockHeader, miner *model.MinerDO, totalStake uint64) (bool, error) {
	pi := vrf.VRFProve(bh.ProveValue.Bytes())
	ok, err := vrf.VRFVerify(miner.VrfPK, pi, vrfM(preBH.Random, bh.Height-preBH.Height))
	if !ok {
		return ok, err
	}
	if ok, qn := vrfSatisfy(pi, miner.Stake, totalStake); ok {
		if bh.TotalQN != qn + preBH.TotalQN {
			return false, errors.New(fmt.Sprintf("qn error.bh hash=%v, height=%v, qn=%v,totalQN=%v, preBH totalQN=%v", bh.Hash.ShortS(), bh.Height, qn, bh.TotalQN, preBH.TotalQN))
		}
		return true, nil
	}
	return false, errors.New("proof not satisfy")
}

func vrfM(random []byte, h uint64) []byte {
	if h <= 0 {
		panic(fmt.Sprintf("vrf height error! deltaHeight=%v", h))
	}
	data := random
	for h > 1 {
		h--
		hash := base.Data2CommonHash(data)
		data = hash.Bytes()
	}
	return data
}

func vrfSatisfy(pi vrf.VRFProve, stake uint64, totalStake uint64) (ok bool, qn uint64) {
	if totalStake == 0 {
		stdLogger.Errorf("total stake is 0!")
		return false, 0
	}
	value := vrf.VRFProof2Hash(pi)

	br := new(big.Rat).SetInt(new(big.Int).SetBytes(value))
	pr := br.Quo(br, max256)

	//brTStake := new(big.Rat).SetFloat64(float64(totalStake))
	vs := vrfThreshold(stake, totalStake)

	s1, _ := pr.Float64()
	s2, _ := vs.Float64()
	blog := newBizLog("vrfSatisfy")

	ok = pr.Cmp(vs) < 0
	//计算qn
	if vs.Cmp(rat1) > 0 {
		vs.Set(rat1)
	}

	step := vs.Quo(vs, new(big.Rat).SetInt64(int64(model.Param.MaxQN)))

	st, _ := step.Float64()

	r, _ := pr.Quo(pr, step).Float64()
	qn = uint64(math.Floor(r) + 1)

	blog.log("minerstake %v, totalstake %v, proveValue %v, stake %v, step %v, qn %v", stake, totalStake, s1, s2, st, qn)

	return
	//return true
}

func vrfThreshold(stake, totalStake uint64) *big.Rat {
	brTStake := new(big.Rat).SetFloat64(float64(totalStake))
	return new(big.Rat).Quo(new(big.Rat).SetInt64(int64(stake*uint64(model.Param.PotentialProposal))), brTStake)
}