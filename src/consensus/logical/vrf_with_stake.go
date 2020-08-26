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
	"fmt"
	"math"
	"math/big"

	"com.tuntun.rocket/node/src/consensus/base"
	"com.tuntun.rocket/node/src/consensus/model"
	"com.tuntun.rocket/node/src/consensus/vrf"
	"com.tuntun.rocket/node/src/middleware/types"
)

var rat1 *big.Rat
var max256 *big.Rat
var maxQn *big.Rat

func init() {
	t := new(big.Int)
	t.SetString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)
	max256 = new(big.Rat).SetInt(t)
	rat1 = new(big.Rat).SetInt64(1)
	maxQn = new(big.Rat).SetInt64(int64(model.Param.MaxQN))
}

func verifyBlockVRF(bh *types.BlockHeader, preBH *types.BlockHeader, castor *model.MinerInfo, totalStake uint64) (bool, error) {
	prove := vrf.VRFProve(bh.ProveValue.Bytes())
	delta := CalDeltaByTime(bh.CurTime, preBH.CurTime)
	ok, err := vrf.VRFVerify(castor.VrfPK, prove, genVrfMsg(preBH.Random, delta))
	if !ok {
		return ok, err
	}
	if ok, qn := validateProve(prove, castor.Stake, totalStake); ok {
		if bh.TotalQN != qn+preBH.TotalQN {
			return false, errors.New(fmt.Sprintf("qn error.bh hash=%v, height=%v, qn=%v,totalQN=%v, preBH totalQN=%v", bh.Hash.ShortS(), bh.Height, qn, bh.TotalQN, preBH.TotalQN))
		}
		return true, nil
	}
	return false, errors.New("proof not satisfy")
}

//func genVrfMsg(random []byte, deltaHeight uint64) []byte {
//	msg := random
//	for deltaHeight > 1 {
//		deltaHeight--
//		msg = base.Data2CommonHash(msg).Bytes()
//	}
//	return msg
//}

func genVrfMsg(random []byte, delta int) []byte {
	msg := random
	for delta > 1 {
		delta--
		msg = base.Data2CommonHash(msg).Bytes()
	}
	return msg
}

func validateProve(prove vrf.VRFProve, stake uint64, totalStake uint64) (ok bool, qn uint64) {
	if totalStake == 0 {
		stdLogger.Errorf("total stake is 0!")
		return false, 0
	}
	blog := newBizLog("vrfSatisfy")
	vrfValueRatio := vrfValueRatio(prove)
	stakeRatio := stakeRatio(1, totalStake)
	ok = vrfValueRatio.Cmp(stakeRatio) < 0

	qn = calQn(vrfValueRatio, stakeRatio)

	vrfValueRatioFloat, _ := vrfValueRatio.Float64()
	stakeRatioFloat, _ := stakeRatio.Float64()
	blog.log("Vrf validate result:%v! miner stake %v, total stake %v, vrf value ratio %v, stake ratio %v,  qn %v", ok, 1, totalStake, vrfValueRatioFloat, stakeRatioFloat, qn)
	return
}

func stakeRatio(stake, totalStake uint64) *big.Rat {
	stakeRat := new(big.Rat).SetInt64(int64(stake * uint64(model.Param.PotentialProposal)))
	totalStakeRat := new(big.Rat).SetFloat64(float64(totalStake))
	return new(big.Rat).Quo(stakeRat, totalStakeRat)
}

func vrfValueRatio(prove vrf.VRFProve) *big.Rat {
	vrfValue := vrf.VRFProof2Hash(prove)
	vrfRat := new(big.Rat).SetInt(new(big.Int).SetBytes(vrfValue))
	return new(big.Rat).Quo(vrfRat, max256)
}

func calQn(vrfValueRatio, stakeRatio *big.Rat) uint64 {
	if stakeRatio.Cmp(rat1) > 0 {
		stakeRatio.Set(rat1)
	}
	step := new(big.Rat).Quo(stakeRatio, maxQn)
	r, _ := new(big.Rat).Quo(vrfValueRatio, step).Float64()
	qn := uint64(math.Floor(r) + 1)
	return qn
}
