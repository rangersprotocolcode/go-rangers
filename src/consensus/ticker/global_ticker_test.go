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
	"log"
	"testing"
	"time"
)

func handler(str string) RoutineFunc {
	return func() bool {
		log.Printf(str)
		return true
	}
}

func TestGlobalTicker_RegisterRoutine(t *testing.T) {

	ticker := newGlobalTicker("test")

	time.Sleep(time.Second * 5)

	ticker.RegisterRoutine("name1", handler("name1 exec1"), uint32(2))

	time.Sleep(time.Second * 5)
	ticker.RegisterRoutine("name2", handler("name2 exec1"), uint32(3))
	time.Sleep(time.Second * 5)

	ticker.RegisterRoutine("name3", handler("name3 exec1"), uint32(4))

	ticker.StopTickerRoutine("name1")

	time.Sleep(time.Second * 5)
	ticker.StopTickerRoutine("name3")
	time.Sleep(time.Second * 55)
}
