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
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package ticker

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/utility"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

const (
	STOPPED = int32(0)
	RUNNING = int32(1)
)

var ticker = newGlobalTicker("global")

type RoutineFunc func() bool

type TickerRoutine struct {
	id              string
	handler         RoutineFunc
	interval        uint32
	lastTicker      uint64
	triggerCh       chan int32
	status          int32
	triggerNextTick int32
}

type GlobalTicker struct {
	beginTime time.Time
	timer     *time.Ticker
	ticker    uint64
	id        string
	routines  sync.Map //string -> *TickerRoutine
}

func GetTickerInstance() *GlobalTicker {
	return ticker
}

func (gt *GlobalTicker) RegisterRoutine(name string, routine RoutineFunc, interval uint32) {
	//log.Printf("RegisterRoutine, id=%v, interval=%v\n", name, interval)
	if rt := gt.getRoutine(name); rt != nil {
		//log.Printf("RegisterRoutine, id=%v already exist!\n", name)
		return
	}
	r := &TickerRoutine{
		interval:        interval,
		handler:         routine,
		lastTicker:      0,
		id:              name,
		triggerCh:       make(chan int32, 5),
		status:          STOPPED,
		triggerNextTick: 0,
	}
	go func() {
	STOP:
		for {
			select {
			case val := <-r.triggerCh:
				if val == -1 {
					//log.Println("ticker routine stopped!, name=", r.id)
					break STOP
				} else {
					gt.trigger(r, val)
				}
			}
		}
	}()

	gt.addRoutine(name, r)
	gt.routines.Range(func(key, value interface{}) bool {
		//log.Println("ticker name ", key)
		return true
	})
}

func (gt *GlobalTicker) RemoveRoutine(name string) {
	routine := gt.getRoutine(name)
	if routine == nil {
		return
	}
	routine.triggerCh <- -1
	//log.Println("routine removed!, name=", routine.id)
	gt.routines.Delete(name)
}

func (gt *GlobalTicker) StartTickerRoutine(name string, triggerNextTicker bool) {
	routine := gt.getRoutine(name)
	if routine == nil {
		return
	}
	if triggerNextTicker && atomic.CompareAndSwapInt32(&routine.triggerNextTick, 0, 1) {
		//log.Printf("routine will trigger next routine! id=%v\n", routine.id)
	}
	if atomic.CompareAndSwapInt32(&routine.status, STOPPED, RUNNING) {
		//log.Printf("routine started! id=%v\n", routine.id)
	} else {
		//log.Printf("routine routine start failed, already in running! id=%v\n", routine.id)
	}
}

func (gt *GlobalTicker) StartAndTriggerRoutine(name string) {
	routine := gt.getRoutine(name)
	if routine == nil {
		return
	}

	if atomic.CompareAndSwapInt32(&routine.status, STOPPED, RUNNING) {
		//log.Printf("StartAndTriggerRoutine: routine started! id=%v\n", routine.id)
	} else {
		//log.Printf("StartAndTriggerRoutine:routine routine start failed, already in running! id=%v\n", routine.id)
	}
	go func() {
		routine.triggerCh <- 2
	}()
}

func (gt *GlobalTicker) StopTickerRoutine(name string) {
	routine := gt.getRoutine(name)
	if routine == nil {
		return
	}

	if atomic.CompareAndSwapInt32(&routine.status, RUNNING, STOPPED) {
		//log.Printf("routine stopped! id=%v\n", routine.id)
	} else {
		//log.Printf("routine routine stop failed, not in running! id=%v\n", routine.id)
	}
}

func newGlobalTicker(id string) *GlobalTicker {
	ticker := &GlobalTicker{
		id:        id,
		beginTime: utility.GetTime(),
	}
	go ticker.routine()
	return ticker
}

func (gt *GlobalTicker) addRoutine(name string, tr *TickerRoutine) {
	gt.routines.Store(name, tr)
}

func (gt *GlobalTicker) getRoutine(name string) *TickerRoutine {
	if v, ok := gt.routines.Load(name); ok {
		return v.(*TickerRoutine)
	}
	return nil
}

func (gt *GlobalTicker) trigger(routine *TickerRoutine, chanVal int32) bool {
	defer func() {
		if r := recover(); r != nil {
			common.DefaultLogger.Errorf("error：%v\n", r)
			s := debug.Stack()
			common.DefaultLogger.Errorf(string(s))
		}
	}()

	t := gt.ticker
	lastTicker := atomic.LoadUint64(&routine.lastTicker)

	if atomic.LoadInt32(&routine.status) != RUNNING {
		//stdLo("ticker routine already stopped!, trigger return")
		return false
	}

	b := false
	if lastTicker < t && atomic.CompareAndSwapUint64(&routine.lastTicker, lastTicker, t) {
		//log.Printf("ticker routine begin, id=%v, globalticker=%v\n", routine.id, t)
		b = routine.handler()
	} else {
		if chanVal == 2 {
			atomic.CompareAndSwapInt32(&routine.triggerNextTick, 0, 1)
			//log.Printf("ticker routine executed this ticker, will trigger next ticker! id=%v, globalticker=%v, lastTicker=%v, status=%v\n", routine.id, t, routine.lastTicker, routine.status)
		} else {
			//log.Printf("ticker routine already executed this ticker! id=%v, globalticker=%v, lastTicker=%v, status=%v\n", routine.id, t, routine.lastTicker, routine.status)
		}
	}
	return b
}

func (gt *GlobalTicker) routine() {
	gt.timer = time.NewTicker(1 * time.Millisecond)
	for range gt.timer.C {
		gt.ticker++
		gt.routines.Range(func(key, value interface{}) bool {
			rt := value.(*TickerRoutine)
			if (atomic.LoadInt32(&rt.status) == RUNNING && gt.ticker-rt.lastTicker >= uint64(rt.interval)) || atomic.LoadInt32(&rt.triggerNextTick) == 1 {
				//rt.lastTicker = gt.ticker
				atomic.CompareAndSwapInt32(&rt.triggerNextTick, 1, 0)
				rt.triggerCh <- 1
			}
			return true
		})
	}
}
