package logical

import (
	"time"
	"x/src/middleware/types"
	"x/src/consensus/model"
	"x/src/common"
	"x/src/consensus/base"
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
	return expire, time.Now().After(expire)
}
func CalcRandomHash(preBH *types.BlockHeader, height uint64) common.Hash {
	data := preBH.Random
	var hash common.Hash

	deltaHeight := height - preBH.Height
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
