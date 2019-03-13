// log before/after lock

package middleware

import (
	"sync"
	"fmt"
	"x/src/middleware/log"
	"time"
	"x/src/common"
)

type Loglock struct {
	lock  sync.RWMutex
	addr  string
	begin time.Time
}

var lockLogger log.Logger

const costLimit = 10 * time.Microsecond
const durationLimit = time.Millisecond

func NewLoglock(title string) Loglock {
	loglock := Loglock{
		lock: sync.RWMutex{},
	}
	loglock.addr = fmt.Sprintf("%p", &loglock)
	if lockLogger ==nil{
		lockLogger = log.GetLoggerByIndex(log.LockLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	}
	return loglock
}

func (lock *Loglock) Lock(msg string) {
	if 0 != len(msg) {
		lockLogger.Debugf("try to lock: %s, with msg: %s", lock.addr, msg)
	}
	begin := time.Now()
	lock.lock.Lock()
	lock.begin = time.Now()
	cost := time.Since(begin)

	if 0 != len(msg) && cost > costLimit {
		lockLogger.Debugf("locked: %s, with msg: %s wait: %v", lock.addr, msg, cost)
	}

}

func (lock *Loglock) RLock(msg string) {
	if 0 != len(msg) {
		lockLogger.Debugf("try to Rlock: %s, with msg: %s", lock.addr, msg)
	}
	begin := time.Now()
	lock.lock.RLock()
	cost := time.Since(begin)
	if 0 != len(msg) && cost > costLimit {
		lockLogger.Debugf("Rlocked: %s, with msg: %s wait: %v", lock.addr, msg, cost)
	}
}

func (lock *Loglock) Unlock(msg string) {
	if 0 != len(msg) {
		lockLogger.Debugf("try to UnLock: %s, with msg: %s", lock.addr, msg)
	}
	begin := time.Now()
	lock.lock.Unlock()
	duration := time.Since(lock.begin)
	cost := time.Since(begin)
	if 0 != len(msg) && (cost > costLimit || duration > durationLimit) {
		lockLogger.Debugf("UnLocked: %s, with msg: %s duration:%v wait: %v", lock.addr, msg, duration, cost)
	}

}

func (lock *Loglock) RUnlock(msg string) {
	if 0 != len(msg) {
		lockLogger.Debugf("try to UnRLock: %s, with msg: %s", lock.addr, msg)
	}
	begin := time.Now()
	lock.lock.RUnlock()
	cost := time.Since(begin)
	if 0 != len(msg) && cost > costLimit {
		lockLogger.Debugf("UnRLocked: %s, with msg: %s wait: %v", lock.addr, msg, cost)
	}

}
