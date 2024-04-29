package logical

import (
	"errors"
	"fmt"
	"runtime"
	"runtime/debug"
	"testing"
	"time"
)

func TestBaseParty_Cancel(t *testing.T) {
	fmt.Println("start")
	c := make(chan byte, 1)
	c <- 1
	fmt.Println("end")
}

func TestLeakOfMemory(t *testing.T) {
	fmt.Println("NumGoroutine:", runtime.NumGoroutine())
	chanLeakOfMemory()
	time.Sleep(time.Second * 3) // 等待 goroutine 执行，防止过早输出结果
	fmt.Println("NumGoroutine:", runtime.NumGoroutine())
}

func chanLeakOfMemory() {
	errCh := make(chan error, 1) // (1)
	go func() {                  // (5)
		time.Sleep(2 * time.Second)
		errCh <- errors.New("chan error") // (2)
		fmt.Println("finish sending")
	}()

	var err error
	select {
	case <-time.After(time.Second): // (3) 大家也经常在这里使用 <-ctx.Done()
		fmt.Println("超时")
	case err = <-errCh: // (4)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(nil)
		}
	}
}

func TestMap(t *testing.T) {
	key := "1"
	mapIns := make(map[string][]byte, 1)
	msgs, ok := mapIns[key]
	if !ok {
		msgs = make([]byte, 0)
	}

	msgs = append(msgs, 1)
	mapIns[key] = msgs

	fmt.Println(mapIns[key])
}

func TestPanic(t *testing.T) {

	go func() {
		defer func() {
			fmt.Println("end")
			if r := recover(); r != nil {
				fmt.Println(string(debug.Stack()))
			}
		}()
		fmt.Println("start")

		panic("test")
	}()

	time.Sleep(2 * time.Second)
	fmt.Println("main end")
}
