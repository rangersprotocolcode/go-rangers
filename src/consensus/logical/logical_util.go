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
		d := time.Since(preBH.CurTime)
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
	return int(after.Sub(before).Seconds())/model.MAX_GROUP_BLOCK_TIME + 1
}
