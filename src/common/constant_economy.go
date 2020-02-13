package common

// 时间
const (
	// 出块间隔，单位ms
	CastingInterval = 1000

	// 10个小时计算一次奖励
	RewardTime = 10 * 3600 * 1000

	// 按照出块速度，计算奖励所需要的块数目
	RewardBlocks = int(RewardTime / CastingInterval)

	// 一年
	OneYear = 365 * 24 * 3600 * 1000

	// 一年出得块数量
	BlocksPerYear = int(OneYear / CastingInterval)
)

// 奖励
const (
	// 第一年的奖励
	FirstYearRewardPerBlock = 15.9

	// 通胀率
	Inflation = 0.05

	// 社区比例
	CommunityReward = 0.2

	// 验证组比例
	ValidatorsReward = 0.3

	// 提案者比例
	ProposerReward = 0.5
)

// 最小质押量
const (
	ValidatorStake = 100000
	ProposerStake  = 1000000
)
