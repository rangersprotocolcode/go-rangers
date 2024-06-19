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

package common

const (
	CastingCheckInterval = 50

	castingInterval = 1000

	rewardTime = 10 * 60 * 60 * 1000

	RefundTime = 50 * 1000

	oneDay = 24 * 3600 * 1000

	epoch = 180 * oneDay
)

const (
	TotalRPGSupply = 2100 * 10000 * 0.35

	ReleaseRate = 0.08

	ValidatorsReward = float64(2) / 7

	AllProposerReward = float64(1) / 2

	ProposerReward = float64(3) / 14
)

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

func GetRewardBlocks() uint64 {
	return rewardTime / GetCastingInterval()
}

func GetRefundBlocks() uint64 {
	return RefundTime / GetCastingInterval()
}

func GetBlocksPerEpoch() uint64 {
	return epoch / GetCastingInterval()
}
