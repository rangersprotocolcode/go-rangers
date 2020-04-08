package utility

import (
	"sync"
	"time"
)

var (
	ntpServers  = []string{"time.nist.gov", "time.windows.com"}
	timeOffset  time.Duration
	ntpInit     sync.Once
	ntpInitFlag = false
	cstZone     = time.FixedZone("CST", 8*3600)
)

func GetTime() time.Time {
	if !ntpInitFlag {
		timeOffset = ntpOffset(true)
		ntpInitFlag = true
		ntpInit.Do(func() {
			timer := time.NewTimer(time.Second * 30)
			go func() {
				<-timer.C
				timeOffset = ntpOffset(false)
			}()
		})
	}

	return time.Now().Add(timeOffset).In(cstZone)
}

func FormatTime(t time.Time) time.Time {
	return t.In(cstZone)
}
