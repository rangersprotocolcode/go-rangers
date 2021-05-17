package logical

import (
	"bytes"
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"com.tuntun.rocket/node/src/middleware/types"
	"encoding/hex"
	"fmt"
	"time"
)

//检查提案节点是否合法
func (p *Processor) IsCastLegalForFork(bh *types.BlockHeader, preHeader *types.BlockHeader) (ok bool, err error) {
	blog := newBizLog("isCastLegal")
	castor := groupsig.DeserializeID(bh.Castor)
	minerDO := p.minerReader.GetProposeMiner(castor, preHeader.StateTree)
	if minerDO == nil {
		err = fmt.Errorf("minerDO is nil, id=%v", castor.ShortS())
		return
	}
	if !minerDO.CanCastAt(bh.Height) {
		err = fmt.Errorf("miner can't cast at height, id=%v, height=%v(%v-%v)", castor.ShortS(), bh.Height, minerDO.ApplyHeight, minerDO.AbortHeight)
		return
	}
	totalStake := p.minerReader.GetTotalStake(preHeader.Height, preHeader.StateTree)
	blog.log("totalStake %v", totalStake)
	if ok2, err2 := verifyBlockVRF(bh, preHeader, minerDO, totalStake); !ok2 {
		err = fmt.Errorf("vrf verify block fail, err=%v", err2)
		return
	}
	selectedGroupFromFork := p.CalcVerifyGroupFromFork(preHeader, bh.CurTime, bh.Height)
	if selectedGroupFromFork == nil {
		err = common.ErrSelectGroupNil
		stdLogger.Debugf("selectGroupId is nil")
		return
	}

	if !bytes.Equal(selectedGroupFromFork.Id, bh.GroupId) { //有可能组已经解散，需要再从链上取
		err = common.ErrSelectGroupInequal
		stdLogger.Debugf("selectGroupId from fork not equal, expect %v, receive %v.bh hash:%s,height:%d,castor:%s", common.ToHex(selectedGroupFromFork.Id), common.ToHex(bh.GroupId), bh.Hash.String(), bh.Height, hex.EncodeToString(bh.Castor))
		return
	}
	ok = true
	return
}

func (p *Processor) CalcVerifyGroupFromFork(preBH *types.BlockHeader, castTime time.Time, height uint64) *types.Group {
	var hash = CalcRandomHash(preBH, castTime)

	selectGroup, err := p.globalGroups.SelectVerifyGroupFromFork(hash, height)
	if err != nil {
		stdLogger.Errorf("CalcVerifyGroupFromFork height=%v, err:%v", height, err)
		return nil
	}
	return selectGroup
}
