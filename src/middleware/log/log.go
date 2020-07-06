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

package log

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"

	"github.com/cihub/seelog"
)

var logManager = map[string]Logger{}

var lock sync.Mutex

func GetLogger(config string) Logger {
	if config == `` {
		config = DefaultConfig
	}
	key := getKey(config)
	lock.Lock()
	r := logManager[key]
	lock.Unlock()

	if r == nil {
		l := newLoggerByConfig(config)
		register(getKey(config), l)
		return l
	}
	return r
}

func GetLoggerByIndex(config string, index string) Logger {
	if index == "0" {
		index = ""
	}
	key := getKey(config)
	lock.Lock()
	r := logManager[key]
	lock.Unlock()

	if r == nil {
		if config == "" {
			config = DefaultConfig
		}
		config = strings.Replace(config, "LOG_INDEX", index, 1)
		l := newLoggerByConfig(config)
		register(getKey(config), l)
		return l
	}
	return r
}
func GetLoggerByName(name string) Logger {
	key := getKey(name)
	lock.Lock()
	r := logManager[key]
	lock.Unlock()

	if r != nil {
		return r
	} else {
		var config string
		if name == "" {
			config = DefaultConfig
			return GetLogger(config)
		} else {
			fileName := name + ".log"
			config = strings.Replace(DefaultConfig, "defaultLOG_INDEX.log", fileName, 1)
			l := newLoggerByConfig(config)
			register(getKey(name), l)
			return l
		}
	}
}

func getKey(s string) string {
	hash := sha256.Sum256([]byte(s))
	return string(hash[:])
}

func newLoggerByConfig(config string) Logger {
	l, err := seelog.LoggerFromConfigAsBytes([]byte(config))
	if err != nil {
		fmt.Printf("Get logger error:%s\n", err.Error())
		panic(err)
	}
	return l
}

func register(name string, logger Logger) {
	lock.Lock()
	defer lock.Unlock()
	if logger != nil {
		logManager[name] = logger
	}
}

func Close() {
	lock.Lock()
	defer lock.Unlock()
	for _, logger := range logManager {
		logger.(seelog.LoggerInterface).Flush()
		logger.(seelog.LoggerInterface).Close()
	}
}
