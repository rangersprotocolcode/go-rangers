package service

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"math/big"
)

func (reward *RewardCalculator) CalculateRewardForFork(height uint64, db *account.AccountDB) bool {
	if !reward.needReward(height) {
		reward.logger.Warnf("no need to reward, height: %d", height)
		return false
	}

	total := reward.calculateRewardForFork(height)
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

// 计算完整奖励
// 假定每10000块计算一次奖励，则这里会计算例如高度为0-9999，10000-19999的结果
func (reward *RewardCalculator) calculateRewardForFork(height uint64) map[common.Address]*big.Int {
	result := make(map[common.Address]*big.Int, 0)
	from := height - common.RewardBlocks
	reward.logger.Debugf("start to calculate, from %d to %d", from, height-1)
	defer reward.logger.Debugf("end to calculate, from %d to %d", from, height-1)

	for i := from; i < height; i++ {
		if i == 0 {
			continue
		}

		var bh *types.BlockHeader
		bh = reward.blockChain.QueryBlockHeaderByHeight(i, true)
		if nil == bh {
			//todo
			reward.logger.Errorf("fail to get blockHeader. height: %d", i)
			continue
		}

		piece := reward.calculateRewardPerBlockForFork(bh)
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
func (reward *RewardCalculator) calculateRewardPerBlockForFork(bh *types.BlockHeader) map[common.Address]*big.Int {
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
	//todo
	group := reward.groupChain.GetGroupById(bh.GroupId)
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
