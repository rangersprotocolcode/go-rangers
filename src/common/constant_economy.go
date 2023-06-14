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
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package common

// 时间
const (
	// 检查出块间隔，单位ms
	CastingCheckInterval = 50

	// 出块间隔，单位ms
	castingInterval = 1000

	// 10个小时，单位ms
	// 计算一次奖励的时间间隔
	rewardTime = 10 * 60 * 60 * 1000
	//10 * 60 * 60 * 1000

	RefundTime = 50 * 1000

	// 一天，单位ms
	oneDay = 24 * 3600 * 1000

	// 释放周期
	epoch = 180 * oneDay
)

var (

	// 按照出块速度，计算奖励所需要的块数目
	RewardBlocks = rewardTime / GetCastingInterval()

	RefundBlocks = RefundTime / GetCastingInterval()

	// 一个epoch内，出块总量
	BlocksPerEpoch = epoch / GetCastingInterval()
)

// 奖励
const (
	// 矿工总奖励
	TotalRPGSupply = 2100 * 10000 * 0.35

	ReleaseRate = 0.08

	// 验证组比例
	ValidatorsReward = float64(2) / 7

	// 所有提案者比例
	AllProposerReward = float64(1) / 2

	// 出块的提案者比例
	ProposerReward = float64(3) / 14
)

// 最小质押量
const (
	ValidatorStake   = uint64(400)
	ProposerStake    = uint64(2000)
	MAXGROUP         = 50
	HeightAfterStake = 300
)

const (
	BLANCE_NAME = "SYSTEM-RPG"
)

var FeeAccount = HexToAddress("0x3966eafd38c5f10cc91eaacaeff1b6682b83ced4")

func GetCastingInterval() uint64 {
	if IsSub() {
		return Genesis.Cast
	}

	return castingInterval
}

var EconomyContract = HexToAddress("0x71d9cfd1b7adb1e8eb4c193ce6ffbe19b4aee0db")
