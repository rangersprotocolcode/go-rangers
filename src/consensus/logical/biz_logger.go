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

package logical

import (
	"com.tuntun.rangers/node/src/utility"
	"fmt"
	"time"

	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus/groupsig"
)

type bLog interface {
	log(format string, params ...interface{})
}

type bizLog struct {
	biz string
}

func newBizLog(biz string) *bizLog {
	return &bizLog{biz: biz}
}

func (bl *bizLog) log(format string, p ...interface{}) {
	stdLogger.Infof("%v:%v", bl.biz, fmt.Sprintf(format, p...))
}

func (bl *bizLog) debug(format string, p ...interface{}) {
	stdLogger.Debugf("%v:%v", bl.biz, fmt.Sprintf(format, p...))
}

type rtLog struct {
	start time.Time
	key   string
}

func newRtLog(key string) *rtLog {
	return &rtLog{
		start: utility.GetTime(),
		key:   key,
	}
}

const TIMESTAMP_LAYOUT = "2006-01-02/15:04:05.000"

func (r *rtLog) log(format string, p ...interface{}) {
	cost := utility.GetTime().Sub(r.start)
	stdLogger.Debugf(fmt.Sprintf("%v:begin at %v, cost %v. %v", r.key, r.start.Format(TIMESTAMP_LAYOUT), cost.String(), fmt.Sprintf(format, p...)))
}

func (r *rtLog) end() {
	stdLogger.Debugf(fmt.Sprintf("%v:%v cost ", utility.GetTime().Format(TIMESTAMP_LAYOUT), r.key))
}

type msgTraceLog struct {
	mtype  string
	key    string
	sender string
}

func newMsgTraceLog(mtype string, key string, sender string) *msgTraceLog {
	return &msgTraceLog{
		mtype:  mtype,
		key:    key,
		sender: sender,
	}
}

func newHashTraceLog(mtype string, hash common.Hash, sid groupsig.ID) *msgTraceLog {
	return newMsgTraceLog(mtype, hash.ShortS(), sid.ShortS())
}

func _doLog(t string, k string, sender string, format string, params ...interface{}) {
	var s string
	if params == nil || len(params) == 0 {
		s = format
	} else {
		s = fmt.Sprintf(format, params...)
	}
	consensusLogger.Infof("%v,#%v#,%v,%v", t, k, sender, s)
}

func (mtl *msgTraceLog) log(format string, params ...interface{}) {
	_doLog(mtl.mtype, mtl.key, mtl.sender, format, params...)
}

func (mtl *msgTraceLog) logStart(format string, params ...interface{}) {
	_doLog(mtl.mtype+"-begin", mtl.key, mtl.sender, format, params...)
}

func (mtl *msgTraceLog) logEnd(format string, params ...interface{}) {
	_doLog(mtl.mtype+"-end", mtl.key, mtl.sender, format, params...)
}

type stageLogTime struct {
	stage string
	begin time.Time
	end   time.Time
}

type slowLog struct {
	lts       []*stageLogTime
	begin     time.Time
	key       string
	threshold float64
}

func newSlowLog(key string, thresholdSecs float64) *slowLog {
	return &slowLog{
		lts:       make([]*stageLogTime, 0),
		begin:     utility.GetTime(),
		key:       key,
		threshold: thresholdSecs,
	}
}

func (log *slowLog) addStage(key string) {
	st := &stageLogTime{
		begin: utility.GetTime(),
		stage: key,
	}
	log.lts = append(log.lts, st)
}

func (log *slowLog) endStage() {
	if len(log.lts) > 0 {
		st := log.lts[len(log.lts)-1]
		st.end = utility.GetTime()
	}
}

func (log *slowLog) log(format string, params ...interface{}) {
	c := utility.GetTime().Sub(log.begin)
	if c.Seconds() < log.threshold {
		return
	}

	s := fmt.Sprintf(format, params...)
	detail := ""
	for _, lt := range log.lts {
		if lt.end.Nanosecond() == 0 {
			continue
		}
		detail = fmt.Sprintf("%v,%v(%v)", detail, lt.stage, lt.end.Sub(lt.begin).String())
	}
	s = fmt.Sprintf("%v:%v,cost %v, detail %v", log.key, s, c.String(), detail)
	slowLogger.Warnf(s)
}
