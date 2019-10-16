package middleware

import (
	"testing"
	"time"
	"fmt"
	"strconv"
	"x/src/utility"
)

func TestReentrantLock_Lock(t *testing.T) {
	lock := NewReentrantLock()

	fmt.Println(time.Now().String())

	for i := 0; i < 10; i++ {
		go method(lock, strconv.Itoa(i))
	}

	time.Sleep(1000 * time.Second)
	fmt.Println(time.Now().String())
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
