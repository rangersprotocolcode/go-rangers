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

package service

import (
	"bytes"
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/middleware/log"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/storage/account"
	"com.tuntun.rangers/node/src/utility"
	"fmt"
	"github.com/pkg/errors"
	"math"
	"math/big"
	"sort"
	"strconv"
)

const refundHeight = 36000

type RefundManager struct {
	logger           log.Logger
	groupChainHelper types.GroupChainHelper
	forkHelper       types.ForkHelper
}

var (
	RefundManagerImpl *RefundManager
	prefix            = "refund"
)

func InitRefundManager(groupChainHelper types.GroupChainHelper, forkHelper types.ForkHelper) {
	RefundManagerImpl = &RefundManager{}
	RefundManagerImpl.logger = log.GetLoggerByIndex(log.RefundLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	RefundManagerImpl.groupChainHelper = groupChainHelper
	RefundManagerImpl.forkHelper = forkHelper
}

func (refund *RefundManager) generateAddress(height uint64) common.Address {
	keyString := prefix + strconv.FormatUint(height, 10)
	return common.BytesToAddress(common.Sha256(utility.StrToBytes(keyString)))
}

func (refund *RefundManager) CheckAndMove(height uint64, db *account.AccountDB) {
	if nil == db {
		return
	}

	address := refund.generateAddress(height)

	refundList := db.GetAllRefund(address)
	if nil == refundList || 0 == len(refundList) {
		refund.logger.Debugf("no refundList for height: %d", height)
		return
	}

	for addr, value := range refundList {
		db.AddBalance(addr, value)
		db.RemoveData(address, addr.Bytes())
		refund.logger.Warnf("refunded, height: %d, address: %s, delta: %d", height, addr.String(), value)
	}
}

func (refund *RefundManager) Add(data map[uint64]types.RefundInfoList, db *account.AccountDB) {
	if nil == db || nil == data || 0 == len(data) {
		return
	}

	for height, list := range data {
		if list.IsEmpty() {
			continue
		}

		address := refund.generateAddress(height)
		for _, refundInfo := range list.List {
			existedBytes := db.GetData(address, refundInfo.Id)
			if nil == existedBytes || 0 == len(existedBytes) {
				db.SetData(address, refundInfo.Id, refundInfo.Value.Bytes())
				refund.logger.Debugf("height: %d, set address: %s, value: %s", height, common.ToHex(refundInfo.Id), refundInfo.Value)
			} else {
				existed := new(big.Int).SetBytes(existedBytes)
				refund.logger.Debugf("height: %d, add address: %s, value: %s, existed: %s", height, common.ToHex(refundInfo.Id), refundInfo.Value, existed.String())

				existed.Add(existed, refundInfo.Value)
				db.SetData(address, refundInfo.Id, existed.Bytes())
				refund.logger.Debugf("height: %d, after, add address: %s, value: %s, existed: %s", height, common.ToHex(refundInfo.Id), refundInfo.Value, existed.String())
			}
		}
	}
}

func (this *RefundManager) GetRefundStake(now uint64, minerId, account []byte, money uint64, accountdb *account.AccountDB, situation string) (uint64, *big.Int, []byte, error) {
	this.logger.Debugf("getRefund, minerId:%s, height: %d, money: %d", common.ToHex(minerId), now, money)
	miner := MinerManagerImpl.GetMiner(minerId, accountdb)
	if nil == miner {
		this.logger.Errorf("getRefund error, minerId:%s, height: %d, money: %d, miner not existed", common.ToHex(minerId), now, money)
		return 0, nil, nil, errors.New("miner not existed")
	}

	if 0 != bytes.Compare(account, miner.Account) {
		msg := fmt.Sprintf("getRefund error, minerId:%s, height: %d, money: %d, auth error. account: %s vs except: %s", common.ToHex(minerId), now, money, common.ToHex(account), common.ToHex(miner.Account))
		this.logger.Errorf(msg)
		return 0, nil, nil, errors.New(msg)
	}

	if money == math.MaxUint64 {
		money = miner.Stake
	}

	// 超出了质押量，不能提
	if miner.Stake < money {
		this.logger.Errorf("getRefund error, minerId:%s, height: %d, money: %d, not enough stake. stake: %d", common.ToHex(minerId), now, money, miner.Stake)
		return 0, nil, nil, errors.New("not enough stake")
	}

	refund := money
	left := miner.Stake - money
	// 验证小于最小质押量，则退出矿工
	if miner.Type == common.MinerTypeProposer && left < common.ProposerStake ||
		miner.Type == common.MinerTypeValidator && left < common.ValidatorStake {
		MinerManagerImpl.RemoveMiner(minerId, account, miner.Type, accountdb, left)
	} else {
		// update miner
		miner.Stake = left
		MinerManagerImpl.UpdateMiner(miner, accountdb, false)
	}

	height := this.getRefundHeight(now, left, miner.Type, minerId, situation)

	this.logger.Debugf("getRefund end, minerId: %s, height: %d, money: %d", common.ToHex(minerId), height, refund)
	return height, utility.Uint64ToBigInt(refund), miner.Account, nil
}

// 计算解锁高度
func (this *RefundManager) getRefundHeight(now, left uint64, minerType byte, minerId []byte, situation string) uint64 {
	height := uint64(0)
	if common.IsProposal012() {
		return now + refundHeight
	}
	// 验证节点，计算最多能加入的组数，来确定解锁块高
	if minerType == common.MinerTypeValidator {
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

			base := dismissHeightList[delta-1]
			if base != math.MaxUint64 {
				height = base + common.RefundBlocks
			}

		}
	} else {
		height = RewardCalculatorImpl.NextRewardHeight(now) + common.RefundBlocks
	}

	if common.IsProposal004() && height <= 0 {
		height = now + common.RefundBlocks*100
	}

	if common.LocalChainConfig.Proposal011Block == now {
		height = height - 50
	}
	return height
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
