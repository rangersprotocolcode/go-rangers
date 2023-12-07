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
	EPOCH            int = 5
	Group_Create_Gap     = 50
	GROUP_Work_GAP       = Group_Create_Gap + EPOCH*8 //组准备就绪后, 等待可以铸块的间隔为4个epoch
)

var groupWorkDuration = 2 * 60 * 60 * 1000 / GetCastingInterval() //组铸块的周期为100个epoch

func GetGroupWorkDuration() uint64 {
	if IsSub() && 0 != Genesis.GroupLife {
		return Genesis.GroupLife / GetCastingInterval()
	}

	return groupWorkDuration
}
