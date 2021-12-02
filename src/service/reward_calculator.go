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
	"math"
	"math/big"
)

type RewardCalculator struct {
	minerManager *MinerManager
	blockChain   types.BlockChainHelper
	groupChain   types.GroupChainHelper
	forkHelper   types.ForkHelper
	logger       log.Logger
}

var RewardCalculatorImpl *RewardCalculator

func InitRewardCalculator(blockChainImpl types.BlockChainHelper, groupChain types.GroupChainHelper, forkHelper types.ForkHelper) {
	RewardCalculatorImpl = &RewardCalculator{minerManager: MinerManagerImpl, blockChain: blockChainImpl, groupChain: groupChain, forkHelper: forkHelper}
	RewardCalculatorImpl.logger = log.GetLoggerByIndex(log.RewardLogConfig, common.GlobalConf.GetString("instance", "index", ""))
}

func GetTotalReward(height uint64) float64 {
	return getTotalReward(height)
}

func (reward *RewardCalculator) CalculateReward(height uint64, accountDB *account.AccountDB, bh *types.BlockHeader, situation string) map[uint64]types.RefundInfoList {
	reward.logger.Debugf("start to calculate, height: %d, situation: %s", height, situation)
	defer reward.logger.Debugf("end to calculate, height: %d, situation: %s", height, situation)

	total := reward.calculateRewardPerBlock(bh, accountDB, situation)
	if nil == total || 0 == len(total) {
		reward.logger.Errorf("fail to reward, height: %d", height)
		return nil
	}

	nextHeight := reward.NextRewardHeight(height)
	refundInfoList := types.RefundInfoList{}
	for addr, money := range total {
		refundInfoList.AddRefundInfo(addr.Bytes(), money)
		reward.logger.Debugf("add reward, addr: %s, delta: %s, expected height: %d", addr.String(), money.String(), nextHeight)
	}

	data := make(map[uint64]types.RefundInfoList, 1)
	data[nextHeight] = refundInfoList
	return data
}

// 计算某一块的奖励
func (reward *RewardCalculator) calculateRewardPerBlock(bh *types.BlockHeader, accountDB *account.AccountDB, situation string) map[common.Address]*big.Int {
	result := make(map[common.Address]*big.Int)

	height := bh.Height
	total := getTotalReward(height)
	hashString := bh.Hash.String()
	reward.logger.Debugf("start to calculate, height: %d, hash: %s, proposer: %s, groupId: %s, totalReward %f", height, hashString, common.ToHex(bh.Castor), common.ToHex(bh.GroupId), total)
	defer reward.logger.Warnf("end to calculate, height %d, hash: %s, result: %v", height, hashString, result)

	// 提案者奖励
	rewardProposer := utility.Float64ToBigInt(total * common.ProposerReward)
	proposerAddr := common.BytesToAddress(MinerManagerImpl.getMinerAccount(bh.Castor, common.MinerTypeProposer, accountDB))
	addReward(result, proposerAddr, rewardProposer)
	proposerResult := result[proposerAddr].String()
	reward.logger.Debugf("calculating, height: %d, hash: %s, proposerAddr: %s, reward: %d, result: %s", height, hashString, proposerAddr.String(), rewardProposer, proposerResult)

	// 其他提案者奖励
	otherRewardProposer := total * common.AllProposerReward
	totalProposerStake, proposersStake := reward.minerManager.GetProposerTotalStakeWithDetail(height, accountDB)
	if totalProposerStake != 0 {
		for addr, stake := range proposersStake {
			delta := utility.Float64ToBigInt(float64(stake) / float64(totalProposerStake) * otherRewardProposer)
			addReward(result, addr, delta)
			proposerResult := result[addr].String()
			reward.logger.Debugf("calculating, height: %d, hash: %s, proposerAddr: %s, stake: %d, reward: %d, result: %s", height, hashString, addr.String(), stake, delta, proposerResult)
		}
	}

	// 验证者奖励
	var group *types.Group
	if situation != "fork" {
		group = reward.groupChain.GetGroupById(bh.GroupId)
	} else {
		group = reward.forkHelper.GetGroupById(bh.GroupId)
	}
	if group == nil {
		reward.logger.Errorf("fail to get group. id: %v", bh.GroupId)
		return nil
	}

	totalValidatorStake, validatorStake := reward.minerManager.GetValidatorsStake(group.Members, accountDB)
	if totalValidatorStake != 0 {
		rewardValidators := total * common.ValidatorsReward
		for addr, stake := range validatorStake {
			result[addr] = utility.Float64ToBigInt(float64(stake) / float64(totalValidatorStake) * rewardValidators)
			reward.logger.Debugf("calculating, height: %d, hash: %s, validatorAddr %s, stake: %d, reward %d", height, hashString, addr.String(), stake, result[addr])
		}
	}

	return result
}

func (reward *RewardCalculator) NextRewardHeight(height uint64) uint64 {
	next := math.Ceil(float64(height) / float64(common.RewardBlocks))
	return uint64(next) * common.RewardBlocks
}

func getEpoch(height uint64) uint64 {
	return height / common.BlocksPerEpoch
}

func getTotalReward(height uint64) float64 {
	return common.TotalRPGSupply * math.Pow(1-common.ReleaseRate, float64(getEpoch(height))) * common.ReleaseRate / common.BlocksPerEpoch
}

func addReward(all map[common.Address]*big.Int, addr common.Address, delta *big.Int) {
	old, ok := all[addr]
	if ok {
		old.Add(old, delta)
	} else {
		nb := &big.Int{}
		nb.SetBytes(delta.Bytes())
		all[addr] = nb
	}
}
