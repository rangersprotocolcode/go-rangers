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
	"com.tuntun.rocket/node/src/network"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"encoding/json"
	"math"
	"math/big"
	"strconv"
)

type RewardCalculator struct {
	minerManager *MinerManager
	blockChain   types.BlockChainHelper
	groupChain   types.GroupChainHelper
	fork         types.ForkHelper
	logger       log.Logger
}

var RewardCalculatorImpl *RewardCalculator

func InitRewardCalculator(blockChainImpl types.BlockChainHelper, groupChain types.GroupChainHelper, fork types.ForkHelper) {
	RewardCalculatorImpl = &RewardCalculator{minerManager: MinerManagerImpl, blockChain: blockChainImpl, groupChain: groupChain, fork: fork}
	RewardCalculatorImpl.logger = log.GetLoggerByIndex(log.RewardLogConfig, common.GlobalConf.GetString("instance", "index", ""))
}

func (reward *RewardCalculator) CalculateReward(height uint64, db *account.AccountDB, situation string) bool {
	reward.logger.Warnf("cal reward, height: %d", height)
	if !reward.needReward(height) {
		reward.logger.Warnf("no need to reward, height: %d", height)
		return false
	}

	total := reward.calculateReward(height, situation)
	if nil == total || 0 == len(total) {
		reward.logger.Errorf("fail to reward, height: %d", height)
		return false
	}

	go reward.notify(total, height)

	for addr, money := range total {
		from := db.GetBalance(addr).String()
		db.AddBalance(addr, money)
		reward.logger.Debugf("add reward, addr: %s, from: %s to %v", addr.String(), from, db.GetBalance(addr))
	}
	return true
}

// send reward detail
func (reward *RewardCalculator) notify(total map[common.Address]*big.Int, height uint64) {
	result := make(map[string]interface{})
	result["from"] = strconv.FormatUint(height-common.RewardBlocks, 10)
	result["to"] = strconv.FormatUint(height, 10)
	data := make(map[string]string, len(total))
	for addr, balance := range total {
		data[addr.GetHexString()] = utility.BigIntToStr(balance)
	}
	result["data"] = data

	resultByte, _ := json.Marshal(result)
	network.GetNetInstance().Notify(false, "rocketprotocol", "reward", string(resultByte))
}

// 计算完整奖励
// 假定每10000块计算一次奖励，则这里会计算例如高度为0-9999，10000-19999的结果
func (reward *RewardCalculator) calculateReward(height uint64, situation string) map[common.Address]*big.Int {
	result := make(map[common.Address]*big.Int, 0)
	from := height - common.RewardBlocks
	reward.logger.Debugf("start to calculate, from %d to %d", from, height-1)
	defer reward.logger.Debugf("end to calculate, from %d to %d", from, height-1)

	for i := from; i < height; i++ {
		if i == 0 {
			continue
		}

		var bh *types.BlockHeader
		if situation != "fork" {
			bh = reward.blockChain.QueryBlockHeaderByHeight(i, true)
		} else {
			bh = reward.fork.GetBlockHeader(i)
		}
		if nil == bh {
			reward.logger.Errorf("fail to get blockHeader. height: %d", i)
			continue
		}

		piece := reward.calculateRewardPerBlock(bh, situation)
		if nil == piece {
			continue
		}

		for addr, value := range piece {
			addReward(result, addr, value)
		}
	}

	return result
}

// 计算某一块的奖励
func (reward *RewardCalculator) calculateRewardPerBlock(bh *types.BlockHeader, situation string) map[common.Address]*big.Int {
	result := make(map[common.Address]*big.Int)

	height := bh.Height
	total := getTotalReward(height)
	hashString := bh.Hash.String()
	reward.logger.Debugf("start to calculate, height: %d, hash: %s, proposer: %s, groupId: %s, totalReward %f", height, hashString, common.ToHex(bh.Castor), common.ToHex(bh.GroupId), total)
	defer reward.logger.Warnf("end to calculate, height %d, hash: %s, result: %v", height, hashString, result)

	accountDB, err := AccountDBManagerInstance.GetAccountDBByHash(bh.StateTree)
	if err != nil {
		reward.logger.Errorf("get account db by height: %d error:%s", height, err.Error())
		return nil
	}

	// 社区奖励
	communityReward := utility.Float64ToBigInt(total * common.CommunityReward)
	result[common.CommunityAddress] = communityReward
	reward.logger.Debugf("calculating, height: %d, hash: %s, CommunityAddress: %s, reward: %d", height, hashString, common.CommunityAddress.String(), communityReward)

	// 提案者奖励
	rewardAllProposer := total * common.AllProposerReward
	rewardProposer := utility.Float64ToBigInt(rewardAllProposer * common.ProposerReward)

	proposerAddr := getAddressFromID(bh.Castor)
	addReward(result, proposerAddr, rewardProposer)
	proposerResult := result[proposerAddr].String()
	reward.logger.Debugf("calculating, height: %d, hash: %s, proposerAddr: %s, reward: %d, result: %s", height, hashString, proposerAddr.String(), rewardProposer, proposerResult)

	// 其他提案者奖励
	totalProposerStake, proposersStake := reward.minerManager.GetProposerTotalStakeWithDetail(height, accountDB)
	if totalProposerStake != 0 {
		otherRewardProposer := rewardAllProposer * (1 - common.ProposerReward)
		for addr, stake := range proposersStake {
			// todo: 本块的提案者要拿钱么
			//if addr == proposerAddr {
			//	continue
			//}
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
		group = reward.fork.GetGroupById(bh.GroupId)
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

func (reward *RewardCalculator) needReward(height uint64) bool {
	return 0 == (height % common.RewardBlocks)
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
