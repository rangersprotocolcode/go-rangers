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
			ticker := time.NewTicker(time.Second * 10)
			go func() {
				for _ = range ticker.C {
					offsetResult := ntpOffset(false)
					if offsetResult != 0 {
						timeOffset = offsetResult
						fmt.Printf("refresh ntp, timeOffset: %s\n", timeOffset)
					} else {
						fmt.Printf("refresh ntp failed, use last timeOffset: %s\n", timeOffset)
					}

				}

			}()
		})
	}

	return time.Now().Add(timeOffset).In(cstZone)
}

func FormatTime(t time.Time) time.Time {
	return t.In(cstZone)
}
