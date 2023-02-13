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
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/base"
	"com.tuntun.rocket/node/src/consensus/model"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/utility"
	"time"
)

func GetCastExpireTime(base time.Time, deltaHeight uint64, castHeight uint64) time.Time {
	t := uint64(0)
	if castHeight == 1 { //铸高度1的时候，过期时间为5倍，以防节点启动不同步时，先提案的块过早过期导致同一节点对高度1提案多次
		t = 2
	}
	return base.Add(time.Second * time.Duration((t+deltaHeight)*uint64(model.Param.MaxGroupCastTime)))
}

func DeltaHeightByTime(bh *types.BlockHeader, preBH *types.BlockHeader) uint64 {
	var (
		deltaHeightByTime uint64
	)
	if bh.Height == 1 {
		d := utility.GetTime().Sub(preBH.CurTime)
		deltaHeightByTime = uint64(d.Seconds())/uint64(model.Param.MaxGroupCastTime) + 1
	} else {
		deltaHeightByTime = bh.Height - preBH.Height
	}
	return deltaHeightByTime
}

func VerifyBHExpire(bh *types.BlockHeader, preBH *types.BlockHeader) (time.Time, bool) {
	expire := GetCastExpireTime(preBH.CurTime, DeltaHeightByTime(bh, preBH), bh.Height)
	return expire, utility.GetTime().After(expire)
}
func CalcRandomHash(preBH *types.BlockHeader, castTime time.Time) common.Hash {
	stdLogger.Debugf("cal random.cast time:%s,pre time:%s,pre hash:%s", castTime.String(), preBH.CurTime.String(), preBH.Hash.String())
	data := preBH.Random
	var hash common.Hash

	deltaHeight := CalDeltaByTime(castTime, preBH.CurTime)
	for ; deltaHeight > 0; deltaHeight-- {
		hash = base.Data2CommonHash(data)
		data = hash.Bytes()
	}
	return hash
}

func IsGroupDissmisedAt(gh *types.GroupHeader, h uint64) bool {
	return gh.DismissHeight <= h
}
func IsGroupWorkQualifiedAt(gh *types.GroupHeader, h uint64) bool {
	return !IsGroupDissmisedAt(gh, h) && gh.WorkHeight <= h
}

func CalDeltaByTime(after time.Time, before time.Time) int {
	// 创世块可能年代久远，导致第一块需要多次计算Hash引发出块超时
	// 这里特殊处理
	result := int(after.Sub(before).Seconds())/model.MAX_GROUP_BLOCK_TIME + 1
	//if result > 8 {
	//	result = 8
	//}

	return result
}
