package core

import (
	"x/src/common"
	"math"
	"math/big"
	"x/src/utility"
	"x/src/middleware/types"
	"x/src/middleware/log"
)

type RewardCalculator struct {
	minerManager *MinerManager
	blockChain   *blockChain
	groupChain   GroupChain
	logger       log.Logger
}

var RewardCalculatorImpl *RewardCalculator

func initRewardCalculator(minerManager *MinerManager, blockChainImpl *blockChain, groupChain GroupChain) {
	RewardCalculatorImpl = &RewardCalculator{minerManager: minerManager, blockChain: blockChainImpl, groupChain: groupChain}
	RewardCalculatorImpl.logger = log.GetLoggerByIndex(log.RewardLogConfig, common.GlobalConf.GetString("instance", "index", ""))
}

// 计算完整奖励
// 假定每10000块计算一次奖励，则这里会计算例如高度为0-9999，10000-19999的结果
func (reward *RewardCalculator) CalculateReward(height uint64) map[common.Address]*big.Int {
	result := make(map[common.Address]*big.Int, 0)
	from := height - common.RewardBlocks
	reward.logger.Debugf("start to calculate, from %d to %d", from, height)

	for i := from; i < height; i++ {
		bh := reward.blockChain.queryBlockHeaderByHeight(i, true)
		piece := reward.calculateReward(bh)
		if nil == piece {
			continue
		}

		for addr, value := range piece {
			old, ok := result[addr]
			if ok {
				old.Add(old, value)
				result[addr] = old
			} else {
				result[addr] = value
			}
		}
	}

	reward.logger.Debugf("end to calculate, from %d to %d", from, height)
	return result
}

// 计算某一块的奖励
func (reward *RewardCalculator) calculateReward(bh *types.BlockHeader) (result map[common.Address]*big.Int) {
	if nil == bh {
		return nil
	}

	result = make(map[common.Address]*big.Int)
	height := bh.Height
	total := getTotalReward(height)
	hashString := bh.Hash.String()
	reward.logger.Debugf("start to calculate, height: %d, hash: %s, totalReward %d", height, hashString, total)
	defer reward.logger.Warnf("end to calculate, height %d, hash: %s, result: %v", height, hashString, result)

	// 社区奖励
	communityReward := utility.Float64ToBigInt(total * common.CommunityReward)
	result[common.CommunityAddress] = communityReward
	reward.logger.Debugf("calculating, height: %d, hash: %s, CommunityAddress %s, reward %d", height, hashString, common.CommunityAddress.String(), communityReward)

	// 提案者奖励
	rewardProposer := total * common.AllProposerReward
	proposerAddr := getAddressFromID(bh.Castor)
	result[proposerAddr] = utility.Float64ToBigInt(rewardProposer * common.ProposerReward)
	reward.logger.Debugf("calculating, height: %d, hash: %s, proposerAddr %s, reward %d", height, hashString, proposerAddr.String(), result[proposerAddr])

	// 其他提案者奖励
	totalProposerStake, proposersStake := reward.minerManager.GetProposerTotalStake(height)
	if totalProposerStake != 0 {
		otherRewardProposer := rewardProposer * (1 - common.ProposerReward)
		for addr, stake := range proposersStake {
			// todo: 本块的提案者要拿钱么
			//if addr == proposerAddr {
			//	continue
			//}
			result[addr] = utility.Float64ToBigInt(float64(stake) / float64(totalProposerStake) * otherRewardProposer)
			reward.logger.Debugf("calculating, height: %d, hash: %s, proposerAddr %s, reward %d", height, hashString, addr.String(), result[addr])
		}
	}

	// 验证者奖励
	group := reward.groupChain.GetGroupById(bh.GroupId)
	if group == nil {
		return
	}

	totalValidatorStake, validatorStake := reward.minerManager.GetValidatorsStake(height, group.Members)
	if totalValidatorStake != 0 {
		rewardValidators := total * common.ValidatorsReward
		for addr, stake := range validatorStake {
			result[addr] = utility.Float64ToBigInt(float64(stake) / float64(totalValidatorStake) * rewardValidators)
			reward.logger.Debugf("calculating, height: %d, hash: %s, validatorAddr %s, reward %d", height, hashString, addr.String(), result[addr])
		}
	}

	return result
}

func (reward *RewardCalculator) NeedReward(height uint64) bool {
	return 0 == (height % common.RewardBlocks)
}

func getYear(height uint64) uint64 {
	return uint64(height / common.BlocksPerYear)
}

func getTotalReward(height uint64) float64 {
	return common.FirstYearRewardPerBlock * math.Pow(common.Inflation, float64(getYear(height)))
}

func getAddressFromID(id []byte) common.Address {
	return common.BytesToAddress(id)
}
