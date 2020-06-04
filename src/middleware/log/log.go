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
