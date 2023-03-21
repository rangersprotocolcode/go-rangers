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

import (
	"com.tuntun.rocket/node/src/middleware/log"
	"strconv"
)

const (
	instanceSection = "instance"
	indexKey        = "index"

	ConfigSec                     = "chain"
	DefaultDatabase               = "database"
	DefaultJoinedGroupDatabaseKey = "joinedGroupDatabase"
)

func Init(instanceIndex int, configFile, env string) {
	initConf(configFile)

	if 0 != instanceIndex {
		InstanceIndex = instanceIndex
		GlobalConf.SetInt(instanceSection, indexKey, instanceIndex)
	} else {
		InstanceIndex = GlobalConf.GetInt(instanceSection, indexKey, 0)
	}

	databaseValue := "chain"
	GlobalConf.SetString(ConfigSec, DefaultDatabase, databaseValue)
	joinedGroupDatabaseValue := "jgs"
	GlobalConf.SetString(ConfigSec, DefaultJoinedGroupDatabaseKey, joinedGroupDatabaseValue)

	DefaultLogger = log.GetLoggerByIndex(log.DefaultConfig, strconv.Itoa(InstanceIndex))

	initChainConfig(env)
}
