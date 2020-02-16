package core

import (
	"x/src/common"
	"math"
	"math/big"
	"x/src/utility"
	"x/src/middleware/types"
	"x/src/middleware/log"
	"x/src/storage/account"
	"x/src/service"
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

func (reward *RewardCalculator) CalculateReward(height uint64, db *account.AccountDB) bool {
	if !reward.needReward(height) {
		reward.logger.Warnf("no need to reward, height: %d", height)
		return false
	}

	total := reward.calculateReward(height)
	if nil == total || 0 == len(total) {
		reward.logger.Errorf("fail to reward, height: %d", height)
		return false
	}

	for addr, money := range total {
		from := db.GetBalance(addr).String()
		db.AddBalance(addr, money)
		reward.logger.Debugf("add reward, addr: %s, from: %s to %v", addr.String(), from, db.GetBalance(addr))
	}
	return true
}

// 计算完整奖励
// 假定每10000块计算一次奖励，则这里会计算例如高度为0-9999，10000-19999的结果
func (reward *RewardCalculator) calculateReward(height uint64) map[common.Address]*big.Int {
	result := make(map[common.Address]*big.Int, 0)
	from := height - common.RewardBlocks
	reward.logger.Debugf("start to calculate, from %d to %d", from, height-1)
	defer reward.logger.Debugf("end to calculate, from %d to %d", from, height-1)

	for i := from; i < height; i++ {
		if i == 0 {
			continue
		}

		bh := reward.blockChain.queryBlockHeaderByHeight(i, true)
		piece := reward.calculateRewardPerBlock(bh)
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
func (reward *RewardCalculator) calculateRewardPerBlock(bh *types.BlockHeader) (result map[common.Address]*big.Int) {
	if nil == bh {
		return nil
	}

	result = make(map[common.Address]*big.Int)
	height := bh.Height
	total := getTotalReward(height)
	hashString := bh.Hash.String()
	reward.logger.Debugf("start to calculate, bh: %s, totalReward %f", bh.ToString(), total)
	defer reward.logger.Warnf("end to calculate, height %d, hash: %s, result: %v", height, hashString, result)

	accountDB, err := service.AccountDBManagerInstance.GetAccountDBByHash(bh.StateTree)
	if err != nil {
		reward.logger.Errorf("get account db by height: %d error:%s", height, err.Error())
		return
	}

	// 社区奖励
	communityReward := utility.Float64ToBigInt(total * common.CommunityReward)
	result[common.CommunityAddress] = communityReward
	reward.logger.Debugf("calculating, height: %d, hash: %s, CommunityAddress: %s, reward: %d", height, hashString, common.CommunityAddress.String(), communityReward)

	// 提案者奖励
	rewardAllProposer := 15.9 * common.AllProposerReward
	rewardProposer := utility.Float64ToBigInt(rewardAllProposer * common.ProposerReward)

	proposerAddr := getAddressFromID(bh.Castor)
	result[proposerAddr] = rewardProposer
	reward.logger.Debugf("calculating, height: %d, hash: %s, proposerAddr: %s, reward: %d", height, hashString, proposerAddr.String(), rewardProposer)

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
			reward.logger.Debugf("calculating, height: %d, hash: %s, proposerAddr: %s, reward: %d", height, hashString, addr.String(), delta)
		}
	}

	// 验证者奖励
	group := reward.groupChain.GetGroupById(bh.GroupId)
	if group == nil {
		reward.logger.Errorf("fail to get group. id: %v", bh.GroupId)
		return
	}

	totalValidatorStake, validatorStake := reward.minerManager.GetValidatorsStake(height, group.Members, accountDB)
	if totalValidatorStake != 0 {
		rewardValidators := total * common.ValidatorsReward
		for addr, stake := range validatorStake {
			result[addr] = utility.Float64ToBigInt(float64(stake) / float64(totalValidatorStake) * rewardValidators)
			reward.logger.Debugf("calculating, height: %d, hash: %s, validatorAddr %s, reward %d", height, hashString, addr.String(), result[addr])
		}
	}

	return result
}

func (reward *RewardCalculator) needReward(height uint64) bool {
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
