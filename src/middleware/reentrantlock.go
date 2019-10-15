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

// 获取资源锁
func (self *ReentrantLock) Lock(newOwner string) {
	// 已经获取到了，直接返回
	self.ownerLock.Lock()
	defer self.ownerLock.Unlock()

	if self.owner == newOwner {
		self.holdCount = 1
		return
	}
	for self.holdCount != 0 {
		self.cond.Wait()
	}
	self.owner = newOwner
	self.holdCount = 1

}

// owner释放资源
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
