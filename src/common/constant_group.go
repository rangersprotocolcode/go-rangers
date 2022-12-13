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

const (
	EPOCH            int = 5
	Group_Create_Gap     = 50
	GROUP_Work_GAP       = Group_Create_Gap + EPOCH*8 //组准备就绪后, 等待可以铸块的间隔为4个epoch
)

var GROUP_Work_DURATION = 2 * 60 * 60 * 1000 / GetCastingInterval() //组铸块的周期为100个epoch
