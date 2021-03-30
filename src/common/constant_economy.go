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

package common

// 时间
const (
	// 检查出块间隔，单位ms
	CastingCheckInterval = 50

	// 出块间隔，单位ms
	CastingInterval = 1000

	// 10个小时，单位ms
	// 计算一次奖励的时间间隔
	RewardTime = 1 * 1000

	RefundTime = 50 * 1000

	// 按照出块速度，计算奖励所需要的块数目
	RewardBlocks = uint64(RewardTime / CastingInterval)

	RefundBlocks = uint64(RefundTime / CastingInterval)

	// 一年，单位ms
	//OneYear = 365 * 24 * 3600 * 1000
	//oneYear = 7 * oneDay

	oneDay = 24 * 3600 * 1000
	epoch  = 180 * oneDay

	// 一个epoch出得块数量
	BlocksPerEpoch = epoch / CastingInterval

	// 一天出得块数量
	BlocksPerDay = uint64(oneDay / CastingInterval)
)

// 奖励
const (
	// 矿工总奖励
	TotalRPGSupply = 2100 * 10000 * 0.49

	ReleaseRate = 0.08

	// 社区比例
	CommunityReward = 0.3

	// 验证组比例
	ValidatorsReward = 0.2

	// 所有提案者比例
	AllProposerReward = 0.5

	// 出块的提案者比例
	ProposerReward = 0.3
)

// 最小质押量
const (
	ValidatorStake = uint64(200)
	ProposerStake  = uint64(10000)

	HeightAfterStake = RewardBlocks
)

var (
	CommunityAddress = HexToAddress("0xffffff")
)
