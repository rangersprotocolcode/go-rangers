package utility

import (
	"fmt"
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
			ticker := time.NewTicker(time.Second * 30)
			go func() {
				for _ = range ticker.C {
					timeOffset = ntpOffset(false)
					fmt.Printf("refresh ntp, timeOffset: %s\n", timeOffset)
				}

			}()
		})
	}

	return time.Now().Add(timeOffset).In(cstZone)
}

func FormatTime(t time.Time) time.Time {
	return t.In(cstZone)
}
