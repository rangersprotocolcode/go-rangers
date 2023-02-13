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

// log before/after lock

package middleware

import (
	"com.tuntun.rocket/node/src/utility"
	"fmt"
	"strconv"
	"sync"
	"time"

	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
)

type Loglock struct {
	lock  sync.RWMutex
	addr  string
	begin time.Time
}

var (
	lockLogger log.Logger
	lock       Loglock

	accountDBLock Loglock
)

const costLimit = 10 * time.Microsecond
const durationLimit = time.Millisecond

func NewLoglock(title string) Loglock {
	loglock := Loglock{
		lock: sync.RWMutex{},
	}
	loglock.addr = fmt.Sprintf("%p", &loglock)
	if lockLogger == nil {
		lockLogger = log.GetLoggerByIndex(log.LockLogConfig, strconv.Itoa(common.InstanceIndex))
	}
	return loglock
}

func (lock *Loglock) Lock(msg string) {
	if 0 != len(msg) {
		lockLogger.Debugf("try to lock: %s, with msg: %s", lock.addr, msg)
	}

	begin := utility.GetTime()
	lock.lock.Lock()
	lock.begin = begin
	cost := utility.GetTime().Sub(begin)

	lockLogger.Debugf("locked: %s, with msg: %s, waited: %v", lock.addr, msg, cost)
}

func (lock *Loglock) RLock(msg string) {
	if 0 != len(msg) {
		lockLogger.Debugf("try to Rlock: %s, with msg: %s", lock.addr, msg)
	}

	begin := utility.GetTime()
	lock.lock.RLock()
	lock.begin = begin
	cost := utility.GetTime().Sub(begin)

	lockLogger.Debugf("Rlocked: %s, with msg: %s, waited: %v", lock.addr, msg, cost)
}

func (lock *Loglock) Unlock(msg string) {
	if 0 != len(msg) {
		lockLogger.Debugf("try to UnLock: %s, with msg: %s", lock.addr, msg)
	}

	lock.lock.Unlock()
	duration := utility.GetTime().Sub(lock.begin)

	lockLogger.Debugf("UnLocked: %s, with msg: %s, duration:%v", lock.addr, msg, duration)
}

func (lock *Loglock) RUnlock(msg string) {
	if 0 != len(msg) {
		lockLogger.Debugf("try to UnRLock: %s, with msg: %s", lock.addr, msg)
	}

	lock.lock.RUnlock()
	duration := utility.GetTime().Sub(lock.begin)

	lockLogger.Debugf("UnRLocked: %s, with msg: %s, duration:%v", lock.addr, msg, duration)
}

func LockBlockchain(msg string) {
	lock.Lock(msg)
}

func UnLockBlockchain(msg string) {
	lock.Unlock(msg)
}

func RLockBlockchain(msg string) {
	lock.RLock(msg)
}

func RUnLockBlockchain(msg string) {
	lock.RUnlock(msg)
}

func LockAccountDB(msg string) {
	accountDBLock.Lock(msg)
}

func UnLockAccountDB(msg string) {
	accountDBLock.Unlock(msg)
}

func RLockAccountDB(msg string) {
	accountDBLock.RLock(msg)
}

func RUnLockAccountDB(msg string) {
	accountDBLock.RUnlock(msg)
}
