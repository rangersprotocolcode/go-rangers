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

package logical

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/model"
	"com.tuntun.rocket/node/src/middleware/log"
	"strconv"
)

const ConsensusConfSection = "consensus"

var (
	consensusLogger log.Logger
	stdLogger       log.Logger
	groupLogger     log.Logger
	slowLogger      log.Logger

	consensusConfManager common.SectionConfManager
)

func InitConsensus() {
	consensusLogger = log.GetLoggerByIndex(log.ConsensusLogConfig, strconv.Itoa(common.InstanceIndex))
	stdLogger = log.GetLoggerByIndex(log.StdConsensusLogConfig, strconv.Itoa(common.InstanceIndex))
	groupLogger = log.GetLoggerByIndex(log.GroupLogConfig, strconv.Itoa(common.InstanceIndex))
	slowLogger = log.GetLoggerByIndex(log.SlowLogConfig, strconv.Itoa(common.InstanceIndex))

	consensusConfManager = common.GlobalConf.GetSectionManager(ConsensusConfSection)
	model.InitParam(consensusConfManager)
}
