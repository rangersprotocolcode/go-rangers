package middleware

import (
	"testing"
	"time"
	"fmt"
	"strconv"
)

func TestReentrantLock_Lock(t *testing.T) {
	lock := NewReentrantLock()

	fmt.Println(time.Now().String())

	for i := 0; i < 100; i++ {
		go method(lock, strconv.Itoa(i))
	}

	time.Sleep(1000 * time.Second)
	fmt.Println(time.Now().String())
}

func method(lock *ReentrantLock, name string) {
	for {
		lock.Lock(name)
		lock.Lock(name)
		fmt.Printf("%s locked\n", name)
		lock.Unlock(name)
		fmt.Printf("%s unlocked\n", name)

		time.Sleep(100 * time.Millisecond)
	}
}
