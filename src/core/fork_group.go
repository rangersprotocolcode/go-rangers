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

package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/db"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/utility"
	"errors"
	"github.com/oleiade/lane"
)

var createBlockNotOnChain = errors.New("create block not on chain")
var verifyGroupErr = errors.New("verify group error")
var preGroupNilErr = errors.New("pre group is nil")
var parentGroupNilErr = errors.New("parent group is nil")

type groupChainFork struct {
	enableRcvGroup bool
	rcvLastGroup   bool
	pending        *lane.Queue

	waitingBlock bool
	header       uint64
	latestGroup  *types.Group

	current uint64
	db      db.Database

	lock   middleware.Loglock
	logger log.Logger
}

func newGroupChainFork(commonAncestor *types.Group) *groupChainFork {
	fork := &groupChainFork{header: commonAncestor.GroupHeight, current: commonAncestor.GroupHeight, latestGroup: commonAncestor, logger: syncLogger}
	fork.enableRcvGroup = true
	fork.rcvLastGroup = false
	fork.waitingBlock = false

	fork.pending = lane.NewQueue()
	fork.lock = middleware.NewLoglock("groupChainFork")
	fork.db = refreshGroupForkDB(*commonAncestor)
	fork.insertGroup(commonAncestor)
	return fork
}

func (fork *groupChainFork) isWaiting() bool {
	fork.lock.RLock("group chain fork isWaiting")
	defer fork.lock.RUnlock("group chain fork isWaiting")

	return fork.waitingBlock
}

func (fork *groupChainFork) rcv(group *types.Group, isLastGroup bool) (needMore bool) {
	fork.lock.Lock("group chain fork rcv")
	defer fork.lock.Unlock("group chain fork rcv")

	if !fork.enableRcvGroup {
		return false
	}
	fork.pending.Enqueue(group)
	fork.rcvLastGroup = isLastGroup
	if isLastGroup || fork.pending.Size() >= syncedGroupCount {
		fork.enableRcvGroup = false
		return false
	}
	return true
}

func (fork *groupChainFork) triggerOnFork(blockFork *blockChainFork) (err error, rcvLastGroup bool, tail *types.Group) {
	fork.lock.Lock("group chain fork triggerOnFork")
	defer fork.lock.Unlock("group chain fork triggerOnFork")

	fork.logger.Debugf("Trigger group on fork..")
	var group *types.Group
	for !fork.pending.Empty() {
		g := fork.pending.Head().(*types.Group)
		err = fork.addGroupOnFork(g, blockFork)
		if err != nil {
			fork.logger.Debugf("Group on fork failed!%s-%d", common.ToHex(g.Id), g.GroupHeight)
			break
		}
		fork.logger.Debugf("Group on fork success!%s-%d", common.ToHex(g.Id), g.GroupHeight)
		group = fork.pending.Pop().(*types.Group)
		fork.waitingBlock = false
	}

	if err == createBlockNotOnChain {
		fork.waitingBlock = true
		fork.logger.Debugf("Trigger group on fork paused. waiting block..")
	}

	if err != nil {
		return err, fork.rcvLastGroup, nil
	}

	if !fork.rcvLastGroup {
		fork.enableRcvGroup = true
	}
	return err, fork.rcvLastGroup, group
}

func (fork *groupChainFork) triggerOnChain(groupChain *groupChain) bool {
	fork.lock.Lock("group chain fork triggerOnChain")
	defer fork.lock.Unlock("group chain fork triggerOnChain")

	if fork.current == fork.header {
		groupChain.removeFromCommonAncestor(fork.getGroup(fork.header))
		fork.current++
	}
	fork.logger.Debugf("Trigger group on chain...current:%d,tail:%d", fork.current, fork.latestGroup.GroupHeight)
	for fork.current <= fork.latestGroup.GroupHeight {
		forkGroup := fork.getGroup(fork.current)
		if forkGroup == nil {
			return false
		}
		err := groupChain.AddGroup(forkGroup)
		if err == nil {
			fork.current++
			fork.logger.Debugf("add group on chain success.%d-%s", forkGroup.GroupHeight, common.ToHex(forkGroup.Id))
			continue
		} else {
			fork.logger.Debugf("add group on chain failed.%d-%s,err:%s", forkGroup.GroupHeight, common.ToHex(forkGroup.Id), err.Error())
			return false
		}
	}
	return true
}

func (fork *groupChainFork) destroy() {
	fork.lock.Lock("group chain fork destroy")
	defer fork.lock.Unlock("group chain fork destroy")
	for i := fork.header; i <= fork.latestGroup.GroupHeight; i++ {
		fork.deleteGroup(i)
	}
	fork.db.Delete([]byte(groupCommonAncestorHeightKey))
	fork.db.Delete([]byte(latestGroupHeightKey))
}

func (fork *groupChainFork) getGroupById(id []byte) *types.Group {
	bytes, _ := fork.db.Get(id)
	group, err := types.UnMarshalGroup(bytes)
	if err != nil {
		logger.Errorf("Fail to umMarshal group, error:%s", err.Error())
	}
	return group
}

func (fork *groupChainFork) addGroupOnFork(coming *types.Group, blockFork *blockChainFork) error {
	verifyResult, err := fork.verifyGroup(coming, blockFork)
	if verifyResult {
		fork.insertGroup(coming)
		fork.latestGroup = coming
	}
	fork.logger.Debugf("Verify group on fork failed.Id:%d,%s", common.ToHex(coming.Id), err.Error())
	return err
}

func (fork *groupChainFork) insertGroup(group *types.Group) error {
	groupByte, err := types.MarshalGroup(group)
	if err != nil {
		fork.logger.Errorf("Fail to marshal group, error:%s", err.Error())
		return err
	}
	err = fork.db.Put(generateHeightKey(group.GroupHeight), groupByte)
	if err != nil {
		fork.logger.Errorf("Fail to insert db, error:%s", err.Error())
		return err
	}

	err = fork.db.Put(group.Id, groupByte)
	if err != nil {
		fork.logger.Errorf("Fail to insert db, error:%s", err.Error())
		return err
	}
	return nil
}

func (fork *groupChainFork) getGroup(height uint64) *types.Group {
	bytes, _ := fork.db.Get(generateHeightKey(height))
	group, err := types.UnMarshalGroup(bytes)
	if err != nil {
		logger.Errorf("Fail to umMarshal group, error:%s", err.Error())
	}
	return group
}

func (fork *groupChainFork) deleteGroup(height uint64) bool {
	group := fork.getGroup(height)
	if group != nil {
		err := fork.db.Delete(group.Id)
		if err != nil {
			logger.Errorf("Fail to delete group, error:%s", err.Error())
			return false
		}
	}
	err := fork.db.Delete(generateHeightKey(height))
	if err != nil {
		logger.Errorf("Fail to delete group, error:%s", err.Error())
		return false
	}
	return true
}

func (fork *groupChainFork) verifyGroup(coming *types.Group, blockFork *blockChainFork) (bool, error) {
	var preGroup *types.Group
	if coming.GroupHeight > fork.header {
		preGroup = fork.getGroupById(coming.Header.PreGroup)
	} else {
		preGroup = groupChainImpl.getGroupById(coming.Header.PreGroup)
	}
	if preGroup == nil {
		return false, preGroupNilErr
	}

	var parentGroup = fork.getGroupById(coming.Header.Parent)
	if parentGroup == nil {
		parentGroup = groupChainImpl.getGroupById(coming.Header.Parent)
	}
	if parentGroup == nil {
		return false, parentGroupNilErr
	}

	createBlockHash := common.BytesToHash(coming.Header.CreateBlockHash)
	var baseBlock *types.Block
	if blockFork != nil {
		baseBlock = blockFork.getBlockByHash(createBlockHash)
	}
	if baseBlock == nil {
		baseBlock = blockChainImpl.queryBlockByHash(createBlockHash)
	}
	if baseBlock == nil {
		return false, common.ErrCreateBlockNil
	}
	return consensusHelper.VerifyGroupForFork(coming, preGroup, parentGroup, baseBlock)
}

func refreshGroupForkDB(commonAncestor types.Group) db.Database {
	db, _ := db.NewDatabase(groupForkDBPrefix)

	startBytes, _ := db.Get([]byte(groupCommonAncestorHeightKey))
	start := utility.ByteToUInt64(startBytes)
	endBytes, _ := db.Get([]byte(latestGroupHeightKey))
	end := utility.ByteToUInt64(endBytes)
	for i := start; i <= end; i++ {
		db.Delete(generateHeightKey(i))
	}

	db.Put([]byte(blockCommonAncestorHeightKey), utility.UInt64ToByte(commonAncestor.GroupHeight))
	db.Put([]byte(latestBlockHeightKey), utility.UInt64ToByte(commonAncestor.GroupHeight))
	return db
}

func (iterator *GroupForkIterator) Current() *types.Group {
	return iterator.current
}

func (iterator *GroupForkIterator) MovePre() *types.Group {
	if SyncProcessor == nil || SyncProcessor.groupFork == nil || iterator.current == nil {
		return nil
	}
	preGroupId := iterator.current.Header.PreGroup
	if iterator.current.GroupHeight > SyncProcessor.groupFork.header {
		iterator.current = SyncProcessor.groupFork.getGroupById(preGroupId)
	} else {
		iterator.current = groupChainImpl.GetGroupById(iterator.current.Header.PreGroup)
	}
	return iterator.current
}
