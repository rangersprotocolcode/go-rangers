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
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

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

	if !bytes.Equal(selectedGroupFromFork.Id, bh.GroupId) {
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
