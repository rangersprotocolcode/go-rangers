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
	"com.tuntun.rocket/node/src/middleware/db"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/utility"
	"errors"
	"github.com/oleiade/lane"
)

var createBlockNotOnChain = errors.New("create block not on chain")
var verifyGroupErr = errors.New("verify group error")

type groupChainFork struct {
	waitingBlock   bool
	enableRcvGroup bool
	rcvLastGroup   bool
	header         uint64
	current        uint64

	latestGroup *types.Group
	pending     *lane.Queue

	db     db.Database
	logger log.Logger
}

func newGroupChainFork(commonAncestor *types.Group) *groupChainFork {
	fork := &groupChainFork{header: commonAncestor.GroupHeight, current: commonAncestor.GroupHeight, latestGroup: commonAncestor, logger: syncLogger}
	fork.enableRcvGroup = true
	fork.rcvLastGroup = false
	fork.waitingBlock = false

	fork.pending = lane.NewQueue()
	fork.db = refreshGroupForkDB(*commonAncestor)
	fork.insertGroup(commonAncestor)
	return fork
}

func (fork *groupChainFork) acceptGroup(coming *types.Group) error {
	//todo verify group
	fork.insertGroup(coming)
	fork.latestGroup = coming
	return nil
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
	err := fork.db.Delete(generateHeightKey(height))
	if err != nil {
		logger.Errorf("Fail to delete group, error:%s", err.Error())
		return false
	}
	return true
}

func (fork *groupChainFork) destroy() {
	for i := fork.header; i <= fork.latestGroup.GroupHeight; i++ {
		fork.deleteGroup(i)
	}
	fork.db.Delete([]byte(groupCommonAncestorHeightKey))
	fork.db.Delete([]byte(latestGroupHeightKey))
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
