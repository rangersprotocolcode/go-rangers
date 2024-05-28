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
	"com.tuntun.rangers/node/src/consensus/access"
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/consensus/net"
	"com.tuntun.rangers/node/src/consensus/ticker"
	"com.tuntun.rangers/node/src/core"
	"com.tuntun.rangers/node/src/middleware"
	"com.tuntun.rangers/node/src/middleware/log"
	"com.tuntun.rangers/node/src/middleware/notify"
	"com.tuntun.rangers/node/src/middleware/types"
	"encoding/hex"
	"fmt"
	lru "github.com/hashicorp/golang-lru"
	"strconv"
	"sync"
	"sync/atomic"
)

type Processor struct {
	ready bool
	conf  common.ConfManager
	mi    *model.SelfMinerInfo

	belongGroups *access.JoinedGroupStorage
	globalGroups *access.GroupAccessor

	minerReader *access.MinerPoolReader
	//groupManager *GroupManager

	Ticker *ticker.GlobalTicker
	vrf    atomic.Value

	MainChain  core.BlockChain
	GroupChain core.GroupChain
	NetServer  net.NetworkServer

	lock sync.Mutex

	partyLock                     middleware.Loglock
	partyManager                  map[string]Party
	logger                        log.Logger
	finishedParty, futureMessages *lru.Cache
}

func (p *Processor) getPrefix() string {
	return p.GetMinerID().ShortS()
}

func (p *Processor) Init(mi model.SelfMinerInfo, conf common.ConfManager, joinedGroupStorage *access.JoinedGroupStorage) bool {
	p.ready = false
	p.lock = sync.Mutex{}

	p.partyManager = make(map[string]Party, 10)
	p.partyLock = middleware.NewLoglock("partyLock")
	p.logger = log.GetLoggerByIndex(log.CLogConfig, strconv.Itoa(common.InstanceIndex))
	p.finishedParty = common.CreateLRUCache(300)
	p.futureMessages = common.CreateLRUCache(50)

	p.conf = conf

	p.MainChain = core.GetBlockChain()
	p.GroupChain = core.GetGroupChain()
	p.mi = &mi
	p.globalGroups = access.NewGroupAccessor(p.GroupChain)
	p.belongGroups = joinedGroupStorage
	p.NetServer = net.NewNetworkServer()

	p.minerReader = access.NewMinerPoolReader()

	p.Ticker = ticker.GetTickerInstance()

	if stdLogger != nil {
		stdLogger.Debugf("proc(%v) inited 2.\n", p.getPrefix())
		consensusLogger.Infof("ProcessorId:%v", p.getPrefix())
	}

	notify.BUS.Subscribe(notify.BlockAddSucc, p)
	notify.BUS.Subscribe(notify.GroupAddSucc, p)
	notify.BUS.Subscribe(notify.AcceptGroup, p)

	return true
}

func (p *Processor) HandleNetMessage(topic string, msg notify.Message) {
	switch topic {
	case notify.BlockAddSucc:
		p.onBlockAddSuccess(msg)
	case notify.GroupAddSucc:
		p.onGroupAddSuccess(msg)
	case notify.AcceptGroup:
		p.onGroupAccept(msg)
	}
}

func (p *Processor) GetMinerID() groupsig.ID {
	return p.mi.GetMinerID()
}

func (p *Processor) GetMinerInfo() *model.MinerInfo {
	return &p.mi.MinerInfo
}

func (p *Processor) isCastLegal(bh *types.BlockHeader, preHeader *types.BlockHeader) (ok bool, group *model.GroupInfo, err error) {
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

	var gid = groupsig.DeserializeID(bh.GroupId)

	selectGroupIdFromCache := p.CalcVerifyGroupFromCache(preHeader, bh.CurTime, bh.Height)

	if selectGroupIdFromCache == nil {
		err = common.ErrSelectGroupNil
		stdLogger.Debugf("selectGroupId is nil")
		return
	}
	var verifyGid = *selectGroupIdFromCache

	if !selectGroupIdFromCache.IsEqual(gid) {
		selectGroupIdFromChain := p.CalcVerifyGroupFromChain(preHeader, bh.CurTime, bh.Height)
		if selectGroupIdFromChain == nil {
			err = common.ErrSelectGroupNil
			return
		}

		if !selectGroupIdFromChain.IsEqual(*selectGroupIdFromCache) {
			go p.updateGlobalGroups()
		}
		if !selectGroupIdFromChain.IsEqual(gid) {
			err = common.ErrSelectGroupInequal
			stdLogger.Debugf("selectGroupId from both cache and chain not equal, expect %v, receive %v.bh hash:%s,height:%d,castor:%s", selectGroupIdFromChain.ShortS(), gid.ShortS(), bh.Hash.String(), bh.Height, hex.EncodeToString(bh.Castor))
			return
		}
		verifyGid = *selectGroupIdFromChain
	}

	group = p.GetGroup(verifyGid)
	if !group.GroupID.IsValid() {
		err = fmt.Errorf("selectedGroup is not valid, expect gid=%v, real gid=%v", verifyGid.ShortS(), group.GroupID.ShortS())
		return
	}

	ok = true
	return
}

func (p *Processor) getMinerPos(gid groupsig.ID, uid groupsig.ID) int32 {
	sgi := p.GetGroup(gid)
	return int32(sgi.GetMemberPosition(uid))
}

func (p *Processor) GetGroup(gid groupsig.ID) *model.GroupInfo {
	if g, err := p.globalGroups.GetGroupByID(gid); err != nil {
		panic("GetSelfGroup failed.")
	} else {
		return g
	}
}

func (p *Processor) getGroupPubKey(gid groupsig.ID) groupsig.Pubkey {
	if g, err := p.globalGroups.GetGroupByID(gid); err != nil {
		panic("GetSelfGroup failed.")
	} else {
		return g.GetGroupPubKey()
	}

}

// getSignKey get the signature private key of the miner in a certain group
func (p *Processor) getSignKey(gid groupsig.ID) groupsig.Seckey {
	if jg := p.belongGroups.GetJoinedGroupInfo(gid); jg != nil {
		return jg.SignSecKey
	}
	return groupsig.Seckey{}
}
