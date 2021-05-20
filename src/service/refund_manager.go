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

package service

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"encoding/json"
	"github.com/pkg/errors"
	"math"
	"math/big"
	"sort"
)

type RefundManager struct {
	logger           log.Logger
	groupChainHelper types.GroupChainHelper
	forkHelper       types.ForkHelper
}

var RefundManagerImpl *RefundManager

func InitRefundManager(groupChainHelper types.GroupChainHelper, forkHelper types.ForkHelper) {
	RefundManagerImpl = &RefundManager{}
	RefundManagerImpl.logger = log.GetLoggerByIndex(log.RefundLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	RefundManagerImpl.groupChainHelper = groupChainHelper
	RefundManagerImpl.forkHelper = forkHelper
}

func (refund *RefundManager) CheckAndMove(height uint64, db *account.AccountDB) {
	if nil == db {
		return
	}

	key := utility.UInt64ToByte(height)
	data := db.GetData(common.RefundAddress, key)
	if nil == data || 0 == len(data) {
		refund.logger.Warnf("no data at height: %d", height)
		return
	}

	var refundInfoList types.RefundInfoList
	err := json.Unmarshal(data, &refundInfoList)
	if err != nil {
		refund.logger.Errorf("fail to unmarshal", err.Error())
		return
	}

	for _, refundInfo := range refundInfoList.List {
		addr := common.BytesToAddress(refundInfo.Id)
		db.AddBalance(addr, refundInfo.Value)
		refund.logger.Warnf("refunded, height: %d, address: %s, delta: %d", height, addr.String(), refundInfo.Value)
	}

	db.RemoveData(common.RefundAddress, key)
}

func (refund *RefundManager) Add(data map[uint64]types.RefundInfoList, db *account.AccountDB) {
	if nil == db {
		return
	}

	for height, list := range data {
		if list.IsEmpty() {
			continue
		}

		// 查询一下
		existedBytes := db.GetData(common.RefundAddress, utility.UInt64ToByte(height))
		if nil == existedBytes || 0 == len(existedBytes) {
			db.SetData(common.RefundAddress, utility.UInt64ToByte(height), list.TOJSON())
			refund.logger.Warnf("add RefundInfoList: %v, height: %d", list, height)
			continue
		}

		// 已有数据，需要叠加
		var refundInfoList types.RefundInfoList
		err := json.Unmarshal(existedBytes, &refundInfoList)
		if err != nil {
			refund.logger.Errorf("fail to unmarshal", err.Error())
			continue
		}

		for _, item := range list.List {
			refundInfoList.AddRefundInfo(item.Id, item.Value)
		}
		db.SetData(common.RefundAddress, utility.UInt64ToByte(height), refundInfoList.TOJSON())
		refund.logger.Warnf("add RefundInfoList: %v, height: %d", refundInfoList, height)
	}
}

func (this *RefundManager) GetRefundStake(now uint64, minerId []byte, money uint64, accountdb *account.AccountDB, situation string) (uint64, *big.Int, error) {
	this.logger.Debugf("getRefund, minerId:%s, height: %d, money: %d", common.ToHex(minerId), now, money)
	miner := MinerManagerImpl.GetMiner(minerId, accountdb)
	if nil == miner {
		this.logger.Debugf("getRefund error, minerId:%s, height: %d, money: %d, miner not existed", common.ToHex(minerId), now, money)
		return 0, nil, errors.New("miner not existed")
	}

	// 超出了质押量，不能提
	if miner.Stake < money {
		this.logger.Debugf("getRefund error, minerId:%s, height: %d, money: %d, not enough stake. stake: %d", common.ToHex(minerId), now, money, miner.Stake)
		return 0, nil, errors.New("not enough stake")
	}

	refund := money
	left := miner.Stake - money
	// 验证小于最小质押量，则退出矿工
	if miner.Type == common.MinerTypeProposer && left < common.ProposerStake ||
		miner.Type == common.MinerTypeValidator && left < common.ValidatorStake {
		MinerManagerImpl.RemoveMiner(minerId, miner.Type, accountdb)
		refund = miner.Stake
	} else {
		// update miner
		miner.Stake = left
		MinerManagerImpl.UpdateMiner(miner, accountdb)
	}

	// 计算解锁高度
	height := RewardCalculatorImpl.NextRewardHeight(now) + common.RefundBlocks

	// 验证节点，计算最多能加入的组数，来确定解锁块高
	if miner.Type == common.MinerTypeValidator {
		// 检查当前加入了多少组
		var groups []*types.Group
		if situation != "fork" {
			groups = this.groupChainHelper.GetAvailableGroupsByMinerId(now, minerId)
		} else {
			groups = this.forkHelper.GetAvailableGroupsByMinerId(now, minerId)
		}
		// 扣完质押之后，还能加入多少组
		leftGroups := int(left / common.ValidatorStake)
		delta := len(groups) - leftGroups

		// 按照退组的信息决定解冻信息
		if delta > 0 {
			dismissHeightList := DismissHeightList{}
			for _, group := range groups {
				dismissHeightList = append(dismissHeightList, group.Header.DismissHeight)
			}
			sort.Sort(dismissHeightList)
			height = dismissHeightList[delta-1] + common.RefundBlocks
		}
	}

	if height < 0 {
		height = math.MaxUint64
	}
	this.logger.Debugf("getRefund end, minerId:%s, height: %d, money: %d", common.ToHex(minerId), height, refund)
	return height, utility.Uint64ToBigInt(refund), nil
}

type DismissHeightList []uint64

func (c DismissHeightList) Len() int {
	return len(c)
}
func (c DismissHeightList) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
func (c DismissHeightList) Less(i, j int) bool {
	return c[i] < c[j]
}
