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

package middleware

import (
	"com.tuntun.rangers/node/src/utility"
	"fmt"
	"strconv"
	"testing"
	"time"
)

func TestReentrantLock_Lock(t *testing.T) {
	lock := NewReentrantLock()

	fmt.Println(utility.GetTime().String())

	for i := 0; i < 10; i++ {
		go method(lock, strconv.Itoa(i))
	}

	time.Sleep(1000 * time.Second)
	fmt.Println(utility.GetTime().String())
}

func method(lock *ReentrantLock, name string) {
	for {
		lock.Lock(name)
		lock.Lock(name)
		fmt.Printf("%s %d locked\n", name, utility.GetGoroutineId())
		//lock.Release(name)
		lock.Unlock(name)
		lock.Unlock(name)
		fmt.Printf("%s %d unlocked\n", name, utility.GetGoroutineId())

		time.Sleep(100 * time.Millisecond)
	}
}
