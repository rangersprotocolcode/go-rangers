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
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package middleware

import "sync"

type ReentrantLock struct {
	owner     string
	ownerLock *sync.Mutex
	cond      *sync.Cond
	holdCount int
}

func NewReentrantLock() *ReentrantLock {
	lock := &ReentrantLock{
		ownerLock: new(sync.Mutex),
	}

	lock.cond = sync.NewCond(lock.ownerLock)
	return lock
}

func (self *ReentrantLock) Lock(newOwner string) {
	self.ownerLock.Lock()
	defer self.ownerLock.Unlock()

	if self.owner == newOwner {
		self.holdCount++
		return
	}
	for self.holdCount != 0 {
		self.cond.Wait()
	}
	self.owner = newOwner
	self.holdCount = 1

}

func (self *ReentrantLock) Unlock(owner string) {
	self.ownerLock.Lock()
	defer self.ownerLock.Unlock()

	if self.holdCount == 0 || self.owner != owner {
		panic("illegalMonitorStateError")
	}

	self.holdCount--
	if self.holdCount == 0 {
		self.cond.Signal()
	}
}

func (self *ReentrantLock) Release(owner string) {
	self.ownerLock.Lock()
	defer self.ownerLock.Unlock()

	if self.holdCount == 0 || self.owner != owner {
		panic("illegalMonitorStateError")
	}

	self.holdCount = 0
	self.cond.Signal()
}
